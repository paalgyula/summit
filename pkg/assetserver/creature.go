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

// CreatureDisplayPrefix is the route of creature display manifests: creature/<displayId>.json.
const CreatureDisplayPrefix = "creature/"

// CreatureDisplay tells the client how to draw a CreatureDisplayInfo id.
type CreatureDisplay struct {
	DisplayID uint32 `json:"displayId"`
	// Model is the asset path of the M2 without extension ("Creature/Boar/Boar"): append .glb.
	Model string `json:"model"`
	// Scale multiplies the model: CreatureDisplayInfo scale times CreatureModelData scale.
	Scale float32 `json:"scale"`
	// Textures are the asset paths (without extension) bound to the M2 texture
	// types 11, 12 and 13 (monster skin 1-3); empty when the model has none.
	Textures [3]string `json:"textures"`
	// Extra is set for humanoid NPCs drawn with a player race model.
	Extra *CreatureDisplayExtra `json:"extra,omitempty"`
}

// CreatureDisplayExtra is a CreatureDisplayInfoExtra row: a player model with a look.
type CreatureDisplayExtra struct {
	Race       uint32 `json:"race"`
	Gender     uint32 `json:"gender"`
	Skin       uint32 `json:"skin"`
	Face       uint32 `json:"face"`
	HairStyle  uint32 `json:"hairStyle"`
	HairColor  uint32 `json:"hairColor"`
	FacialHair uint32 `json:"facialHair"`
	// Items are item display ids: helm, shoulders, shirt, chest, belt, legs, boots, wrists, gloves, tabard, cape.
	Items [11]uint32 `json:"items"`
	// BakedTexture is the pre-composed body texture (Textures/BakedNpcTextures/<name>), without extension.
	BakedTexture string `json:"bakedTexture,omitempty"`
}

// creatureTables holds the parsed creature DBCs, loaded on first use.
type creatureTables struct {
	mu     sync.Mutex
	loaded bool
	err    error

	displays map[uint32]creatureDisplayRow
	models   map[uint32]creatureModelRow
	extras   map[uint32]CreatureDisplayExtra
}

type creatureDisplayRow struct {
	modelID  uint32
	extraID  uint32
	scale    float32
	textures [3]string
}

type creatureModelRow struct {
	path  string
	scale float32
}

// CreatureDisplay resolves one display id from the DBCs.
func (s *Server) CreatureDisplay(displayID uint32) (*CreatureDisplay, error) {
	t := &s.creatures
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
		return nil, fmt.Errorf("creature display %d: not in CreatureDisplayInfo", displayID)
	}

	scale := row.scale
	if scale <= 0 {
		scale = 1
	}

	var modelPath string
	if model, ok := t.models[row.modelID]; ok {
		modelPath = model.path
		if model.scale > 0 {
			scale *= model.scale
		}
	} else if row.extraID == 0 {
		return nil, fmt.Errorf("creature display %d: model %d not in CreatureModelData", displayID, row.modelID)
	}

	out := &CreatureDisplay{
		DisplayID: displayID,
		Model:     modelPath,
		Scale:     scale,
	}

	dir := path.Dir(modelPath)

	for i, rawTex := range row.textures {
		if rawTex == "" {
			continue
		}
		tex := strings.ReplaceAll(rawTex, "\\", "/")
		tex = strings.TrimRight(tex, ".")
		tex = strings.TrimSuffix(tex, ".blp")
		tex = strings.TrimSuffix(tex, ".BLP")
		if strings.Contains(tex, "/") || dir == "." || dir == "" {
			out.Textures[i] = tex
		} else {
			out.Textures[i] = dir + "/" + tex
		}
	}

	if row.extraID != 0 {
		if extra, ok := t.extras[row.extraID]; ok {
			e := extra
			out.Extra = &e
		}
	}

	return out, nil
}

