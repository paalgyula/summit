package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// MovementInfo contains parsed movement data from client packets.
type MovementInfo struct {
	GUID     wow.GUID
	Flags    uint32
	Flags2   uint16
	Time     uint32
	Position player.WorldLocation

	// Transport info
	TransportGUID wow.GUID
	TransportPos  player.WorldLocation
	TransportTime uint32
	TransportSeat int8

	// Swimming/flying
	Pitch float32

	// Falling
	FallTime uint32

	// Jumping
	JumpInfo JumpInfo

	// Spline
	SplineElevation float32
}

// JumpInfo contains jump-related movement data.
type JumpInfo struct {
	ZSpeed   float32
	SinAngle float32
	CosAngle float32
	XYSpeed  float32
}

// ReadMovementInfo reads movement info from a packet reader.
func ReadMovementInfo(reader *wow.PacketReader, guid wow.GUID) *MovementInfo {
	info := &MovementInfo{
		GUID: guid,
	}

	// Read movement flags (4 bytes)
	_ = reader.Read(&info.Flags)

	// Read extra movement flags (2 bytes)
	_ = reader.Read(&info.Flags2)

	// Read timestamp (4 bytes)
	_ = reader.Read(&info.Time)

	// Read position (X, Y, Z, O)
	_ = reader.Read(&info.Position.X)
	_ = reader.Read(&info.Position.Y)
	_ = reader.Read(&info.Position.Z)
	_ = reader.Read(&info.Position.O)

	// Read transport info if MOVEMENTFLAG_ONTRANSPORT is set
	if info.Flags&uint32(wow.MovementFlagOnTransport) != 0 {
		var transportGUID uint64
		_ = reader.Read(&transportGUID)
		info.TransportGUID = wow.GUID(transportGUID)

		_ = reader.Read(&info.TransportPos.X)
		_ = reader.Read(&info.TransportPos.Y)
		_ = reader.Read(&info.TransportPos.Z)
		_ = reader.Read(&info.TransportPos.O)
		_ = reader.Read(&info.TransportTime)
		_ = reader.Read(&info.TransportSeat)
	}

	// Read swim/fly pitch if MOVEMENTFLAG_SWIMMING or MOVEMENTFLAG_FLYING is set
	if info.Flags&uint32(wow.MovementFlagSwimming) != 0 || info.Flags&uint32(wow.MovementFlagFlying) != 0 {
		_ = reader.Read(&info.Pitch)
	}

	// Read fall time
	if info.Flags&uint32(wow.MovementFlagFalling) != 0 {
		_ = reader.Read(&info.FallTime)
	}

	// Read jump info if MOVEMENTFLAG_FALLING is set
	if info.Flags&uint32(wow.MovementFlagFalling) != 0 {
		_ = reader.Read(&info.JumpInfo.ZSpeed)
		_ = reader.Read(&info.JumpInfo.SinAngle)
		_ = reader.Read(&info.JumpInfo.CosAngle)
		_ = reader.Read(&info.JumpInfo.XYSpeed)
	}

	// Read spline elevation if MOVEMENTFLAG_SPLINE_ELEVATION is set
	if info.Flags&uint32(wow.MovementFlagSplineElevation) != 0 {
		_ = reader.Read(&info.SplineElevation)
	}

	return info
}

// WriteMovementInfo writes movement info to a packet.
func WriteMovementInfo(pkt *wow.Packet, info *MovementInfo) {
	_ = pkt.Write(info.Flags)
	_ = pkt.Write(info.Flags2)
	_ = pkt.Write(info.Time)

	_ = pkt.Write(info.Position.X)
	_ = pkt.Write(info.Position.Y)
	_ = pkt.Write(info.Position.Z)
	_ = pkt.Write(info.Position.O)

	// Write transport info if MOVEMENTFLAG_ONTRANSPORT is set
	if info.Flags&uint32(wow.MovementFlagOnTransport) != 0 {
		_ = pkt.Write(uint64(info.TransportGUID))
		_ = pkt.Write(info.TransportPos.X)
		_ = pkt.Write(info.TransportPos.Y)
		_ = pkt.Write(info.TransportPos.Z)
		_ = pkt.Write(info.TransportPos.O)
		_ = pkt.Write(info.TransportTime)
		_ = pkt.Write(info.TransportSeat)
	}

	// Write swim/fly pitch
	if info.Flags&uint32(wow.MovementFlagSwimming) != 0 || info.Flags&uint32(wow.MovementFlagFlying) != 0 {
		_ = pkt.Write(info.Pitch)
	}

	// Write fall time
	if info.Flags&uint32(wow.MovementFlagFalling) != 0 {
		_ = pkt.Write(info.FallTime)
	}

	// Write jump info
	if info.Flags&uint32(wow.MovementFlagFalling) != 0 {
		_ = pkt.Write(info.JumpInfo.ZSpeed)
		_ = pkt.Write(info.JumpInfo.SinAngle)
		_ = pkt.Write(info.JumpInfo.CosAngle)
		_ = pkt.Write(info.JumpInfo.XYSpeed)
	}

	// Write spline elevation
	if info.Flags&uint32(wow.MovementFlagSplineElevation) != 0 {
		_ = pkt.Write(info.SplineElevation)
	}
}

