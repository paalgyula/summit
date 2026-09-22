package world

import (
	"fmt"
	"math"

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

	// Moving ends dances, sitting and the like
	if info.Flags&uint32(wow.MovementFlagMoving) != 0 {
		gc.clearEmotes()
		// A moving caster cannot keep a cast-time spell going: the client's own
		// prediction drops the cast, the server confirms it with SPELL_FAILURE.
		gc.interruptCast(SpellCastFailedSpellInterrupted)
	}

	// Check areatriggers after position update
	gc.checkAreaTriggers()
}

// broadcastMovement broadcasts the movement to other players in range.
func (gc *WorldSession) broadcastMovement(opcode wow.OpCode, info *MovementInfo) {
	// Create the movement packet
	pkt := wow.NewPacket(opcode)
	_ = pkt.Write(info.GUID)
	WriteMovementInfo(pkt, info)

	// Send to all other sessions
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
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

	server, ok := gc.ws.(*Server)
	if ok {
		for _, other := range server.GetOtherSessions(gc) {
			if other.player != nil && other.player.IsInWorld {
				other.socket.Send(pkt)
			}
		}
	}
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

// checkAreaTriggers checks if the player has entered any areatriggers.
func (gc *WorldSession) checkAreaTriggers() {
	if gc.player == nil {
		return
	}

	server, ok := gc.ws.(*Server)
	if !ok || server.areaTriggerMgr == nil {
		return
	}

	// Get areatriggers for current map
	triggers := server.areaTriggerMgr.GetTriggersForMap(gc.player.Location.Map)

	for _, at := range triggers {
		// Check if player is inside the trigger
		if at.IsInside(gc.player.Location.X, gc.player.Location.Y, gc.player.Location.Z) {
			// Check if script is registered
			server.areaTriggerMgr.CheckTrigger(
				gc.player.ID,
				at.Entry,
				gc.player.Location.X,
				gc.player.Location.Y,
				gc.player.Location.Z,
			)
		}
	}
}

// MonsterMoveType defines the orientation / stop type for SMSG_MONSTER_MOVE.
type MonsterMoveType uint8

const (
	MonsterMoveNormal       MonsterMoveType = 0
	MonsterMoveStop         MonsterMoveType = 1
	MonsterMoveFacingSpot   MonsterMoveType = 2
	MonsterMoveFacingTarget MonsterMoveType = 3
	MonsterMoveFacingAngle  MonsterMoveType = 4
)

// Spline flags (MoveSplineFlag).
const (
	SplineFlagNone       uint32 = 0x00000000
	SplineFlagRunMode    uint32 = 0x00000000
	SplineFlagWalkMode   uint32 = 0x00000100
	SplineFlagFlying     uint32 = 0x00000200
	SplineFlagCatmullRom uint32 = 0x00020000
)

// BuildMonsterMovePacket builds an SMSG_MONSTER_MOVE packet to move an NPC to dest.
func BuildMonsterMovePacket(guid wow.GUID, startX, startY, startZ, destX, destY, destZ float32, splineID, durationMs, splineFlags uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerMonsterMove)

	pkt.WriteBytes(guid.Pack())
	_ = pkt.Write(uint8(0)) // toggle byte (sets/unsets MOVEMENTFLAG2_UNK7)
	_ = pkt.Write(startX)
	_ = pkt.Write(startY)
	_ = pkt.Write(startZ)
	_ = pkt.Write(splineID)
	_ = pkt.Write(uint8(MonsterMoveNormal))
	_ = pkt.Write(splineFlags)
	_ = pkt.Write(durationMs)
	_ = pkt.Write(uint32(1)) // 1 waypoint
	_ = pkt.Write(destX)
	_ = pkt.Write(destY)
	_ = pkt.Write(destZ)

	return pkt
}

