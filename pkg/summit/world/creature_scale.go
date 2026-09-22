package world

import (
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc"
	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/rs/zerolog/log"
)

// creatureScaleCache lazily loads CreatureDisplayInfo.dbc and CreatureModelData.dbc
// to compute the final display scale for creatures:
//
//	Final Scale = template.Scale * CreatureDisplayInfo.Scale * CreatureModelData.Scale
//
// This matches AzerothCore's Creature::GetNativeObjectScale() behaviour.
type creatureScaleCache struct {
	mu sync.RWMutex

	displays map[uint32]wotlk.CreatureDisplayInfoEntry // displayID -> display info
	models   map[uint32]wotlk.CreatureModelDataEntry   // modelID   -> model data

	loaded bool
}

//nolint:gochecknoglobals
var creatureScales = &creatureScaleCache{
	displays: make(map[uint32]wotlk.CreatureDisplayInfoEntry),
	models:   make(map[uint32]wotlk.CreatureModelDataEntry),
}

// LoadCreatureScaleDBC loads CreatureDisplayInfo.dbc and CreatureModelData.dbc
// from the given directory. Safe to call multiple times; only the first call loads.
func LoadCreatureScaleDBC(dbcPath string) error {
	creatureScales.mu.Lock()
	defer creatureScales.mu.Unlock()

	if creatureScales.loaded {
		return nil
	}

	start := time.Now()

	// Load CreatureDisplayInfo.dbc
	displays, err := dbc.Load[wotlk.CreatureDisplayInfoEntry]("CreatureDisplayInfo.dbc", dbcPath)
	if err != nil {
		log.Warn().Err(err).Msg("creature scale: CreatureDisplayInfo.dbc unavailable, using template scale only")
		// Not fatal — fall back to template scale
		creatureScales.loaded = true
		return nil
	}

	for _, d := range displays {
		if d.ID == 0 {
			continue
		}

		creatureScales.displays[d.ID] = d
	}

	// Load CreatureModelData.dbc
	models, err := dbc.Load[wotlk.CreatureModelDataEntry]("CreatureModelData.dbc", dbcPath)
	if err != nil {
		log.Warn().Err(err).Msg("creature scale: CreatureModelData.dbc unavailable, using display scale only")
		creatureScales.loaded = true
		return nil
	}

	for _, m := range models {
		if m.ID == 0 {
			continue
		}

		creatureScales.models[m.ID] = m
	}

	creatureScales.loaded = true

	log.Info().
		Int("displays", len(creatureScales.displays)).
		Int("models", len(creatureScales.models)).
		Msgf("creature scale cache loaded in %s", time.Since(start).String())

	return nil
}

// ComputeCreatureScale returns the final display scale for a creature:
//
//	templateScale * CreatureDisplayInfo.Scale * CreatureModelData.Scale
//
// When DBC data is unavailable or displayID is 0, returns the template
// scale (defaulting to 1.0 when <= 0). This matches AzerothCore's
// ObjectMgr::ChooseDisplayId + CreatureDisplayInfo multiply chain.
func ComputeCreatureScale(templateScale float32, displayID uint32) float32 {
	scale := templateScale
	if scale <= 0 {
		scale = 1.0
	}

	if displayID == 0 {
		return scale
	}

	creatureScales.mu.RLock()
	defer creatureScales.mu.RUnlock()

	if len(creatureScales.displays) == 0 {
		return scale
	}

	di, ok := creatureScales.displays[displayID]
	if !ok {
		return scale
	}

	// CreatureDisplayInfo.Scale: per-display multiplier (often 1.0)
	if di.Scale > 0 {
		scale *= di.Scale
	}

	// CreatureModelData.Scale: per-model geometry multiplier
	if mi, ok := creatureScales.models[di.ModelID]; ok && mi.Scale > 0 {
		scale *= mi.Scale
	}

	return scale
}
