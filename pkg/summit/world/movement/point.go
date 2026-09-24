package movement

// PointGenerator moves the creature to a specific point.
// It mirrors AzerothCore's PointMovementGenerator.
type PointGenerator struct {
	x, y, z  float32
	pointID  uint32
	arrived  bool
	walk     bool
}

// NewPointGenerator creates a PointGenerator for the given coordinates.
func NewPointGenerator(x, y, z float32, pointID uint32, walk bool) *PointGenerator {
	return &PointGenerator{
		x:       x,
		y:       y,
		z:       z,
		pointID: pointID,
		walk:    walk,
	}
}

func (g *PointGenerator) Initialize(_ interface{}) {
	g.arrived = false
}

func (g *PointGenerator) Update(_ interface{}, _ uint32) bool {
	// TODO: Implement point movement with MoveSplineInit
	// For now, this is a stub that will be completed in Story 8
	return !g.arrived
}

func (g *PointGenerator) Finalize(_ interface{}) {}

func (g *PointGenerator) Reset(_ interface{}) {
	g.arrived = false
}

func (g *PointGenerator) Type() MovementGeneratorType {
	return MotionTypePOINT
}

func (g *PointGenerator) GetSplineId() uint32 {
	return 0
}

// GetPointID returns the script-defined point ID.
func (g *PointGenerator) GetPointID() uint32 {
	return g.pointID
}