// BuildMonsterMoveStopPacket builds an SMSG_MONSTER_MOVE packet that stops NPC movement.
func BuildMonsterMoveStopPacket(guid wow.GUID, x, y, z float32, splineID uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerMonsterMove)

	pkt.WriteBytes(guid.Pack())
	_ = pkt.Write(uint8(0))
	_ = pkt.Write(x)
	_ = pkt.Write(y)
	_ = pkt.Write(z)
	_ = pkt.Write(splineID)
	_ = pkt.Write(uint8(MonsterMoveStop))

	return pkt
}

// BuildMonsterMoveSplinePacket builds an SMSG_MONSTER_MOVE packet with multiple spline points.
func BuildMonsterMoveSplinePacket(guid wow.GUID, startX, startY, startZ float32, points [][3]float32, splineID, durationMs, splineFlags uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerMonsterMove)

	pkt.WriteBytes(guid.Pack())
	_ = pkt.Write(uint8(0)) // toggle byte
	_ = pkt.Write(startX)
	_ = pkt.Write(startY)
	_ = pkt.Write(startZ)
	_ = pkt.Write(splineID)
	_ = pkt.Write(uint8(MonsterMoveNormal))
	_ = pkt.Write(splineFlags)
	_ = pkt.Write(durationMs)
	_ = pkt.Write(uint32(len(points))) // waypoint count

	for _, p := range points {
		_ = pkt.Write(p[0]) // X
		_ = pkt.Write(p[1]) // Y
		_ = pkt.Write(p[2]) // Z
	}

	return pkt
}

// CatmullRomInterpolate generates smooth spline points using Catmull-Rom interpolation.
// p0, p1, p2, p3 are the four control points, t is the interpolation parameter [0, 1].
func CatmullRomInterpolate(p0, p1, p2, p3 [3]float32, t float32) [3]float32 {
	t2 := t * t
	t3 := t2 * t

	result := [3]float32{
		0.5 * ((2*p1[0]) +
			(-p0[0]+p2[0])*t +
			(2*p0[0]-5*p1[0]+4*p2[0]-p3[0])*t2 +
			(-p0[0]+3*p1[0]-3*p2[0]+p3[0])*t3),
		0.5 * ((2*p1[1]) +
			(-p0[1]+p2[1])*t +
			(2*p0[1]-5*p1[1]+4*p2[1]-p3[1])*t2 +
			(-p0[1]+3*p1[1]-3*p2[1]+p3[1])*t3),
		0.5 * ((2*p1[2]) +
			(-p0[2]+p2[2])*t +
			(2*p0[2]-5*p1[2]+4*p2[2]-p3[2])*t2 +
			(-p0[2]+3*p1[2]-3*p2[2]+p3[2])*t3),
	}

	return result
}

// GenerateCatmullRomSpline generates a smooth spline path through waypoints using Catmull-Rom interpolation.
// The path will loop if looping is true.
func GenerateCatmullRomSpline(waypoints [][3]float32, pointsPerSegment int, looping bool) [][3]float32 {
	if len(waypoints) < 2 {
		return waypoints
	}

	var result [][3]float32

	numWaypoints := len(waypoints)
	numSegments := numWaypoints
	if !looping {
		numSegments = numWaypoints - 1
	}

	for i := 0; i < numSegments; i++ {
		// Get four control points for Catmull-Rom
		p0 := waypoints[(i-1+numWaypoints)%numWaypoints]
		p1 := waypoints[i]
		p2 := waypoints[(i+1)%numWaypoints]
		p3 := waypoints[(i+2)%numWaypoints]

		// Generate points along this segment
		for j := 0; j < pointsPerSegment; j++ {
			t := float32(j) / float32(pointsPerSegment)
			point := CatmullRomInterpolate(p0, p1, p2, p3, t)
			result = append(result, point)
		}
	}

	// Add the final point if not looping
	if !looping {
		result = append(result, waypoints[numWaypoints-1])
	}

	return result
}

// Distance3D calculates the Euclidean distance between two 3D points.
func Distance3D(x1, y1, z1, x2, y2, z2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}
