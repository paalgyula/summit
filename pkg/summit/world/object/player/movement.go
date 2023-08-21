package player

import "github.com/paalgyula/summit/pkg/wow"

// Movement info.
type Movement struct {
	MoveFlags  wow.MovementFlag
	MoveFlags2 uint8
	Time       uint32 // time in millisecond
	Position   WorldLocation

	TransportGUID uint64
	TransportPos  WorldLocation
	TransportTime uint32

	SwimmingPitch float32
	FallTime      uint32

	JumpVelocity float32
	JumpSinAngle float32
	JumpCosAngle float32
	JumpXyspeed  float32

	Spline float32
}
