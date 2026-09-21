package mapmanager

import (
	"math"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Default visibility distances (in yards)
const (
	// DefaultVisibilityDistance is the default visibility range for most objects.
	DefaultVisibilityDistance float32 = 100.0

	// VisibilityDistanceNear is for close-range visibility (e.g., in cities).
	VisibilityDistanceNear float32 = 50.0

	// VisibilityDistanceFar is for long-range visibility (e.g., open world).
	VisibilityDistanceFar float32 = 200.0

	// VisibilityDistanceInfinite is for objects that are always visible.
	VisibilityDistanceInfinite float32 = 500.0

	// GridCellSize is the size of a grid cell in world units.
	GridCellSize float32 = 50.0

	// MaxVisibilityObjects is the maximum number of objects to track per player.
	MaxVisibilityObjects = 200
)

// VisibilityTracker tracks which objects each player can currently see.
// It stores the set of object GUIDs that have been sent to each player.
type VisibilityTracker struct {
	// playerVisible tracks which GUIDs each player has been sent.
	// Key: player GUID, Value: set of object GUIDs
	playerVisible map[wow.GUID]map[wow.GUID]struct{}
}

// NewVisibilityTracker creates a new VisibilityTracker.
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

// GetVisibilityRange returns the visibility range for an object.
// This mirrors AzerothCore's WorldObject::GetVisibilityRange.
func GetVisibilityRange(obj *object.Object, m *Map) float32 {
	// Check map's visibility range first
	if m != nil && m.visibilityRange > 0 {
		return m.visibilityRange
	}

	// Default visibility based on object type
	switch obj.ObjectTypeID() {
	case wow.TypeIDPlayer:
		return DefaultVisibilityDistance
	case wow.TypeIDUnit:
		return DefaultVisibilityDistance
	case wow.TypeIDGameObject:
		return DefaultVisibilityDistance
	case wow.TypeIDDynamicoObject:
		return DefaultVisibilityDistance
	case wow.TypeIDCorpse:
		return DefaultVisibilityDistance
	default:
		return DefaultVisibilityDistance
	}
}

// GetSightRange returns the sight range for a player.
// This is the distance at which the player can see objects.
func GetSightRange(p *player.Player) float32 {
	return DefaultVisibilityDistance
}

// Distance2D calculates the 2D distance between two objects (ignoring Z).
func Distance2D(a, b *object.Object) float32 {
	// Get positions from the objects
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
