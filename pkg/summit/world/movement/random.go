package movement

import (
	"math"
	"math/rand"
	"time"
)

// Spline flags (MoveSplineFlag) - matching the world package constants.
const (
	SplineFlagNone       uint32 = 0x00000000
	SplineFlagRunMode    uint32 = 0x00000000
	SplineFlagWalkMode   uint32 = 0x00000100
	SplineFlagFlying     uint32 = 0x00000200
	SplineFlagCatmullRom uint32 = 0x00020000
)

// RandomGenerator picks random points within wander distance of the spawn point
// and moves the creature there. It mirrors AzerothCore's RandomMovementGenerator.
type RandomGenerator struct {
	wanderDist   float32
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
	o, ok := owner.(MovementOwner)
	if !ok || !o.IsAlive() || o.IsInCombat() {
		return true // stay active but don't move
	}

	now := time.Now().UnixMilli()

	// Initialize first wander timer
	if g.nextMoveTime == 0 {
		g.nextMoveTime = now + randomWait(5000, 15000)
		return true
	}

	// Wait for timer
	if now < g.nextMoveTime {
		return true
	}

	// Don't start a new wander if already moving
	if o.HasActiveMovement() {
		return true
	}

	// Pick random point within wander distance of spawn
	spawnX, spawnY, spawnZ := o.GetSpawnPosition()
	angle := rand.Float64() * 2 * math.Pi
	dist := 1.0 + rand.Float64()*float64(g.wanderDist-1.0)
	if dist < 1.0 {
		dist = 1.0
	}

	destX := spawnX + float32(dist*math.Cos(angle))
	destY := spawnY + float32(dist*math.Sin(angle))
	destZ := spawnZ // keep same Z as spawn

	// Creatures walk while wandering (AzerothCore behavior)
	flags := uint32(SplineFlagWalkMode)

	// Start movement
	o.MoveTo(destX, destY, destZ, time.Now(), flags, nil)

	// Schedule next wander
	g.nextMoveTime = now + randomWait(10000, 25000)
	g.moveCount++

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
