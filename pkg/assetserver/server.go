package assetserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/paalgyula/summit/pkg/converter/adt"
	"github.com/paalgyula/summit/pkg/converter/blp"
	"github.com/paalgyula/summit/pkg/converter/m2"
	"github.com/paalgyula/summit/pkg/converter/mpq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	_ = mime.AddExtensionType(".glb", "model/gltf-binary")
	_ = mime.AddExtensionType(".gltf", "model/gltf+json")
	_ = mime.AddExtensionType(".webp", "image/webp")
	_ = mime.AddExtensionType(".bin", "application/octet-stream")
}

type Config struct {
	ListenAddr string
	AssetDir   string
	CacheDir   string
	MPQDir     string
	CORSOrigin string
}

type Server struct {
	cfg      Config
	log      zerolog.Logger
	echo     *echo.Echo
	mpqs     []*mpq.Archive
	mpqLock  sync.RWMutex
	inFlight sync.Map
}

// NewServer initializes the Echo asset server with standard middlewares.
func NewServer(cfg Config) (*Server, error) {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.AssetDir == "" {
		cfg.AssetDir = "./assets"
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = filepath.Join(cfg.AssetDir, ".cache")
	}
	if cfg.CORSOrigin == "" {
		cfg.CORSOrigin = "*"
	}

	_ = os.MkdirAll(cfg.AssetDir, 0755)
	_ = os.MkdirAll(cfg.CacheDir, 0755)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Standard Echo Middlewares
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:  []string{cfg.CORSOrigin},
		AllowMethods:  []string{http.MethodGet, http.MethodHead, http.MethodOptions},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Range", "Authorization"},
		ExposeHeaders: []string{"Content-Length", "Content-Range", "Accept-Ranges"},
	}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
		Skipper: func(c echo.Context) bool {
			// Skip images, audio, and compressed models that don't benefit from gzip
			path := strings.ToLower(c.Request().URL.Path)
			return strings.HasSuffix(path, ".webp") ||
				strings.HasSuffix(path, ".png") ||
				strings.HasSuffix(path, ".glb") ||
				strings.HasSuffix(path, ".mp3")
		},
	}))

	s := &Server{
		cfg:  cfg,
		log:  log.With().Str("service", "assetserver").Logger(),
		echo: e,
	}

	if cfg.MPQDir != "" {
		s.loadMPQs(cfg.MPQDir)
	}

	e.Match([]string{http.MethodGet, http.MethodHead}, "/health", s.handleHealth)
	e.Match([]string{http.MethodGet, http.MethodHead}, "/api/stats", s.handleStats)
	e.Match([]string{http.MethodGet, http.MethodHead}, "/*", s.handleAsset)

	return s, nil
}

// Echo returns the underlying Echo instance.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

func (s *Server) loadMPQs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		s.log.Warn().Err(err).Str("dir", dir).Msg("cannot read MPQ directory")
		return
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mpq") {
			fullPath := filepath.Join(dir, e.Name())
			arc, err := mpq.Open(fullPath)
			if err != nil {
				s.log.Warn().Err(err).Str("file", fullPath).Msg("failed to load MPQ archive")
				continue
			}
			s.mpqs = append(s.mpqs, arc)
			s.log.Info().Str("file", e.Name()).Int("indexedFiles", len(arc.ListFiles())).Msg("loaded MPQ archive")
		}
	}
}

