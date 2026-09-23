package movement

// WaypointGenerator follows a waypoint path loaded from the database.
// It mirrors AzerothCore's WaypointMovementGenerator.
type WaypointGenerator struct {
	pathID         uint32
	waypointIndex  int
	delayEnd       int64 // Unix ms
	repeating      bool
	recalculateSpeed bool
}

// NewWaypointGenerator creates a WaypointGenerator for the given path.
func NewWaypointGenerator(pathID uint32, repeating bool) *WaypointGenerator {
	return &WaypointGenerator{
		pathID:    pathID,
		repeating: repeating,
	}
}

func (g *WaypointGenerator) Initialize(_ interface{}) {
	g.waypointIndex = 0
	g.delayEnd = 0
}

func (g *WaypointGenerator) Update(_ interface{}, _ uint32) bool {
	// TODO: Implement waypoint following with MoveSplineInit
	// For now, this is a stub that will be completed in Story 2
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
