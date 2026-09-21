package object

import "github.com/paalgyula/summit/pkg/wow"

// MovementBlock is everything a CreateObject block carries between the
// update flags and the values block. Every create builder in the server
// goes through WriteMovementBlock so the wire layout stays the same for
// players, creatures and game objects.
type MovementBlock struct {
	Flags  wow.MovementFlag
	Flags2 uint16
	Time   uint32

	X, Y, Z, O float32

	FallTime uint32
	// Jump data, only written while Flags has MovementFlagFalling.
	JumpZSpeed, JumpSinAngle, JumpCosAngle, JumpXYSpeed float32

	// Indexed by wow.MoveType; the pitch rate is not a MoveType and is
	// always written as 0.
	Speeds [wow.MoveTypeMax]float32

	// Written for UpdateFlagLowGUID.
	LowGUID uint32
}

// WriteMovementBlock writes the 3.3.5a movement update (Object::BuildMovementUpdate).
//
//nolint:errcheck
func WriteMovementBlock(buf *UpdateBlockBuffer, updateFlags wow.ObjectUpdateFlags, mv *MovementBlock) {
	if updateFlags&wow.UpdateFlagLiving != 0 {
		_ = buf.Write(mv.Flags)
		_ = buf.Write(mv.Flags2)
		_ = buf.Write(mv.Time)

		_ = buf.Write(mv.X)
		_ = buf.Write(mv.Y)
		_ = buf.Write(mv.Z)
		_ = buf.Write(mv.O)

		// Transport data is never written: MovementFlagOnTransport is not set by the server.

		if mv.Flags&(wow.MovementFlagSwimming|wow.MovementFlagFlying) != 0 {
			_ = buf.Write(float32(0)) // pitch
		}

		_ = buf.Write(mv.FallTime)

		if mv.Flags&wow.MovementFlagFalling != 0 {
			_ = buf.Write(mv.JumpZSpeed)
			_ = buf.Write(mv.JumpSinAngle)
			_ = buf.Write(mv.JumpCosAngle)
			_ = buf.Write(mv.JumpXYSpeed)
		}

		if mv.Flags&wow.MovementFlagSplineElevation != 0 {
			_ = buf.Write(float32(0))
		}

		_ = buf.Write(mv.Speeds[wow.MoveTypeWalk])
		_ = buf.Write(mv.Speeds[wow.MoveTypeRun])
		_ = buf.Write(mv.Speeds[wow.MoveTypeRunBack])
		_ = buf.Write(mv.Speeds[wow.MoveTypeSwim])
		_ = buf.Write(mv.Speeds[wow.MoveTypeSwimBack])
		_ = buf.Write(mv.Speeds[wow.MoveTypeFlight])
		_ = buf.Write(mv.Speeds[wow.MoveTypeFlightBack])
		_ = buf.Write(mv.Speeds[wow.MoveTypeTurnRate])
		_ = buf.Write(float32(0)) // pitch rate

		// Spline data is never written: MovementFlagSplineEnabled is not set by the server.
	} else if updateFlags&wow.UpdateFlagPosition != 0 {
		_ = buf.WriteOne(0) // transport packed GUID (none)
		_ = buf.Write(mv.X)
		_ = buf.Write(mv.Y)
		_ = buf.Write(mv.Z)
		_ = buf.Write(mv.X)
		_ = buf.Write(mv.Y)
		_ = buf.Write(mv.Z)
		_ = buf.Write(mv.O)
		_ = buf.Write(float32(0)) // corpse orientation
	} else if updateFlags&wow.UpdateFlagStationaryPosition != 0 {
		_ = buf.Write(mv.X)
		_ = buf.Write(mv.Y)
		_ = buf.Write(mv.Z)
		_ = buf.Write(mv.O)
	}

	if updateFlags&wow.UpdateFlagUnknown != 0 {
		_ = buf.WriteUint32(0)
	}

	if updateFlags&wow.UpdateFlagLowGUID != 0 {
		_ = buf.WriteUint32(mv.LowGUID)
	}

	if updateFlags&wow.UpdateFlagHasTarget != 0 {
		_ = buf.WriteOne(0) // packed GUID of the target (none)
	}

	if updateFlags&wow.UpdateFlagTransport != 0 {
		_ = buf.WriteUint32(0) // transport path progress
	}

	if updateFlags&wow.UpdateFlagVehicle != 0 {
		_ = buf.WriteUint32(0) // vehicle id
		_ = buf.Write(mv.O)
	}

	if updateFlags&wow.UpdateFlagRotation != 0 {
		_ = buf.Write(int64(0)) // packed rotation
	}
}

// DefaultUnitSpeeds are the 3.3.5a base movement speeds.
func DefaultUnitSpeeds() [wow.MoveTypeMax]float32 {
	var s [wow.MoveTypeMax]float32
	s[wow.MoveTypeWalk] = 2.5
	s[wow.MoveTypeRun] = 7.0
	s[wow.MoveTypeRunBack] = 4.5
	s[wow.MoveTypeSwim] = 4.722222
	s[wow.MoveTypeSwimBack] = 2.5
	s[wow.MoveTypeFlight] = 7.0
	s[wow.MoveTypeFlightBack] = 4.5
	s[wow.MoveTypeTurnRate] = 3.141594

	return s
}
