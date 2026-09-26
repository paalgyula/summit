package movement

import (
	"math"
	"time"
)

// FollowGenerator pursues a target unit at a specified distance and angle.
// It mirrors AzerothCore's FollowMovementGenerator from TargetedMovementGenerator.cpp.
type FollowGenerator struct {
	target           interface{}
	range_           float32 // desired follow distance (e.g. 2.0 yards)
	angle            float32 // follow angle relative to target facing (e.g. math.Pi for behind)
	lastTargetPos    [3]float32
	recalculateTimer uint32
	recalcInterval   uint32 // ms between recalculations (default 400ms like AzerothCore)
}

// NewFollowGenerator creates a FollowGenerator following the given target.
func NewFollowGenerator(target interface{}, dist float32, angle float32) *FollowGenerator {
	if dist <= 0 {
		dist = 2.0 // default follow distance
	}
	return &FollowGenerator{
		target:         target,
		range_:         dist,
		angle:          angle,
		recalcInterval: 400, // 400ms interval like AzerothCore
	}
}

func (g *FollowGenerator) Initialize(_ interface{}) {
	g.recalculateTimer = 0
	g.lastTargetPos = [3]float32{}
}

func (g *FollowGenerator) Update(owner interface{}, diff uint32) bool {
	o, ok := owner.(MovementOwner)
	if !ok || !o.IsAlive() {
		return false
	}

	type followTarget interface {
		GetPositionX() float32
		GetPositionY() float32
		GetPositionZ() float32
		IsAlive() bool
	}

	target, ok := g.target.(followTarget)
	if !ok || !target.IsAlive() {
		return false // target dead or invalid, stop following
	}

	tx := target.GetPositionX()
	ty := target.GetPositionY()
	tz := target.GetPositionZ()

	var to float32
	type orientationGetter interface {
		GetOrientation() float32
	}
	if og, ok := g.target.(orientationGetter); ok {
		to = og.GetOrientation()
	}

	// Calculate desired follow position (offset by angle from target facing)
	offsetAngle := float64(to + g.angle)
	destX := tx + g.range_*float32(math.Cos(offsetAngle))
	destY := ty + g.range_*float32(math.Sin(offsetAngle))
	destZ := tz

	ox, oy, oz := o.GetCurrentPosition()
	dx := destX - ox
	dy := destY - oy
	dz := destZ - oz
	distToDest := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	tdx := tx - ox
	tdy := ty - oy
	tdz := tz - oz
	distToTarget := float32(math.Sqrt(float64(tdx*tdx + tdy*tdy + tdz*tdz)))

	// Within tolerance?
	if distToDest < 1.0 || distToTarget <= g.range_+0.5 {
		if o.HasActiveMovement() {
			o.StopMoving(nil)
		}
		// Match target orientation
		o.SetOrientation(to)
		return true
	}

	g.recalculateTimer += diff
	targetMovedDiffX := tx - g.lastTargetPos[0]
	targetMovedDiffY := ty - g.lastTargetPos[1]
	targetMovedDist := float32(math.Sqrt(float64(targetMovedDiffX*targetMovedDiffX + targetMovedDiffY*targetMovedDiffY)))

	if !o.HasActiveMovement() || (g.recalculateTimer >= g.recalcInterval && targetMovedDist > 1.0) {
		g.recalculateTimer = 0

		flags := uint32(SplineFlagRunMode)
		if distToTarget < 5.0 {
			flags = SplineFlagWalkMode
		}

		o.MoveTo(destX, destY, destZ, time.Now(), flags, nil)
		g.lastTargetPos[0] = tx
		g.lastTargetPos[1] = ty
		g.lastTargetPos[2] = tz
	}

	return true
}

func (g *FollowGenerator) Finalize(_ interface{}) {}

func (g *FollowGenerator) Reset(_ interface{}) {
	g.recalculateTimer = 0
}

func (g *FollowGenerator) Type() MovementGeneratorType {
	return MotionTypeFOLLOW
}

func (g *FollowGenerator) GetSplineId() uint32 {
	return 0
}

// GetTarget returns the followed target.
func (g *FollowGenerator) GetTarget() interface{} {
	return g.target
}

// SetTarget sets a new target to follow.
func (g *FollowGenerator) SetTarget(target interface{}) {
	g.target = target
	g.lastTargetPos = [3]float32{}
}

// SetOffsetAndAngle adjusts follow distance and relative angle.
func (g *FollowGenerator) SetOffsetAndAngle(dist, angle float32) {
	if dist > 0 {
		g.range_ = dist
	}
	g.angle = angle
	g.lastTargetPos = [3]float32{}
}
