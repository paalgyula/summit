package wow

type MoveType int

const (
	MoveTypeWalk MoveType = iota
	MoveTypeRun
	MoveTypeRunBack
	MoveTypeSwim
	MoveTypeSwimBack
	MoveTypeTurnRate
	MoveTypeFlight
	MoveTypeFlightBack
	MoveTypeMax
)

// const MoveTypeMax = MoveTypeFlightBack + 1

type MovementFlag uint32

const (
	MovementFlagNone            MovementFlag = 0x00000000
	MovementFlagForward         MovementFlag = 0x00000001
	MovementFlagBackward        MovementFlag = 0x00000002
	MovementFlagStrafeLeft      MovementFlag = 0x00000004
	MovementFlagStrafeRight     MovementFlag = 0x00000008
	MovementFlagTurnLeft        MovementFlag = 0x00000010
	MovementFlagTurnRight       MovementFlag = 0x00000020
	MovementFlagPitchUp         MovementFlag = 0x00000040
	MovementFlagPitchDown       MovementFlag = 0x00000080
	MovementFlagWalkMode        MovementFlag = 0x00000100 // Walking
	MovementFlagOnTransport     MovementFlag = 0x00000200 // Used for flying on some creatures
	MovementFlagLevitating      MovementFlag = 0x00000400
	MovementFlagRoot            MovementFlag = 0x00000800
	MovementFlagFalling         MovementFlag = 0x00001000
	MovementFlagFallingFar      MovementFlag = 0x00004000
	MovementFlagSwimming        MovementFlag = 0x00200000 // appears with fly flag also
	MovementFlagAscending       MovementFlag = 0x00400000 // swim up also
	MovementFlagCanFly          MovementFlag = 0x00800000
	MovementFlagFlying          MovementFlag = 0x01000000
	MovementFlagFlying2         MovementFlag = 0x02000000 // Actual flying mode
	MovementFlagSplineElevation MovementFlag = 0x04000000 // used for flight paths
	MovementFlagSplineEnabled   MovementFlag = 0x08000000 // used for flight paths
	MovementFlagWaterwalking    MovementFlag = 0x10000000 // prevent unit from falling through water
	MovementFlagSafeFall        MovementFlag = 0x20000000 // Feather Fall (spell)
	MovementFlagHover           MovementFlag = 0x40000000

	MovementFlagMoving MovementFlag = MovementFlagForward | MovementFlagBackward | MovementFlagStrafeLeft |
		MovementFlagStrafeRight | MovementFlagPitchUp | MovementFlagPitchDown | MovementFlagFalling |
		MovementFlagFallingFar | MovementFlagAscending | MovementFlagSplineElevation

	MovementFlagTurning MovementFlag = MovementFlagTurnLeft | MovementFlagTurnRight

	MovementFlagMaskMovingFly MovementFlag = MovementFlagFlying2 | MovementFlagAscending | MovementFlagCanFly
)

// TransportMovementInfo holds data related to movement on a transport.
type TransportMovementInfo struct {
	GUID GUID    // Transport GUID
	X, Y, Z, O float32 // Transport's own position
	Seat int8
	Time uint32
}

// JumpInfo holds data related to a jump.
type JumpInfo struct {
	Velocity    float32
	SinAngle    float32
	CosAngle    float32
	XYSpeed     float32
}

// MovementInfo captures all data from a client movement packet (MSG_MOVE_*).
// Reference: TrinityCore's MovementInfo struct.
type MovementInfo struct {
	UnitGUID    GUID // GUID of the unit that is moving (read from packet, but server uses authoritative one for broadcast)
	Flags       MovementFlag
	Timestamp   uint32  // Client's timestamp for this movement
	X, Y, Z, O  float32 // Mover's position and orientation

	HasTransportData bool
	Transport        TransportMovementInfo

	// Pitch is present if swimming or flying.
	// It might be part of orientation (O) in some representations or a separate field.
	// For simplicity, if the client sends it separately after main X,Y,Z,O based on flags,
	// it needs to be read.
	Pitch float32

	HasFallData bool
	FallTime    uint32

	HasJumpData bool // Often comes with FallData or specific jump opcodes
	Jump        JumpInfo

	HasSplineElevationData bool
	SplineElevation float32

	// TODO: Add fields for Spline data if MovementFlagSplineEnabled is set
	// This includes multiple points for the spline.
}

