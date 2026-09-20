package areatrigger

import (
	"math"
)

// AreaTrigger represents a trigger zone in the world.
type AreaTrigger struct {
	Entry       uint32
	MapID       uint32
	X, Y, Z     float32
	Radius      float32
	Length      float32
	Width       float32
	Height      float32
	Orientation float32
}

// AreaTriggerTeleport defines teleport destination for an areatrigger.
type AreaTriggerTeleport struct {
	TargetMapID               uint32
	TargetX, TargetY, TargetZ float32
	TargetO                   float32
}

// IsInside checks if the given position is inside the areatrigger.
// Uses radius-based check for circular triggers.
func (at *AreaTrigger) IsInside(x, y, z float32) bool {
	if at.Radius > 0 {
		// Circle check
		dx := x - at.X
		dy := y - at.Y
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		return dist <= at.Radius
	}

	// Box check (for rectangular triggers)
	if at.Length > 0 && at.Width > 0 {
		dx := x - at.X
		dy := y - at.Y

		// Rotate point by negative orientation
		cosO := float32(math.Cos(float64(-at.Orientation)))
		sinO := float32(math.Sin(float64(-at.Orientation)))

		rotX := dx*cosO - dy*sinO
		rotY := dx*sinO + dy*cosO

		halfLen := at.Length / 2
		halfWidth := at.Width / 2

		return math.Abs(float64(rotX)) <= float64(halfLen) &&
			math.Abs(float64(rotY)) <= float64(halfWidth)
	}

	return false
}

// DistanceTo returns the distance from the given point to the areatrigger center.
func (at *AreaTrigger) DistanceTo(x, y, z float32) float32 {
	dx := x - at.X
	dy := y - at.Y
	dz := z - at.Z

	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}
