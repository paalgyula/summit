package world

import (
	"math"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
)

func approxEqual(a, b float32) bool {
	return math.Abs(float64(a)-float64(b)) < 1e-6
}

// TestComputeCreatureScale_DBCData verifies the scale chain:
//
//	template.Scale × CreatureDisplayInfo.Scale × CreatureModelData.Scale
func TestComputeCreatureScale_DBCData(t *testing.T) {
	// Seed the cache directly (avoids needing DBC files on disk).
	creatureScales.mu.Lock()
	creatureScales.displays = map[uint32]wotlk.CreatureDisplayInfoEntry{
		1234: {ID: 1234, ModelID: 5678, Scale: 1.5},
	}
	creatureScales.models = map[uint32]wotlk.CreatureModelDataEntry{
		5678: {ID: 5678, Scale: 2.0},
	}
	creatureScales.loaded = true
	creatureScales.mu.Unlock()

	// template 1.0 × display 1.5 × model 2.0 = 3.0
	got := ComputeCreatureScale(1.0, 1234)
	if !approxEqual(got, 3.0) {
		t.Errorf("want 3.0, got %f", got)
	}
}

// TestComputeCreatureScale_ZeroTemplate verifies template scale 0 defaults to 1.0.
func TestComputeCreatureScale_ZeroTemplate(t *testing.T) {
	creatureScales.mu.Lock()
	creatureScales.displays = map[uint32]wotlk.CreatureDisplayInfoEntry{
		100: {ID: 100, ModelID: 200, Scale: 0.8},
	}
	creatureScales.models = map[uint32]wotlk.CreatureModelDataEntry{
		200: {ID: 200, Scale: 1.2},
	}
	creatureScales.loaded = true
	creatureScales.mu.Unlock()

	// template 0 → default 1.0 × display 0.8 × model 1.2 = 0.96
	got := ComputeCreatureScale(0, 100)
	if !approxEqual(got, 0.96) {
		t.Errorf("want 0.96, got %f", got)
	}
}

// TestComputeCreatureScale_NoDisplayID verifies fallback to template scale.
func TestComputeCreatureScale_NoDisplayID(t *testing.T) {
	creatureScales.mu.Lock()
	creatureScales.displays = map[uint32]wotlk.CreatureDisplayInfoEntry{}
	creatureScales.models = map[uint32]wotlk.CreatureModelDataEntry{}
	creatureScales.loaded = true
	creatureScales.mu.Unlock()

	// displayID 0 → just template scale
	got := ComputeCreatureScale(2.5, 0)
	if !approxEqual(got, 2.5) {
		t.Errorf("want 2.5, got %f", got)
	}
}

// TestComputeCreatureScale_EmptyCache verifies the cache is usable before
// loading (graceful degradation).
func TestComputeCreatureScale_EmptyCache(t *testing.T) {
	creatureScales.mu.Lock()
	creatureScales.displays = map[uint32]wotlk.CreatureDisplayInfoEntry{}
	creatureScales.models = map[uint32]wotlk.CreatureModelDataEntry{}
	creatureScales.loaded = true
	creatureScales.mu.Unlock()

	got := ComputeCreatureScale(1.5, 999)
	if !approxEqual(got, 1.5) {
		t.Errorf("want 1.5, got %f", got)
	}
}

// TestComputeCreatureScale_DisplayOnly verifies that a display entry with
// no matching model still applies the display scale.
func TestComputeCreatureScale_DisplayOnly(t *testing.T) {
	creatureScales.mu.Lock()
	creatureScales.displays = map[uint32]wotlk.CreatureDisplayInfoEntry{
		500: {ID: 500, ModelID: 600, Scale: 3.0},
	}
	creatureScales.models = map[uint32]wotlk.CreatureModelDataEntry{}
	creatureScales.loaded = true
	creatureScales.mu.Unlock()

	// template 1.0 × display 3.0 = 3.0 (model missing, skipped)
	got := ComputeCreatureScale(1.0, 500)
	if !approxEqual(got, 3.0) {
		t.Errorf("want 3.0, got %f", got)
	}
}

// TestComputeCreatureScale_UnknownDisplayID verifies fallback for unknown display.
func TestComputeCreatureScale_UnknownDisplayID(t *testing.T) {
	creatureScales.mu.Lock()
	creatureScales.displays = map[uint32]wotlk.CreatureDisplayInfoEntry{
		100: {ID: 100, ModelID: 200, Scale: 1.0},
	}
	creatureScales.models = map[uint32]wotlk.CreatureModelDataEntry{
		200: {ID: 200, Scale: 1.0},
	}
	creatureScales.loaded = true
	creatureScales.mu.Unlock()

	// Unknown displayID → just template scale
	got := ComputeCreatureScale(2.0, 999)
	if !approxEqual(got, 2.0) {
		t.Errorf("want 2.0, got %f", got)
	}
}

// TestComputeCreatureScale_NegativeTemplate verifies negative template scale
// defaults to 1.0.
func TestComputeCreatureScale_NegativeTemplate(t *testing.T) {
	creatureScales.mu.Lock()
	creatureScales.displays = map[uint32]wotlk.CreatureDisplayInfoEntry{
		10: {ID: 10, ModelID: 20, Scale: 1.0},
	}
	creatureScales.models = map[uint32]wotlk.CreatureModelDataEntry{
		20: {ID: 20, Scale: 1.0},
	}
	creatureScales.loaded = true
	creatureScales.mu.Unlock()

	// Negative template → default 1.0
	got := ComputeCreatureScale(-1.0, 10)
	if !approxEqual(got, 1.0) {
		t.Errorf("want 1.0, got %f", got)
	}
}