// ReadClientMovementInfo deserializes MovementInfo from a packet reader.
// This is complex as the presence of certain fields depends on the MovementFlags.
func ReadClientMovementInfo(r *PacketReader) (*MovementInfo, error) {
	info := &MovementInfo{}

	// Client movement packets usually start with the GUID of the mover,
	// but this function is expected to be called after the GUID has already been read.
	// info.UnitGUID would be set by the caller.

	if err := r.Read(&info.Flags); err != nil {
		return nil, err
	}

	// Optional: In some client versions or specific packets, an extra byte (often flags2 or similar)
	// might be present after MovementFlags. For 3.3.5a MSG_MOVE_*, it's usually not there.
	// var flags2 uint8
	// if err := r.Read(&flags2); err != nil { return nil, err }


	if err := r.Read(&info.Timestamp); err != nil {
		return nil, err
	}
	if err := r.Read(&info.X); err != nil {
		return nil, err
	}
	if err := r.Read(&info.Y); err != nil {
		return nil, err
	}
	if err := r.Read(&info.Z); err != nil {
		return nil, err
	}
	if err := r.Read(&info.O); err != nil {
		return nil, err
	}

	if info.Flags&MovementFlagOnTransport != 0 {
		info.HasTransportData = true
		var transportGUIDRaw uint64
		if err := r.ReadPackedGUID(&transportGUIDRaw); err != nil {
			return nil, err
		}
		info.Transport.GUID.Set(transportGUIDRaw)

		if err := r.Read(&info.Transport.X); err != nil {
			return nil, err
		}
		if err := r.Read(&info.Transport.Y); err != nil {
			return nil, err
		}
		if err := r.Read(&info.Transport.Z); err != nil {
			return nil, err
		}
		if err := r.Read(&info.Transport.O); err != nil {
			return nil, err
		}
		if err := r.Read(&info.Transport.Seat); err != nil {
			return nil, err
		}
		// Transport time is not part of the standard CMSG_MOVE_ON_TRANSPORT block in 3.3.5a
		// It's sent in MSG_MOVE_CHNG_TRANSPORT.
	}

	// Order of these conditional blocks matters and should match packet structure.
	// Typically, pitch comes before fall/jump data if present.
	if info.Flags&MovementFlagSwimming != 0 || info.Flags&MovementFlagFlying2 != 0 || info.Flags&MovementFlagFlying != 0 {
		if err := r.Read(&info.Pitch); err != nil {
			return nil, err
		}
	}

	if info.Flags&MovementFlagFalling != 0 { // Or MovementFlagFallingFar
		info.HasFallData = true
		if err := r.Read(&info.FallTime); err != nil {
			return nil, err
		}
		// Jump related data (zspeed, sin/cos angle, xyspeed) often accompanies falling flag in CMSG_MOVE_*.
		// This is also the structure for MSG_MOVE_JUMP.
		info.HasJumpData = true
		if err := r.Read(&info.Jump.Velocity); err != nil { // zspeed
			return nil, err
		}
		if err := r.Read(&info.Jump.SinAngle); err != nil {
			return nil, err
		}
		if err := r.Read(&info.Jump.CosAngle); err != nil {
			return nil, err
		}
		if err := r.Read(&info.Jump.XYSpeed); err != nil {
			return nil, err
		}
	}

	if info.Flags&MovementFlagSplineElevation != 0 {
		info.HasSplineElevationData = true
		if err := r.Read(&info.SplineElevation); err != nil {
			return nil, err
		}
	}

	// TODO: Handle MovementFlagSplineEnabled - this involves reading spline points.
	// if info.Flags&MovementFlagSplineEnabled != 0 { ... }


	// This parsing is a common structure. Specific opcodes might have slight variations
	// or append more data. Refer to TrinityCore's parsing for exact details per opcode if issues arise.
	return info, nil
}

