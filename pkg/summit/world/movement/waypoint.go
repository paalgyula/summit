package movement

import (
	"time"

	"github.com/paalgyula/summit/pkg/store"
)

// WaypointGenerator follows a waypoint path loaded from the database.
// It mirrors AzerothCore's WaypointMovementGenerator.
type WaypointGenerator struct {
	pathID          uint32
	waypointIndex   int
	delayEnd        int64 // Unix ms
	repeating       bool
	recalculateSpeed bool
	path            *store.WaypointPath
}

// NewWaypointGenerator creates a WaypointGenerator for the given path.
func NewWaypointGenerator(pathID uint32, path *store.WaypointPath, repeating bool) *WaypointGenerator {
	return &WaypointGenerator{
		pathID:    pathID,
		path:      path,
		repeating: repeating,
	}
}

func (g *WaypointGenerator) Initialize(_ interface{}) {
	g.waypointIndex = 0
	g.delayEnd = 0
}

func (g *WaypointGenerator) Update(owner interface{}, _ uint32) bool {
	o, ok := owner.(MovementOwner)
	if !ok || !o.IsAlive() || o.IsInCombat() {
		return true // stay active but don't move
	}

	if g.path == nil || len(g.path.Points) == 0 {
		return false // no path, done
	}

	now := time.Now().UnixMilli()

	// If waiting out a delay at a waypoint, check the timer
	if g.delayEnd > 0 {
		if now < g.delayEnd {
			return true // still waiting
		}
		g.delayEnd = 0 // delay expired, advance below
	}

	// Get current waypoint
	if g.waypointIndex >= len(g.path.Points) {
		if g.repeating {
			g.waypointIndex = 0
		} else {
			return false // path complete
		}
	}

	wp := g.path.Points[g.waypointIndex]

	// Determine movement flags
	var flags uint32
	if wp.MoveType == 0 {
		flags = SplineFlagWalkMode
	} else {
		flags = SplineFlagRunMode
	}

	// Start movement to this waypoint
	o.MoveTo(wp.X, wp.Y, wp.Z, time.Now(), flags, nil)

	// Set orientation if specified
	if wp.Orientation != 0 {
		o.SetOrientation(wp.Orientation)
	}

	// Set delay if waypoint has one
	if wp.Delay > 0 {
		g.delayEnd = now + int64(wp.Delay)
	}

	// Advance to next waypoint
	g.waypointIndex++

	return true
}

func (g *WaypointGenerator) Finalize(_ interface{}) {}

func (g *WaypointGenerator) Reset(_ interface{}) {
	g.waypointIndex = 0
	g.delayEnd = 0
}

func (g *WaypointGenerator) Type() MovementGeneratorType {
	return MotionTypeWAYPOINT
}

func (g *WaypointGenerator) GetSplineId() uint32 {
	return 0
}

// GetCurrentNode returns the current waypoint index.
func (g *WaypointGenerator) GetCurrentNode() int {
	return g.waypointIndex
}

// GetPathID returns the path ID.
func (g *WaypointGenerator) GetPathID() uint32 {
	return g.pathID
}
