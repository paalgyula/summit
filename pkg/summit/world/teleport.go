package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/areatrigger"
	"github.com/paalgyula/summit/pkg/wow"
)

// TeleportFlags defines teleport behavior flags.
type TeleportFlags uint32

const (
	TeleportToNotLeaveCombat      TeleportFlags = 0x01
	TeleportToNotLeaveTransport   TeleportFlags = 0x02
	TeleportToSpell               TeleportFlags = 0x04
	TeleportToNotLeaveCombatSpell TeleportFlags = TeleportToNotLeaveCombat | TeleportToSpell
)

// HandleAreaTriggerOpcode handles CMSG_AREATRIGGER - player enters an area trigger.
func (gc *WorldSession) HandleAreaTriggerOpcode(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var triggerID uint32
	_ = reader.Read(&triggerID)

	gc.log.Debug().Uint32("triggerID", triggerID).Msg("area trigger")

	server, ok := gc.ws.(*Server)
	if !ok || server.areaTriggerMgr == nil {
		return
	}

	// Check if this is a teleport trigger
	teleport := server.areaTriggerMgr.GetTeleport(triggerID)
	if teleport == nil {
		gc.log.Debug().Uint32("triggerID", triggerID).Msg("no teleport for trigger")
		return
	}

	// Check instance entry rights if cross-map
	if teleport.TargetMapID != gc.player.Location.Map {
		// TODO: Check instance entry rights
		gc.log.Debug().
			Uint32("from", gc.player.Location.Map).
			Uint32("to", teleport.TargetMapID).
			Msg("cross-map teleport via area trigger")
	}

	// Teleport the player
	gc.TeleportTo(teleport.TargetMapID, teleport.TargetX, teleport.TargetY, teleport.TargetZ, teleport.TargetO)
}

// TeleportTo teleports the player to the given location.
// This is the main teleport entry point that handles both same-map and cross-map teleports.
func (gc *WorldSession) TeleportTo(mapID uint32, x, y, z, o float32) {
	if gc.player == nil {
		return
	}

	oldMapID := gc.player.Location.Map

	// Same map teleport
	if oldMapID == mapID {
		gc.teleportNear(mapID, x, y, z, o)
		return
	}

	// Cross-map teleport
	gc.teleportFar(mapID, x, y, z, o)
}

// teleportNear handles same-map teleportation.
func (gc *WorldSession) teleportNear(mapID uint32, x, y, z, o float32) {
	if gc.player == nil {
		return
	}

	// Store old position for broadcast
	oldX := gc.player.Location.X
	oldY := gc.player.Location.Y
	oldZ := gc.player.Location.Z
	oldO := gc.player.Location.O

	// Update player position
	gc.player.Location.X = x
	gc.player.Location.Y = y
	gc.player.Location.Z = z
	gc.player.Location.O = o
	gc.player.Location.Map = mapID

	// Send MSG_MOVE_TELEPORT_ACK to the player
	gc.sendTeleportAck(x, y, z, o)

	// Broadcast MSG_MOVE_TELEPORT to other players
	gc.broadcastTeleport(oldX, oldY, oldZ, oldO, x, y, z, o)

	gc.log.Debug().
		Float32("x", x).Float32("y", y).Float32("z", z).
		Msg("teleported near")
}

// teleportFar handles cross-map teleportation.
func (gc *WorldSession) teleportFar(mapID uint32, x, y, z, o float32) {
	if gc.player == nil {
		return
	}

	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Send SMSG_TRANSFER_PENDING (shows loading screen)
	gc.sendTransferPending(mapID)

	// Remove player from current map
	if server.mapManager != nil {
		oldMap := server.mapManager.FindMap(gc.player.Location.Map, gc.player.CurrentInstanceID)
		if oldMap != nil {
			oldMap.RemovePlayer(gc.player.ID)
		}
	}

	// Update player position
	gc.player.Location.X = x
	gc.player.Location.Y = y
	gc.player.Location.Z = z
	gc.player.Location.O = o
	gc.player.Location.Map = mapID
	gc.player.CurrentMapID = mapID

	// Send SMSG_NEW_WORLD (client loads new map)
	gc.sendNewWorld(mapID, x, y, z, o)

	// Add player to new map
	if server.mapManager != nil {
		newMap := server.mapManager.CreateBaseMap(mapID)
		if newMap != nil {
			newMap.AddPlayer(gc.player)
			gc.player.SetMap(newMap)
		}
	}

	// Send initial packets for the new map
	gc.sendInitialPacketsAfterTeleport()

	gc.log.Debug().
		Uint32("mapID", mapID).
		Float32("x", x).Float32("y", y).Float32("z", z).
		Msg("teleported far")
}