// WriteServerMovementInfo serializes MovementInfo for broadcasting to other clients.
// This needs to precisely match the structure expected by the client for movement updates.
// 'opcode' is the original CMSG_MOVE_* opcode which is often relayed.
func WriteServerMovementInfo(w *Packet, info *MovementInfo, movedUnitGUID GUID, originalClientOpcode OpCode) {
	w.WritePackedGUID(movedUnitGUID)
	w.Write(info.Flags)
	// var flags2 uint8 // if applicable for the client version/packet
	// w.Write(flags2)
	w.Write(info.Timestamp)
	w.Write(info.X)
	w.Write(info.Y)
	w.Write(info.Z)
	w.Write(info.O)

	if info.Flags&MovementFlagOnTransport != 0 {
		w.WritePackedGUID(info.Transport.GUID)
		w.Write(info.Transport.X)
		w.Write(info.Transport.Y)
		w.Write(info.Transport.Z)
		w.Write(info.Transport.O)
		w.Write(info.Transport.Seat)
	}

	if info.Flags&MovementFlagSwimming != 0 || info.Flags&MovementFlagFlying2 != 0 || info.Flags&MovementFlagFlying != 0 {
		w.Write(info.Pitch)
	}

	if info.Flags&MovementFlagFalling != 0 { // Or MovementFlagFallingFar
		w.Write(info.FallTime)
		w.Write(info.Jump.Velocity)
		w.Write(info.Jump.SinAngle)
		w.Write(info.Jump.CosAngle)
		w.Write(info.Jump.XYSpeed)
	}

	if info.Flags&MovementFlagSplineElevation != 0 {
		w.Write(info.SplineElevation)
	}

	// Data specific to certain opcodes that might be relayed or used in constructing MONSTER_MOVE:
	// Example: If originalClientOpcode == MsgMoveJump, TC's MonsterMove packet (which can also represent player jumps to others)
	// would write the jump velocity, sin/cos angle, and xySpeed again here.
	// However, for simple relaying of CMSG_MOVE_*, this common structure is often sufficient as flags dictate content.
	// For a robust system, you might need to switch on originalClientOpcode or have more specialized write functions
	// if the broadcast format (e.g. SMSG_MONSTER_MOVE) differs significantly from CMSG_MOVE_*.

	// For SMSG_UPDATE_OBJECT blocks, the structure is different (see Object.WriteMovementUpdate).
}


// IsMoving returns true if any directional movement flag is set.
func (mf MovementFlag) IsMoving() bool {
	// Simplified: Check primary movement triggers. Does not include spline movement explicitly here.
	return mf&(MovementFlagForward|MovementFlagBackward|MovementFlagStrafeLeft|MovementFlagStrafeRight|MovementFlagFalling) != 0
}

// IsTurning returns true if any turning flag is set.
func (mf MovementFlag) IsTurning() bool {
	return mf&(MovementFlagTurnLeft|MovementFlagTurnRight|MovementFlagPitchUp|MovementFlagPitchDown) != 0
}

// IsOnTransport checks if the MovementFlagOnTransport is set.
// Note: TrinityCore uses 0x00000200 for transport in CMSG_MOVE packets,
// but also has 0x00004000 in its general MovementFlags enum.
// The CMSG_MOVE_ONTRANSPORT flag is 0x00000200.
func (mf MovementFlag) IsOnTransport() bool {
	return mf&MovementFlagOnTransport != 0 // This is the flag from CMSG_MOVE_*
}
