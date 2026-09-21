package mapmanager

import (
	"math"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// ---------------------------------------------------------------------------
// Visibility distance constants — backward compatible with old names
// ---------------------------------------------------------------------------

const (
	// DefaultVisibilityDistance is the default visibility range for most objects.
	DefaultVisibilityDistance float32 = 100.0

	// VisibilityDistanceNear is for close-range visibility (e.g., in cities).
	VisibilityDistanceNear float32 = 50.0

	// VisibilityDistanceFar is for long-range visibility (e.g., open world).
	VisibilityDistanceFar float32 = 200.0

	// VisibilityDistanceInfiniteLegacy is for objects that are always visible.
	// Renamed to avoid collision with VisibilityDistanceType::Infinite.
	VisibilityDistanceInfiniteLegacy float32 = 500.0

	// MaxVisibilityObjects is the maximum number of objects to track per player.
	MaxVisibilityObjects = 200
)

// ---------------------------------------------------------------------------
// Per-map visibility overrides — mirrors Map::InitVisibilityDistance
// ---------------------------------------------------------------------------

// InitVisibilityDistance sets the map's default visibility range based on map type.
// Mirrors AzerothCore's Map::InitVisibilityDistance in Map.cpp.
func (m *Map) InitVisibilityDistance() {
	// Start with continent default
	m.visibilityRange = float32(object.VisibilityDistances[object.VisibilityDistanceNormal])

	switch m.ID {
	case 609: // Scarlet Enclave (DK starting zone)
		m.visibilityRange = 125.0
	}

	// Instances override the default
	if m.IsDungeon() {
		m.visibilityRange = 170.0 // DEFAULT_VISIBILITY_INSTANCE
	} else if m.IsBattleground() || m.IsBattleArena() {
		m.visibilityRange = 250.0 // DEFAULT_VISIBILITY_BGARENAS
	}
}

// ---------------------------------------------------------------------------
// GetVisibilityRange — per-object and per-map visibility
// ---------------------------------------------------------------------------

// GetVisibilityRange returns the effective visibility range for an object on a map.
// Mirrors AzerothCore's WorldObject::GetVisibilityRange in Object.cpp.
//
// Priority:
//  1. Object's override type (if set and not Normal)
//  2. Map's visibility range
//  3. Default (100y)
func GetVisibilityRange(obj *object.Object, m *Map) float32 {
	// Check object override first
	if obj != nil {
		vdType := obj.GetVisibilityOverrideType()
		if vdType != object.VisibilityDistanceNormal {
			return float32(object.VisibilityDistances[vdType])
		}
	}

	// Fall back to map range
	if m != nil && m.visibilityRange > 0 {
		return m.visibilityRange
	}

	return DefaultVisibilityDistance
}

// GetSightRange returns the sight range for a player.
// This is the distance at which the player can see objects.
func GetSightRange(p *player.Player) float32 {
	// Use the player's map visibility range if available
	// For now, return default — will be wired to map in Phase 5
	return DefaultVisibilityDistance
}

// GetEffectiveSightRange returns the sight range for a player on a specific map.
// This considers the map's visibility range (dungeon, BG, etc.).
func GetEffectiveSightRange(p *player.Player, m *Map) float32 {
	if m != nil && m.visibilityRange > 0 {
		return m.visibilityRange
	}
	return DefaultVisibilityDistance
}

// ---------------------------------------------------------------------------
// Distance helpers
// ---------------------------------------------------------------------------

// Distance2D calculates the 2D distance between two objects (ignoring Z).
func Distance2D(a, b *object.Object) float32 {
	// For now, use a simplified approach - objects don't store position directly
	// The position is stored on the Player/NPC wrapper
	// This will be improved when we add position to Object
	return 0
}

// Distance2DPositions calculates the 2D distance between two positions.
func Distance2DPositions(x1, y1, x2, y2 float32) float32 {
	dx := x1 - x2
	dy := y1 - y2
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// IsWithinRange returns true if two positions are within the given range.
func IsWithinRange(x1, y1, x2, y2, range_ float32) bool {
	return Distance2DPositions(x1, y1, x2, y2) <= range_
}

// ---------------------------------------------------------------------------
// VisibilityTracker — unchanged from original, kept for backward compat
// ---------------------------------------------------------------------------

// VisibilityTracker tracks which objects each player can currently see.
// It stores the set of object GUIDs that have been sent to each player.
type VisibilityTracker struct {
	// playerVisible tracks which GUIDs each player has been sent.
	// Key: player GUID, Value: set of object GUIDs
	playerVisible map[wow.GUID]map[wow.GUID]struct{}
}

// NewVisibilityTracker creates a NewVisibilityTracker.
func NewVisibilityTracker() *VisibilityTracker {
	return &VisibilityTracker{
		playerVisible: make(map[wow.GUID]map[wow.GUID]struct{}),
	}
}

// IsVisible returns true if the player has been sent the object.
func (vt *VisibilityTracker) IsVisible(playerGUID, objectGUID wow.GUID) bool {
	if objects, ok := vt.playerVisible[playerGUID]; ok {
		_, seen := objects[objectGUID]
		return seen
	}
	return false
}

// SetVisible marks an object as visible to a player.
func (vt *VisibilityTracker) SetVisible(playerGUID, objectGUID wow.GUID) {
	if _, ok := vt.playerVisible[playerGUID]; !ok {
		vt.playerVisible[playerGUID] = make(map[wow.GUID]struct{})
	}
	vt.playerVisible[playerGUID][objectGUID] = struct{}{}
}

// ClearVisible removes an object from a player's visible set.
func (vt *VisibilityTracker) ClearVisible(playerGUID, objectGUID wow.GUID) {
	if objects, ok := vt.playerVisible[playerGUID]; ok {
		delete(objects, objectGUID)
		if len(objects) == 0 {
			delete(vt.playerVisible, playerGUID)
		}
	}
}

// ClearPlayer removes all visibility tracking for a player.
func (vt *VisibilityTracker) ClearPlayer(playerGUID wow.GUID) {
	delete(vt.playerVisible, playerGUID)
}

// GetVisibleCount returns the number of objects visible to a player.
func (vt *VisibilityTracker) GetVisibleCount(playerGUID wow.GUID) int {
	if objects, ok := vt.playerVisible[playerGUID]; ok {
		return len(objects)
	}
	return 0
}
