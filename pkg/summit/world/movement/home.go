package movement

import (
	"math"
	"time"
)

// HomeGenerator moves the creature back to its spawn point after combat.
// It mirrors AzerothCore's HomeMovementGenerator.
type HomeGenerator struct {
	arrived   bool
	walk      bool
	recalc    bool
}

// NewHomeGenerator creates a HomeGenerator. If walk is true, the creature walks home.
func NewHomeGenerator(walk bool) *HomeGenerator {
	return &HomeGenerator{walk: walk}
}

func (g *HomeGenerator) Initialize(_ interface{}) {
	g.arrived = false
	g.recalc = false
}

func (g *HomeGenerator) Update(owner interface{}, _ uint32) bool {
	if g.arrived {
		return false
	}

	o, ok := owner.(MovementOwner)
	if !ok || !o.IsAlive() {
		return false
	}

	// Get current position
	ox, oy, oz := o.GetCurrentPosition()

	// Get spawn position
	sx, sy, sz := o.GetSpawnPosition()

	// Calculate distance to spawn
	dx := sx - ox
	dy := sy - oy
	dz := sz - oz
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	// If close enough to spawn, we've arrived
	if distance < 0.5 {
		g.arrived = true
		return false
	}

	// Determine movement flags
	var flags uint32
	if g.walk {
		flags = SplineFlagWalkMode
	} else {
		flags = SplineFlagRunMode
	}

	// Move to spawn position
	o.MoveTo(sx, sy, sz, time.Now(), flags, nil)

	return true
}

func (g *HomeGenerator) Finalize(_ interface{}) {}

func (g *HomeGenerator) Reset(_ interface{}) {
	g.arrived = false
	g.recalc = false
}

func (g *HomeGenerator) Type() MovementGeneratorType {
	return MotionTypeHOME
}

func (g *HomeGenerator) GetSplineId() uint32 {
	return 0
}

// HasArrived returns true if the creature has reached its spawn point.
func (g *HomeGenerator) HasArrived() bool {
	return g.arrived
}
