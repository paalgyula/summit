package movement

// IdleGenerator is a no-op movement generator. It is used as the default
// movement when no other movement is active. It is a singleton in AzerothCore.
type IdleGenerator struct{}

// NewIdleGenerator creates an IdleGenerator.
func NewIdleGenerator() *IdleGenerator {
	return &IdleGenerator{}
}

func (g *IdleGenerator) Initialize(_ interface{}) {}

func (g *IdleGenerator) Update(_ interface{}, _ uint32) bool {
	return true // Always active, never expires
}

func (g *IdleGenerator) Finalize(_ interface{}) {}

func (g *IdleGenerator) Reset(_ interface{}) {}

func (g *IdleGenerator) Type() MovementGeneratorType {
	return MotionTypeIDLE
}

func (g *IdleGenerator) GetSplineId() uint32 {
	return 0
}