// HandleMovementOpcodes handles all movement opcodes from the client.
// This is the main entry point for processing movement packets.
func (gc *WorldSession) HandleMovementOpcodes(opcode wow.OpCode, data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	// Read packed GUID
	var packedGUID uint64
	_ = reader.Read(&packedGUID)
	guid := wow.GUID(packedGUID)

	// Verify the GUID matches our player
	if guid != gc.player.GUID() {
		gc.log.Warn().
			Str("expected", fmt.Sprintf("0x%x", uint64(gc.player.GUID()))).
			Str("got", fmt.Sprintf("0x%x", uint64(guid))).
			Msg("movement GUID mismatch")
		return
	}

	// Read movement info
	movementInfo := ReadMovementInfo(reader, guid)

	// Update player position
	gc.updatePlayerPosition(movementInfo)

	// Broadcast movement to other players
	gc.broadcastMovement(opcode, movementInfo)
}

// updatePlayerPosition updates the player's position from movement info.
func (gc *WorldSession) updatePlayerPosition(info *MovementInfo) {
	gc.player.Location.X = info.Position.X
	gc.player.Location.Y = info.Position.Y
	gc.player.Location.Z = info.Position.Z
	gc.player.Location.O = info.Position.O

	// Update movement flags on player
	gc.player.MoveFlags = wow.MovementFlag(info.Flags)
}

// broadcastMovement broadcasts the movement to other players in range.
func (gc *WorldSession) broadcastMovement(opcode wow.OpCode, info *MovementInfo) {
	// Create the movement packet
	pkt := wow.NewPacket(opcode)
	WriteMovementInfo(pkt, info)

	// TODO: Implement SendMessageToSet - broadcast to nearby players
	// For now, just log the movement
	gc.log.Trace().
		Str("opcode", opcode.String()).
		Float32("x", info.Position.X).
		Float32("y", info.Position.Y).
		Float32("z", info.Position.Z).
		Uint32("flags", info.Flags).
		Msg("movement received")
}

// HandleMovementSpeed handles MSG_MOVE_SET_RUN_SPEED and similar speed change packets.
func (gc *WorldSession) HandleMovementSpeed(opcode wow.OpCode, data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	// Read movement info
	var packedGUID uint64
	_ = reader.Read(&packedGUID)
	guid := wow.GUID(packedGUID)

	if guid != gc.player.GUID() {
		return
	}

	movementInfo := ReadMovementInfo(reader, guid)

	// Read new speed
	var speed float32
	_ = reader.Read(&speed)

	// Update player speed based on opcode
	// TODO: Store speed on player object

	gc.log.Trace().
		Str("opcode", opcode.String()).
		Float32("speed", speed).
		Msg("speed change received")

	// Broadcast speed change to other players
	pkt := wow.NewPacket(opcode)
	WriteMovementInfo(pkt, movementInfo)
	_ = pkt.Write(speed)

	// TODO: SendMessageToSet
}

// HandleMoveFallReset handles CMSG_MOVE_FALL_RESET.
func (gc *WorldSession) HandleMoveFallReset(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var packedGUID uint64
	_ = reader.Read(&packedGUID)
	guid := wow.GUID(packedGUID)

	if guid != gc.player.GUID() {
		return
	}

	movementInfo := ReadMovementInfo(reader, guid)

	// Reset fall distance
	gc.player.Location.Z = movementInfo.Position.Z

	gc.log.Trace().Msg("fall reset received")
}

// HandleMoveTimeSkipped handles CMSG_MOVE_TIME_SKIPPED.
func (gc *WorldSession) HandleMoveTimeSkipped(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var packedGUID uint64
	_ = reader.Read(&packedGUID)
	guid := wow.GUID(packedGUID)

	if guid != gc.player.GUID() {
		return
	}

	var skippedTime uint32
	_ = reader.Read(&skippedTime)

	gc.log.Trace().Uint32("skipped", skippedTime).Msg("time skipped")
}