// sendTeleportAck sends MSG_MOVE_TELEPORT_ACK to the client.
func (gc *WorldSession) sendTeleportAck(x, y, z, o float32) {
	pkt := wow.NewPacket(wow.MsgMoveTeleportAck)

	// Pack GUID
	_ = pkt.Write(gc.player.GUID())

	// Movement counter
	_ = pkt.Write(uint32(0))

	// Build movement info
	_ = pkt.Write(uint32(0)) // movement flags
	_ = pkt.Write(uint16(0)) // extra movement flags
	_ = pkt.Write(uint32(0)) // timestamp

	// Position
	_ = pkt.Write(x)
	_ = pkt.Write(y)
	_ = pkt.Write(z)
	_ = pkt.Write(o)

	gc.socket.Send(pkt)
}

// sendTransferPending sends SMSG_TRANSFER_PENDING to the client.
func (gc *WorldSession) sendTransferPending(mapID uint32) {
	pkt := wow.NewPacket(wow.ServerTransferPending)

	_ = pkt.Write(mapID)

	// Transport info (if on transport - not implemented yet)
	// _ = pkt.Write(transportEntry)
	// _ = pkt.Write(currentMapID)

	gc.socket.Send(pkt)
}

// sendNewWorld sends SMSG_NEW_WORLD to the client.
func (gc *WorldSession) sendNewWorld(mapID uint32, x, y, z, o float32) {
	pkt := wow.NewPacket(wow.ServerNewWorld)

	_ = pkt.Write(mapID)
	_ = pkt.Write(x)
	_ = pkt.Write(y)
	_ = pkt.Write(z)
	_ = pkt.Write(o)

	gc.socket.Send(pkt)
}

// broadcastTeleport broadcasts MSG_MOVE_TELEPORT to other players.
func (gc *WorldSession) broadcastTeleport(oldX, oldY, oldZ, oldO, newX, newY, newZ, newO float32) {
	if gc.player == nil {
		return
	}

	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	pkt := wow.NewPacket(wow.MsgMoveTeleport)

	// Pack GUID
	_ = pkt.Write(gc.player.GUID())

	// Movement flags
	_ = pkt.Write(uint32(0))
	_ = pkt.Write(uint16(0))

	// Timestamp
	_ = pkt.Write(uint32(0))

	// New position
	_ = pkt.Write(newX)
	_ = pkt.Write(newY)
	_ = pkt.Write(newZ)
	_ = pkt.Write(newO)

	// Send to all other sessions
	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
}

// sendInitialPacketsAfterTeleport sends the initial packets needed after a cross-map teleport.
func (gc *WorldSession) sendInitialPacketsAfterTeleport() {
	if gc.player == nil {
		return
	}

	// Send login verify world
	gc.sendLoginVerifyWorld(gc.player)

	// Send feature system status
	gc.sendFeatureSystemStatus()

	// Send learned dance moves
	gc.sendLearnedDanceMoves()

	// Send bindpoint update
	gc.sendBindPointUpdate(gc.player)

	// Send instance difficulty
	gc.sendInstanceDifficulty(gc.player)

	// Send initial spells
	gc.sendInitialSpells(gc.player)

	// Send action buttons
	gc.sendActionButtons(gc.player)

	// Send login set timespeed
	gc.sendLoginSetTimeSpeed()

	// Send self create update
	gc.sendCreateObjectSelf()

	// Send init world states
	gc.sendInitWorldStates(gc.player)
}

// HandleMoveWorldportAck handles MSG_MOVE_WORLDPORT_ACK - client ready after cross-map teleport.
func (gc *WorldSession) HandleMoveWorldportAck(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	gc.log.Debug().Msg("worldport ack received")

	// The client is ready, we can finalize the teleport
	// Any post-teleport logic can go here
}

// IsWithinDistance checks if two positions are within the given distance.
func IsWithinDistance(x1, y1, z1, x2, y2, z2, distance float32) bool {
	dx := x1 - x2
	dy := y1 - y2
	dz := z1 - z2

	dist := dx*dx + dy*dy + dz*dz
	maxDist := distance * distance

	return dist <= maxDist
}

// HandleTeleportAck handles the teleport acknowledgment from the client.
func (gc *WorldSession) HandleTeleportAck(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var ackGUID uint64
	_ = reader.Read(&ackGUID)

	var ackCounter uint32
	_ = reader.Read(&ackCounter)

	gc.log.Debug().Uint64("guid", ackGUID).Msg("teleport ack")
}

// getAreaTriggerForPlayer checks if the player is in any areatrigger.
func (gc *WorldSession) getAreaTriggerForPlayer() *areatrigger.AreaTrigger {
	if gc.player == nil {
		return nil
	}

	server, ok := gc.ws.(*Server)
	if !ok || server.areaTriggerMgr == nil {
		return nil
	}

	triggers := server.areaTriggerMgr.GetTriggersForMap(gc.player.Location.Map)

	for _, at := range triggers {
		if at.IsInside(gc.player.Location.X, gc.player.Location.Y, gc.player.Location.Z) {
			return at
		}
	}

	return nil
}