// load parses CreatureDisplayInfo, CreatureModelData and optionally CreatureDisplayInfoExtra (3.3.5a layouts).
func (t *creatureTables) load(s *Server) error {
	read := func(name string) (*dbc.File, error) {
		data, err := s.readDBC(name)
		if err != nil {
			return nil, fmt.Errorf("%s.dbc: %w", name, err)
		}

		f, err := dbc.Read(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%s.dbc: %w", name, err)
		}

		return f, nil
	}

	cdi, err := read("CreatureDisplayInfo")
	if err != nil {
		return err
	}

	cmd, err := read("CreatureModelData")
	if err != nil {
		return err
	}

	// CreatureDisplayInfoExtra is optional; non-humanoid displays can still work without it
	cde, errExtra := read("CreatureDisplayInfoExtra")
	if errExtra != nil {
		s.log.Warn().Err(errExtra).Msg("CreatureDisplayInfoExtra.dbc unavailable, humanoid displays will have limited appearance")
	}

	t.displays = make(map[uint32]creatureDisplayRow, cdi.Records)
	for r := 0; r < cdi.Records; r++ {
		t.displays[cdi.Uint32(r, 0)] = creatureDisplayRow{
			modelID:  cdi.Uint32(r, 1),
			extraID:  cdi.Uint32(r, 3),
			scale:    cdi.Float32(r, 4),
			textures: [3]string{cdi.String(r, 6), cdi.String(r, 7), cdi.String(r, 8)},
		}
	}

	t.models = make(map[uint32]creatureModelRow, cmd.Records)
	for r := 0; r < cmd.Records; r++ {
		t.models[cmd.Uint32(r, 0)] = creatureModelRow{
			path:  modelAssetPath(cmd.String(r, 2)),
			scale: cmd.Float32(r, 4),
		}
	}

	if cde != nil {
		t.extras = make(map[uint32]CreatureDisplayExtra, cde.Records)
		for r := 0; r < cde.Records; r++ {
			e := CreatureDisplayExtra{
				Race:       cde.Uint32(r, 1),
				Gender:     cde.Uint32(r, 2),
				Skin:       cde.Uint32(r, 3),
				Face:       cde.Uint32(r, 4),
				HairStyle:  cde.Uint32(r, 5),
				HairColor:  cde.Uint32(r, 6),
				FacialHair: cde.Uint32(r, 7),
			}

			for i := range e.Items {
				e.Items[i] = cde.Uint32(r, 8+i)
			}

			if bake := cde.String(r, 20); bake != "" {
				e.BakedTexture = "Textures/BakedNpcTextures/" + strings.TrimSuffix(bake, path.Ext(bake))
			}

			t.extras[cde.Uint32(r, 0)] = e
		}
	}

	return nil
}

// modelAssetPath turns a DBC model path (Creature\Boar\Boar.mdx) into the
// asset path the M2 is served under, without extension.
func modelAssetPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")

	return strings.TrimSuffix(p, path.Ext(p))
}

// creatureDisplayID parses creature/<id>.json; ok is false for any other path.
// It accepts variations like /creature/123.json, /Creature/123.json, /creatures/123, api/creature/123.json.
func creatureDisplayID(relPath string) (uint32, bool) {
	norm := filepath.ToSlash(strings.ToLower(relPath))
	norm = strings.TrimPrefix(norm, "/")
	norm = strings.TrimPrefix(norm, "api/")
	norm = strings.TrimPrefix(norm, "assets/")
	norm = strings.TrimPrefix(norm, "/")

	var idStr string
	if strings.HasPrefix(norm, "creature/") {
		idStr = strings.TrimPrefix(norm, "creature/")
	} else if strings.HasPrefix(norm, "creatures/") {
		idStr = strings.TrimPrefix(norm, "creatures/")
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

// tryJITCreatureDisplay writes the manifest of a display id into the cache.
func (s *Server) tryJITCreatureDisplay(displayID uint32, cachedPath string) error {
	display, err := s.CreatureDisplay(displayID)
	if err != nil {
		return err
	}

	return writeCacheFile(cachedPath, func(f io.Writer) error {
		return json.NewEncoder(f).Encode(display)
	})
}