// Start launches the Echo HTTP server.
func (s *Server) Start() error {
	s.log.Info().
		Str("addr", s.cfg.ListenAddr).
		Str("assetDir", s.cfg.AssetDir).
		Str("cacheDir", s.cfg.CacheDir).
		Msg("starting WoW asset server (Echo)")

	if err := s.echo.Start(s.cfg.ListenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the Echo server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStats(c echo.Context) error {
	s.mpqLock.RLock()
	defer s.mpqLock.RUnlock()

	return c.JSON(http.StatusOK, map[string]any{
		"mpq_count": len(s.mpqs),
		"asset_dir": s.cfg.AssetDir,
		"cache_dir": s.cfg.CacheDir,
	})
}

func (s *Server) handleAsset(c echo.Context) error {
	relPath := filepath.Clean(strings.TrimPrefix(c.Request().URL.Path, "/"))
	if relPath == "." || relPath == "/" {
		return echo.ErrNotFound
	}

	// Cache-Control headers for web assets
	c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Response().Header().Set("Accept-Ranges", "bytes")

	// 1. Check cache FIRST - if already converted file exists, serve immediately (NO JIT!)
	cachedPath := filepath.Join(s.cfg.CacheDir, relPath)
	if fi, err := os.Stat(cachedPath); err == nil && !fi.IsDir() {
		return c.File(cachedPath)
	}

	// 2. Check direct on disk in AssetDir
	diskPath := filepath.Join(s.cfg.AssetDir, relPath)
	if fi, err := os.Stat(diskPath); err == nil && !fi.IsDir() {
		return c.File(diskPath)
	}

	// Deduplicate in-flight conversions so concurrent requests don't duplicate work
	actualLock, _ := s.inFlight.LoadOrStore(relPath, &sync.Mutex{})
	mtx := actualLock.(*sync.Mutex)
	mtx.Lock()
	defer func() {
		mtx.Unlock()
		s.inFlight.Delete(relPath)
	}()

	// Double-check cache under lock in case another request just finished converting it
	if fi, err := os.Stat(cachedPath); err == nil && !fi.IsDir() {
		return c.File(cachedPath)
	}

	ext := strings.ToLower(filepath.Ext(relPath))

	// 3. JIT Transcode to WebP (and write to cache)
	if ext == ".webp" {
		if err := s.tryJITWebP(relPath, cachedPath); err == nil {
			return c.File(cachedPath)
		}
	}

	// 4. JIT Transcode to GLB (and write to cache)
	if ext == ".glb" {
		if err := s.tryJITGLB(relPath, cachedPath); err == nil {
			return c.File(cachedPath)
		}
	}

	// 5. Raw file from MPQs - cache it immediately so future requests never hit MPQs again
	if data, err := s.findInMPQs(relPath); err == nil {
		_ = os.MkdirAll(filepath.Dir(cachedPath), 0755)
		_ = os.WriteFile(cachedPath, data, 0644)
		s.log.Debug().Str("path", relPath).Msg("cached raw MPQ file")
		return c.File(cachedPath)
	}

	return echo.ErrNotFound
}

func (s *Server) tryJITWebP(relPath, cachedPath string) error {
	blpRelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".blp"

	var blpData []byte
	diskBlp := filepath.Join(s.cfg.AssetDir, blpRelPath)
	if data, err := os.ReadFile(diskBlp); err == nil {
		blpData = data
	} else if data, err := s.findInMPQs(blpRelPath); err == nil {
		blpData = data
	} else {
		return errors.New("source BLP not found")
	}

	img, err := blp.Decode(bytes.NewReader(blpData))
	if err != nil {
		s.log.Error().Err(err).Str("path", blpRelPath).Msg("failed to decode BLP for JIT WebP")
		return err
	}

	_ = os.MkdirAll(filepath.Dir(cachedPath), 0755)
	if err := blp.SaveWebP(img, cachedPath, 85.0, true); err != nil {
		s.log.Error().Err(err).Msg("failed to save JIT WebP to cache")
		return err
	}

	s.log.Debug().Str("path", relPath).Msg("JIT converted BLP to WebP (cached)")
	return nil
}

func (s *Server) tryJITGLB(relPath, cachedPath string) error {
	m2RelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".m2"

	var m2Data []byte
	diskM2 := filepath.Join(s.cfg.AssetDir, m2RelPath)
	if data, err := os.ReadFile(diskM2); err == nil {
		m2Data = data
	} else if data, err := s.findInMPQs(m2RelPath); err == nil {
		m2Data = data
	}

	if len(m2Data) > 0 {
		model, err := m2.Read(bytes.NewReader(m2Data))
		if err != nil {
			s.log.Error().Err(err).Str("path", m2RelPath).Msg("failed to read M2 for JIT GLB")
			return err
		}

		var skin *m2.Skin
		skinRelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + "00.skin"
		var skinData []byte
		diskSkin := filepath.Join(s.cfg.AssetDir, skinRelPath)
		if data, err := os.ReadFile(diskSkin); err == nil {
			skinData = data
		} else if data, err := s.findInMPQs(skinRelPath); err == nil {
			skinData = data
		}
		if len(skinData) > 0 {
			if s, err := m2.ReadSkin(bytes.NewReader(skinData)); err == nil {
				skin = s
			}
		}

		// Sequences stored outside the .m2 live in <Model><seq:04>-<var:02>.anim next to it
		modelBase := strings.TrimSuffix(relPath, filepath.Ext(relPath))
		animLoader := func(sequenceID, variation uint16) ([]byte, error) {
			return s.readSource(fmt.Sprintf("%s%04d-%02d.anim", modelBase, sequenceID, variation))
		}

		doc, err := m2.ConvertToGLTF(model, skin, m2.WithAnimLoader(animLoader))
		if err != nil {
			s.log.Error().Err(err).Msg("failed to convert M2 to glTF")
			return err
		}

		_ = os.MkdirAll(filepath.Dir(cachedPath), 0755)
		if err := doc.SaveGLB(cachedPath); err != nil {
			s.log.Error().Err(err).Msg("failed to save JIT GLB to cache")
			return err
		}

		s.log.Debug().Str("path", relPath).Msg("JIT converted M2 to GLB (cached)")
		return nil
	}

	// Check if source is ADT terrain
	adtRelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".adt"
	var adtData []byte
	diskAdt := filepath.Join(s.cfg.AssetDir, adtRelPath)
	if data, err := os.ReadFile(diskAdt); err == nil {
		adtData = data
	} else if data, err := s.findInMPQs(adtRelPath); err == nil {
		adtData = data
	}

	if len(adtData) > 0 {
		parsedAdt, err := adt.ReadADT(bytes.NewReader(adtData))
		if err != nil {
			s.log.Error().Err(err).Str("path", adtRelPath).Msg("failed to read ADT for JIT GLB")
			return err
		}

		_ = os.MkdirAll(filepath.Dir(cachedPath), 0755)
		f, err := os.Create(cachedPath)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := parsedAdt.ExportGLB(f); err != nil {
			s.log.Error().Err(err).Msg("failed to export ADT to GLB")
			return err
		}

		s.log.Debug().Str("path", relPath).Msg("JIT converted ADT to GLB (cached)")
		return nil
	}

	return errors.New("source model (M2 or ADT) not found")
}

// readSource returns a raw source file from the asset directory or the MPQs.
func (s *Server) readSource(relPath string) ([]byte, error) {
	if data, err := os.ReadFile(filepath.Join(s.cfg.AssetDir, relPath)); err == nil {
		return data, nil
	}
	return s.findInMPQs(relPath)
}

func (s *Server) findInMPQs(name string) ([]byte, error) {
	s.mpqLock.RLock()
	defer s.mpqLock.RUnlock()

	normalized := strings.ReplaceAll(name, "/", "\\")
	for _, arc := range s.mpqs {
		if arc.HasFile(normalized) {
			return arc.ReadFile(normalized)
		}
	}
	return nil, errors.New("file not found in MPQs")
}
