package movement

// ConfusedGenerator moves the creature randomly while confused.
// It mirrors AzerothCore's ConfusedMovementGenerator.
type ConfusedGenerator struct {
	duration     uint32 // total duration in ms
	elapsed      uint32
	nextMoveTime uint32
}

// NewConfusedGenerator creates a ConfusedGenerator for the given duration.
func NewConfusedGenerator(duration uint32) *ConfusedGenerator {
	if duration == 0 {
		duration = 5000 // default 5 seconds
	}
	return &ConfusedGenerator{
		duration: duration,
	}
}

func (g *ConfusedGenerator) Initialize(_ interface{}) {
	g.elapsed = 0
	g.nextMoveTime = 0
}

func (g *ConfusedGenerator) Update(_ interface{}, diff uint32) bool {
	g.elapsed += diff
	if g.elapsed >= g.duration {
		return false
	}
	// TODO: Implement confused movement with MoveSplineInit
	// For now, this is a stub that will be completed in Story 8
	return true
}

func (g *ConfusedGenerator) Finalize(_ interface{}) {}

func (g *ConfusedGenerator) Reset(_ interface{}) {
	g.elapsed = 0
	g.nextMoveTime = 0
}

func (g *ConfusedGenerator) Type() MovementGeneratorType {
	return MotionTypeCONFUSED
}

func (g *ConfusedGenerator) GetSplineId() uint32 {
	return 0
}
