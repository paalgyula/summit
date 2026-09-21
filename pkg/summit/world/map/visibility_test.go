package mapmanager

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// VisibilityDistanceType — mirrors AzerothCore's ObjectDefines.h enum
// ---------------------------------------------------------------------------

func TestVisibilityDistanceTypeConstants(t *testing.T) {
	assert.Equal(t, object.VisibilityDistanceType(0), object.VisibilityDistanceNormal)
	assert.Equal(t, object.VisibilityDistanceType(1), object.VisibilityDistanceTiny)
	assert.Equal(t, object.VisibilityDistanceType(2), object.VisibilityDistanceSmall)
	assert.Equal(t, object.VisibilityDistanceType(3), object.VisibilityDistanceLarge)
	assert.Equal(t, object.VisibilityDistanceType(4), object.VisibilityDistanceGigantic)
	assert.Equal(t, object.VisibilityDistanceType(5), object.VisibilityDistanceInfinite)
	assert.Equal(t, object.VisibilityDistanceType(6), object.VisibilityDistanceMax)
}

func TestVisibilityDistanceValues(t *testing.T) {
	// Values must match AzerothCore's ObjectDefines.h
	assert.InDelta(t, 100.0, object.VisibilityDistances[object.VisibilityDistanceNormal], 0.1)
	assert.InDelta(t, 25.0, object.VisibilityDistances[object.VisibilityDistanceTiny], 0.1)
	assert.InDelta(t, 50.0, object.VisibilityDistances[object.VisibilityDistanceSmall], 0.1)
	assert.InDelta(t, 200.0, object.VisibilityDistances[object.VisibilityDistanceLarge], 0.1)
	assert.InDelta(t, 400.0, object.VisibilityDistances[object.VisibilityDistanceGigantic], 0.1)
	assert.InDelta(t, 533.0, object.VisibilityDistances[object.VisibilityDistanceInfinite], 0.1)
}

func TestIsFarVisible(t *testing.T) {
	assert.False(t, object.VisibilityDistanceNormal.IsFarVisible())
	assert.False(t, object.VisibilityDistanceTiny.IsFarVisible())
	assert.False(t, object.VisibilityDistanceSmall.IsFarVisible())
	assert.True(t, object.VisibilityDistanceLarge.IsFarVisible())
	assert.True(t, object.VisibilityDistanceGigantic.IsFarVisible())
	assert.False(t, object.VisibilityDistanceInfinite.IsFarVisible())
}

// ---------------------------------------------------------------------------
// Per-map visibility overrides — mirrors Map::InitVisibilityDistance
// ---------------------------------------------------------------------------

func TestInitVisibilityDistance_Continent(t *testing.T) {
	m := NewMap(0, 0, nil) // map 0 = Eastern Kingdoms
	m.InitVisibilityDistance()
	assert.InDelta(t, 100.0, m.visibilityRange, 0.1, "default continent should be 100y")
}

func TestInitVisibilityDistance_EbonHold(t *testing.T) {
	m := NewMap(609, 0, nil) // Scarlet Enclave
	m.InitVisibilityDistance()
	assert.InDelta(t, 125.0, m.visibilityRange, 0.1, "Ebon Hold should be 125y")
}

func TestInitVisibilityDistance_Dungeon(t *testing.T) {
	entry := &MapEntry{InstanceType: 1} // INSTANCE_MULTIMAP
	m := NewMap(389, 0, entry)          // Ragefire Chasm
	m.InitVisibilityDistance()
	assert.InDelta(t, 170.0, m.visibilityRange, 0.1, "dungeon should be 170y")
}

func TestInitVisibilityDistance_Battleground(t *testing.T) {
	entry := &MapEntry{InstanceType: 3} // INSTANCE_BATTLEGROUND
	m := NewMap(489, 0, entry)          // Warsong Gulch
	m.InitVisibilityDistance()
	assert.InDelta(t, 250.0, m.visibilityRange, 0.1, "BG should be 250y")
}

func TestInitVisibilityDistance_Arena(t *testing.T) {
	entry := &MapEntry{InstanceType: 2} // INSTANCE_ARENA
	m := NewMap(559, 0, entry)          // Nagrand Arena
	m.InitVisibilityDistance()
	assert.InDelta(t, 250.0, m.visibilityRange, 0.1, "arena should be 250y")
}

func TestInitVisibilityDistance_Raid(t *testing.T) {
	entry := &MapEntry{InstanceType: 4} // INSTANCE_RAID
	m := NewMap(531, 0, entry)          // Hyjal Summit
	m.InitVisibilityDistance()
	// Raids use default continent range (100y) unless overridden
	assert.InDelta(t, 100.0, m.visibilityRange, 0.1, "raid should use default")
}

// ---------------------------------------------------------------------------
// GetVisibilityRange — per-object override
// ---------------------------------------------------------------------------

func TestGetVisibilityRange_DefaultFromMap(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.visibilityRange = 100.0

	obj := object.NewObject()
	r := GetVisibilityRange(obj, m)
	assert.InDelta(t, 100.0, r, 0.1)
}

func TestGetVisibilityRange_MapOverride(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.visibilityRange = 170.0 // dungeon

	obj := object.NewObject()
	r := GetVisibilityRange(obj, m)
	assert.InDelta(t, 170.0, r, 0.1)
}

func TestGetVisibilityRange_ObjectOverride(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.visibilityRange = 100.0

	obj := object.NewObject()
	obj.SetVisibilityOverrideType(object.VisibilityDistanceLarge)
	r := GetVisibilityRange(obj, m)
	assert.InDelta(t, 200.0, r, 0.1, "Large override should be 200y")
}

func TestGetVisibilityRange_ObjectOverrideGigantic(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.visibilityRange = 100.0

	obj := object.NewObject()
	obj.SetVisibilityOverrideType(object.VisibilityDistanceGigantic)
	r := GetVisibilityRange(obj, m)
	assert.InDelta(t, 400.0, r, 0.1, "Gigantic override should be 400y")
}

func TestGetVisibilityRange_ObjectOverrideInfinite(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.visibilityRange = 100.0

	obj := object.NewObject()
	obj.SetVisibilityOverrideType(object.VisibilityDistanceInfinite)
	r := GetVisibilityRange(obj, m)
	assert.InDelta(t, 533.0, r, 0.1, "Infinite override should be 533y")
}

// ---------------------------------------------------------------------------
// Backward compatibility — existing constants still work
// ---------------------------------------------------------------------------

func TestBackwardCompat_Constants(t *testing.T) {
	// The old constants should still be usable
	assert.InDelta(t, float32(50.0), VisibilityDistanceNear, 0.1)
	assert.InDelta(t, float32(200.0), VisibilityDistanceFar, 0.1)
	assert.InDelta(t, float32(500.0), VisibilityDistanceInfiniteLegacy, 0.1)
}
