package assetserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/paalgyula/summit/pkg/converter/dbc"
)

// GameObjectDisplayPrefix is the route of game object display manifests: gameobject/<displayId>.json.
const GameObjectDisplayPrefix = "gameobject/"

// GameObjectDisplay tells the client how to draw a GameObjectDisplayInfo id.
type GameObjectDisplay struct {
	DisplayID uint32  `json:"displayId"`
	Model     string  `json:"model"`
	Scale     float32 `json:"scale"`
}

// goDisplayTables caches the parsed GameObjectDisplayInfo DBC.
type goDisplayTables struct {
	mu     sync.Mutex
	loaded bool
	err    error

	displays map[uint32]goDisplayRow
}

type goDisplayRow struct {
	modelPath string
}

// GameObjectDisplay resolves one display id from GameObjectDisplayInfo.dbc.
func (s *Server) GameObjectDisplay(displayID uint32) (*GameObjectDisplay, error) {
	t := &s.goDisplays
	t.mu.Lock()
	if !t.loaded {
		t.err = t.load(s)
		if t.err == nil {
			t.loaded = true
		}
	}
	err := t.err
	t.mu.Unlock()

	if err != nil {
		return nil, err
	}

	row, ok := t.displays[displayID]
	if !ok {
		return nil, fmt.Errorf("gameobject display %d: not in GameObjectDisplayInfo", displayID)
	}

	return &GameObjectDisplay{
		DisplayID: displayID,
		Model:     row.modelPath,
		Scale:     1, // GOs don't have a separate scale in the DBC; the model's own scale is used
	}, nil
}

// load parses GameObjectDisplayInfo.dbc (3.3.5a layout).
func (t *goDisplayTables) load(s *Server) error {
	data, err := s.readDBC("GameObjectDisplayInfo")
	if err != nil {
		return fmt.Errorf("GameObjectDisplayInfo.dbc: %w", err)
	}

	f, err := dbc.Read(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("GameObjectDisplayInfo.dbc: %w", err)
	}

	t.displays = make(map[uint32]goDisplayRow, f.Records)
	for r := 0; r < f.Records; r++ {
		id := f.Uint32(r, 0)
		modelRaw := f.String(r, 1) // Field1: model path string ref
		if modelRaw == "" {
			continue
		}
		// Normalize path: backslash → forward slash, strip extension
		model := strings.ReplaceAll(modelRaw, "\\", "/")
		model = strings.TrimSuffix(model, path.Ext(model))
		t.displays[id] = goDisplayRow{
			modelPath: model,
		}
	}

	return nil
}

// gameObjectDisplayID parses gameobject/<id>.json; ok is false for any other path.
func gameObjectDisplayID(relPath string) (uint32, bool) {
	norm := filepath.ToSlash(strings.ToLower(relPath))
	norm = strings.TrimPrefix(norm, "/")
	norm = strings.TrimPrefix(norm, "api/")
	norm = strings.TrimPrefix(norm, "assets/")
	norm = strings.TrimPrefix(norm, "/")

	var idStr string
	if strings.HasPrefix(norm, "gameobject/") {
		idStr = strings.TrimPrefix(norm, "gameobject/")
	} else if strings.HasPrefix(norm, "gameobjects/") {
		idStr = strings.TrimPrefix(norm, "gameobjects/")
	} else {
		return 0, false
	}

	idStr = strings.TrimSuffix(idStr, ".json")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, false
	}

	return uint32(id), true
}

// tryJITGameObjectDisplay writes the manifest of a display id into the cache.
func (s *Server) tryJITGameObjectDisplay(displayID uint32, cachedPath string) error {
	display, err := s.GameObjectDisplay(displayID)
	if err != nil {
		return err
	}

	return writeCacheFile(cachedPath, func(f io.Writer) error {
		return json.NewEncoder(f).Encode(display)
	})
}
