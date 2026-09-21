package assetserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/paalgyula/summit/pkg/converter/adt"
	"github.com/paalgyula/summit/pkg/converter/blp"
	"github.com/paalgyula/summit/pkg/converter/m2"
	"github.com/paalgyula/summit/pkg/converter/mpq"
	"github.com/paalgyula/summit/pkg/converter/wmo"
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
	// UpstreamURL is another asset server to fetch raw source files (ADT, M2,
	// WMO, BLP, ...) from when they are neither on disk nor in the MPQs, so a
	// dev stack can run without a local WoW client.
	UpstreamURL string
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
	s.cfg.UpstreamURL = strings.TrimRight(cfg.UpstreamURL, "/")

	e.Match([]string{http.MethodGet, http.MethodHead}, "/health", s.handleHealth)
	e.Match([]string{http.MethodGet, http.MethodHead}, "/api/stats", s.handleStats)
	e.Match([]string{http.MethodGet, http.MethodHead}, "/*", s.handleAsset)

	return s, nil
}

// Echo returns the underlying Echo instance.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// loadMPQs opens every archive of a WoW Data/ directory: the root MPQs and
// the ones in the locale folders (Data/enUS/...), which hold the Interface\
// Glues scenes among others. Archives are kept in the client's load order,
// so a lookup walks them backwards and patches override the base data.
func (s *Server) loadMPQs(dir string) {
	var paths []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		s.log.Warn().Err(err).Str("dir", dir).Msg("cannot read MPQ directory")
		return
	}
	for _, e := range entries {
		name := e.Name()
		switch {
		case !e.IsDir() && strings.HasSuffix(strings.ToLower(name), ".mpq"):
			paths = append(paths, filepath.Join(dir, name))
		case e.IsDir() && isLocaleDir(name):
			sub, err := os.ReadDir(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			for _, f := range sub {
				if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".mpq") {
					paths = append(paths, filepath.Join(dir, name, f.Name()))
				}
			}
		}
	}

	sort.SliceStable(paths, func(i, j int) bool {
		ri, rj := mpqRank(paths[i]), mpqRank(paths[j])
		if ri != rj {
			return ri < rj
		}
		ni, nj := mpqNumber(paths[i]), mpqNumber(paths[j])
		if ni != nj {
			return ni < nj
		}
		return strings.ToLower(filepath.Base(paths[i])) < strings.ToLower(filepath.Base(paths[j]))
	})

	for _, fullPath := range paths {
		base := strings.ToLower(filepath.Base(fullPath))
		if strings.HasPrefix(base, "backup-") {
			continue // the client's uninstall backup of the locale data
		}
		arc, err := mpq.Open(fullPath)
		if err != nil {
			s.log.Warn().Err(err).Str("file", fullPath).Msg("failed to load MPQ archive")
			continue
		}
		s.mpqs = append(s.mpqs, arc)
		s.log.Info().Str("file", strings.TrimPrefix(fullPath, dir+string(filepath.Separator))).Int("indexedFiles", len(arc.ListFiles())).Msg("loaded MPQ archive")
	}
}

// isLocaleDir matches the client's locale folders (enUS, deDE, ...).
func isLocaleDir(name string) bool {
	if len(name) != 4 {
		return false
	}
	return strings.ToLower(name[:2]) == name[:2] && strings.ToUpper(name[2:]) == name[2:]
}

// mpqRank orders the archives the way the client loads them: base data,
// locale data, patches, locale patches. Later archives take precedence.
func mpqRank(path string) int {
	base := strings.ToLower(filepath.Base(path))
	locale := isLocaleDir(filepath.Base(filepath.Dir(path)))
	patch := strings.HasPrefix(base, "patch")
	switch {
	case !patch && !locale:
		return 0
	case !patch && locale:
		return 1
	case patch && !locale:
		return 2
	default:
		return 3
	}
}

// mpqNumber extracts the trailing number of patch-2.MPQ / patch-enUS-3.MPQ (1 when absent).
func mpqNumber(path string) int {
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(path)), ".mpq")
	if i := strings.LastIndex(base, "-"); i >= 0 {
		if n, err := strconv.Atoi(base[i+1:]); err == nil {
			return n
		}
	}
	return 1
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

	// 5. Raw file from MPQs (or upstream) - cache it so future requests never hit them again
	if data, err := s.readSource(relPath); err == nil {
		if err := writeCacheFile(cachedPath, func(f io.Writer) error {
			_, err := f.Write(data)
			return err
		}); err != nil {
			return err
		}
		s.log.Debug().Str("path", relPath).Msg("cached raw MPQ file")
		return c.File(cachedPath)
	}

	return echo.ErrNotFound
}

func (s *Server) tryJITWebP(relPath, cachedPath string) error {
	blpRelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".blp"

	blpData, err := s.readSource(blpRelPath)
	if err != nil {
		return errors.New("source BLP not found")
	}

	img, err := blp.Decode(bytes.NewReader(blpData))
	if err != nil {
		s.log.Error().Err(err).Str("path", blpRelPath).Msg("failed to decode BLP for JIT WebP")
		return err
	}

	if err := writeCacheFile(cachedPath, func(f io.Writer) error {
		return blp.ToWebP(img, f, 85.0, true)
	}); err != nil {
		s.log.Error().Err(err).Msg("failed to save JIT WebP to cache")
		return err
	}

	s.log.Debug().Str("path", relPath).Msg("JIT converted BLP to WebP (cached)")
	return nil
}

