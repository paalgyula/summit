package movement

// FleeingGenerator moves the creature away from a threat.
// It mirrors AzerothCore's FleeingMovementGenerator.
type FleeingGenerator struct {
	enemy        interface{}
	duration     uint32 // total duration in ms
	elapsed      uint32
	nextMoveTime uint32
}

// NewFleeingGenerator creates a FleeingGenerator that flees from enemy for duration ms.
func NewFleeingGenerator(enemy interface{}, duration uint32) *FleeingGenerator {
	if duration == 0 {
		duration = 8000 // default 8 seconds
	}
	return &FleeingGenerator{
		enemy:    enemy,
		duration: duration,
	}
}

func (g *FleeingGenerator) Initialize(_ interface{}) {
	g.elapsed = 0
	g.nextMoveTime = 0
}

func (g *FleeingGenerator) Update(_ interface{}, diff uint32) bool {
	g.elapsed += diff
	if g.elapsed >= g.duration {
		return false
	}
	// TODO: Implement fleeing movement with MoveSplineInit
	// For now, this is a stub that will be completed in Story 8
	return true
}

func (g *FleeingGenerator) Finalize(_ interface{}) {}

func (g *FleeingGenerator) Reset(_ interface{}) {
	g.elapsed = 0
	g.nextMoveTime = 0
}

func (g *FleeingGenerator) Type() MovementGeneratorType {
	return MotionTypeFLEEING
}

func (g *FleeingGenerator) GetSplineId() uint32 {
	return 0
}
