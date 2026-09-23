package movement

import (
	"math/rand"
	"time"
)

// RandomGenerator picks random points within wander distance of the spawn point
// and moves the creature there. It mirrors AzerothCore's RandomMovementGenerator.
type RandomGenerator struct {
	wanderDist  float32
	nextMoveTime int64 // Unix ms
	moveCount    uint8
}

// NewRandomGenerator creates a RandomGenerator with the given wander distance.
func NewRandomGenerator(wanderDist float32) *RandomGenerator {
	if wanderDist <= 0 {
		wanderDist = 5.0 // default wander distance
	}
	return &RandomGenerator{
		wanderDist: wanderDist,
	}
}

func (g *RandomGenerator) Initialize(_ interface{}) {
	g.nextMoveTime = 0
	g.moveCount = 0
}

func (g *RandomGenerator) Update(owner interface{}, _ uint32) bool {
	// TODO: Implement random movement with MoveSplineInit
	// For now, this is a stub that will be completed in Story 2
	return true
}

func (g *RandomGenerator) Finalize(_ interface{}) {}

func (g *RandomGenerator) Reset(_ interface{}) {
	g.nextMoveTime = 0
	g.moveCount = 0
}

func (g *RandomGenerator) Type() MovementGeneratorType {
	return MotionTypeRANDOM
}

func (g *RandomGenerator) GetSplineId() uint32 {
	return 0
}

// randomWait returns a random duration between min and max milliseconds.
func randomWait(min, max int) int64 {
	return int64(min + rand.Intn(max-min))
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