func (s *Server) tryJITGLB(relPath, cachedPath string) error {
	m2RelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".m2"

	m2Data, _ := s.readSource(m2RelPath)

	if len(m2Data) > 0 {
		model, err := m2.Read(bytes.NewReader(m2Data))
		if err != nil {
			s.log.Error().Err(err).Str("path", m2RelPath).Msg("failed to read M2 for JIT GLB")
			return err
		}

		var skin *m2.Skin
		skinRelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + "00.skin"
		skinData, _ := s.readSource(skinRelPath)
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

		if err := writeCacheFile(cachedPath, doc.ToGLB); err != nil {
			s.log.Error().Err(err).Msg("failed to save JIT GLB to cache")
			return err
		}

		s.log.Debug().Str("path", relPath).Msg("JIT converted M2 to GLB (cached)")
		return nil
	}

	// World map objects: a root .wmo plus one _NNN.wmo file per group
	wmoRelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".wmo"
	if wmoData, err := s.readSource(wmoRelPath); err == nil {
		root, err := wmo.ReadRoot(bytes.NewReader(wmoData))
		if err != nil {
			s.log.Error().Err(err).Str("path", wmoRelPath).Msg("failed to read WMO for JIT GLB")
			return err
		}
		for g := 0; g < int(root.Header.NGroups); g++ {
			groupPath := wmo.GroupFileName(wmoRelPath, g)
			groupData, err := s.readSource(groupPath)
			if err != nil {
				s.log.Warn().Str("path", groupPath).Msg("WMO group file missing")
				root.Groups = append(root.Groups, nil)
				continue
			}
			grp, err := wmo.ReadGroup(bytes.NewReader(groupData))
			if err != nil {
				s.log.Warn().Err(err).Str("path", groupPath).Msg("failed to read WMO group")
				grp = nil
			}
			root.Groups = append(root.Groups, grp)
		}

		if err := writeCacheFile(cachedPath, root.ExportGLB); err != nil {
			s.log.Error().Err(err).Msg("failed to export WMO to GLB")
			return err
		}
		s.log.Debug().Str("path", relPath).Int("groups", len(root.Groups)).Msg("JIT converted WMO to GLB (cached)")
		return nil
	}

	// Check if source is ADT terrain
	adtRelPath := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".adt"
	adtData, _ := s.readSource(adtRelPath)

	if len(adtData) > 0 {
		parsedAdt, err := adt.ReadADT(bytes.NewReader(adtData))
		if err != nil {
			s.log.Error().Err(err).Str("path", adtRelPath).Msg("failed to read ADT for JIT GLB")
			return err
		}

		if err := writeCacheFile(cachedPath, parsedAdt.ExportGLB); err != nil {
			s.log.Error().Err(err).Msg("failed to export ADT to GLB")
			return err
		}

		s.log.Debug().Str("path", relPath).Msg("JIT converted ADT to GLB (cached)")
		return nil
	}

	return errors.New("source model (M2, WMO or ADT) not found")
}

// writeCacheFile writes a converted asset atomically: into a temporary file
// next to the target, renamed into place once complete. Concurrent requests
// check the cache before taking the conversion lock, so a partially written
// file must never be visible under its final name.
func writeCacheFile(cachedPath string, write func(w io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(cachedPath), filepath.Base(cachedPath)+".*.part")
	if err != nil {
		return err
	}
	if err := write(tmp); err != nil {
		tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), cachedPath)
}

// readSource returns a raw source file from the asset directory, the MPQs
// or, failing both, the upstream asset server (cached on disk from then on).
func (s *Server) readSource(relPath string) ([]byte, error) {
	if data, err := os.ReadFile(filepath.Join(s.cfg.AssetDir, relPath)); err == nil {
		return data, nil
	}
	if data, err := s.findInMPQs(relPath); err == nil {
		return data, nil
	}
	return s.fetchUpstream(relPath)
}

// fetchUpstream downloads a raw source file from the upstream asset server
// into the asset directory.
func (s *Server) fetchUpstream(relPath string) ([]byte, error) {
	if s.cfg.UpstreamURL == "" {
		return nil, errors.New("file not found")
	}
	if strings.Contains(relPath, "..") {
		return nil, errors.New("invalid path")
	}

	url := s.cfg.UpstreamURL + "/" + strings.ReplaceAll(relPath, "\\", "/")
	resp, err := http.Get(url) //nolint:gosec // the upstream is operator configured
	if err != nil {
		return nil, fmt.Errorf("upstream %s: %w", relPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream %s: %s", relPath, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("upstream %s: %w", relPath, err)
	}

	diskPath := filepath.Join(s.cfg.AssetDir, filepath.FromSlash(strings.ReplaceAll(relPath, "\\", "/")))
	_ = os.MkdirAll(filepath.Dir(diskPath), 0755)
	_ = os.WriteFile(diskPath, data, 0644)
	s.log.Debug().Str("path", relPath).Int("bytes", len(data)).Msg("fetched source from upstream")
	return data, nil
}

func (s *Server) findInMPQs(name string) ([]byte, error) {
	s.mpqLock.RLock()
	defer s.mpqLock.RUnlock()

	normalized := strings.ReplaceAll(name, "/", "\\")
	// Last loaded wins: patches override the base archives
	for i := len(s.mpqs) - 1; i >= 0; i-- {
		if s.mpqs[i].HasFile(normalized) {
			return s.mpqs[i].ReadFile(normalized)
		}
	}
	return nil, errors.New("file not found in MPQs")
}
