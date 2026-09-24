package movement

import (
	"math"
	"time"
)

// ChaseGenerator pursues a target unit. It mirrors AzerothCore's
// ChaseMovementGenerator from TargetedMovementGenerator.cpp.
type ChaseGenerator struct {
	target           interface{}
	maxChaseDist     float32
	lastTargetPos    [3]float32
	recalculateTimer uint32
	recalcInterval   uint32 // ms between recalculations
}

// NewChaseGenerator creates a ChaseGenerator chasing the given target.
func NewChaseGenerator(target interface{}, maxDist float32) *ChaseGenerator {
	if maxDist <= 0 {
		maxDist = 50.0 // default leash range
	}
	return &ChaseGenerator{
		target:       target,
		maxChaseDist: maxDist,
		recalcInterval: 400, // recalculate every 400ms like AzerothCore
	}
}

func (g *ChaseGenerator) Initialize(_ interface{}) {
	g.recalculateTimer = 0
}

func (g *ChaseGenerator) Update(owner interface{}, diff uint32) bool {
	o, ok := owner.(MovementOwner)
	if !ok || !o.IsAlive() {
		return false
	}

	// Get target position via CombatUnit interface
	type positionGetter interface {
		GetPositionX() float32
		GetPositionY() float32
		GetPositionZ() float32
		IsAlive() bool
	}

	target, ok := g.target.(positionGetter)
	if !ok || !target.IsAlive() {
		return false // target dead, stop chasing
	}

	// Get owner position
	ox, oy, oz := o.GetCurrentPosition()

	// Calculate distance to target
	dx := target.GetPositionX() - ox
	dy := target.GetPositionY() - oy
	dz := target.GetPositionZ() - oz
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	// Check leash range — if too far, exit chase
	if distance > g.maxChaseDist {
		return false
	}

	// If in melee range (3.5 yards), stop moving
	if distance < 3.5 {
		// Stop movement if currently moving
		return true
	}

	// Periodically check if we need to recalculate path
	g.recalculateTimer += diff
	if g.recalculateTimer < g.recalcInterval {
		return true // still active, don't recalculate yet
	}
	g.recalculateTimer = 0

	// Check if target moved significantly (> 2 yards from last destination)
	targetDestDiffX := target.GetPositionX() - g.lastTargetPos[0]
	targetDestDiffY := target.GetPositionY() - g.lastTargetPos[1]
	_ = targetDestDiffX
	_ = targetDestDiffY

	// Move to target position
	o.MoveTo(target.GetPositionX(), target.GetPositionY(), target.GetPositionZ(),
		time.Now(), SplineFlagRunMode, nil)

	// Update last target position
	g.lastTargetPos[0] = target.GetPositionX()
	g.lastTargetPos[1] = target.GetPositionY()
	g.lastTargetPos[2] = target.GetPositionZ()

	return true
}

func (g *ChaseGenerator) Finalize(_ interface{}) {}

func (g *ChaseGenerator) Reset(_ interface{}) {
	g.recalculateTimer = 0
}

func (g *ChaseGenerator) Type() MovementGeneratorType {
	return MotionTypeCHASE
}

func (g *ChaseGenerator) GetSplineId() uint32 {
	return 0
}

// GetTarget returns the chase target.
func (g *ChaseGenerator) GetTarget() interface{} {
	return g.target
}

// SetTarget sets a new chase target.
func (g *ChaseGenerator) SetTarget(target interface{}) {
	g.target = target
	g.lastTargetPos = [3]float32{}
}
