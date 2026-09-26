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
	tx, ty, tz := target.GetPositionX(), target.GetPositionY(), target.GetPositionZ()

	// Check leash range from spawn position
	sx, sy, sz := o.GetSpawnPosition()
	sdx := ox - sx
	sdy := oy - sy
	sdz := oz - sz
	distFromSpawn := float32(math.Sqrt(float64(sdx*sdx + sdy*sdy + sdz*sdz)))
	if distFromSpawn > g.maxChaseDist {
		return false // leash exceeded from spawn
	}

	// Calculate distance to target
	dx := tx - ox
	dy := ty - oy
	dz := tz - oz
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	// Check leash range to target
	if distance > g.maxChaseDist {
		return false
	}

	// If in melee range (3.5 yards), stop moving and face target
	if distance <= 3.5 {
		if o.HasActiveMovement() {
			o.StopMoving(nil)
		}
		o.SetOrientation(float32(math.Atan2(float64(dy), float64(dx))))
		return true
	}

	// Periodically check if we need to recalculate path or start moving
	g.recalculateTimer += diff
	targetDestDiffX := tx - g.lastTargetPos[0]
	targetDestDiffY := ty - g.lastTargetPos[1]
	targetMovedDist := float32(math.Sqrt(float64(targetDestDiffX*targetDestDiffX + targetDestDiffY*targetDestDiffY)))

	if !o.HasActiveMovement() || (g.recalculateTimer >= g.recalcInterval && targetMovedDist > 1.5) {
		g.recalculateTimer = 0
		o.MoveTo(tx, ty, tz, time.Now(), SplineFlagRunMode, nil)
		g.lastTargetPos[0] = tx
		g.lastTargetPos[1] = ty
		g.lastTargetPos[2] = tz
	}

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
