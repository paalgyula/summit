package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Friend status constants
const (
	FriendStatusOffline = 0
	FriendStatusOnline  = 1
	FriendStatusAway    = 2
	FriendStatusBusy    = 3
)

// HandlePlayerLogin handles the CMSG_PLAYER_LOGIN opcode.
// This is the main entry point when a player selects a character to enter the world.
//
// Flow (matching AzerothCore):
// 1. Read player GUID from packet
// 2. Load player from database
// 3. Send initial packets before add to map
// 4. Add player to map
// 5. Send initial packets after add to map
func (gc *WorldSession) HandlePlayerLogin(data wow.PacketData) {
	reader := wow.NewPacketReader(data)

	var playerGUID uint64
	if err := reader.Read(&playerGUID); err != nil {
		gc.log.Error().Err(err).Msg("failed to read player GUID")
		return
	}

	guid := wow.GUID(playerGUID)
	gc.log.Info().Uint32("guid", guid.Counter()).Msg("player login request")

	// Load player from database
	p, err := gc.ws.GetCharacter(guid.Counter())
	if err != nil {
		gc.log.Error().Err(err).Msg("failed to load player from database")
		gc.sendCharacterLoginFailed()
		return
	}

	if p == nil {
		gc.log.Error().Uint32("guid", guid.Counter()).Msg("player not found")
		gc.sendCharacterLoginFailed()
		return
	}

	// Store player in session
	gc.player = p

	// Send login verify world
	gc.sendLoginVerifyWorld(p)

	// Send feature system status
	gc.sendFeatureSystemStatus()

	// Send MOTD (TODO: implement MOTD system)
	gc.sendMOTD()

	// Send learned dance moves (empty)
	gc.sendLearnedDanceMoves()

	// Send initial packets before add to map
	gc.sendInitialPacketsBeforeAddToMap(p)

	// Add player to map (simplified - just mark as in world)
	gc.addPlayerToMap(p)

	// Send initial packets after add to map
	gc.sendInitialPacketsAfterAddToMap(p)

	// Set player online in database
	gc.setPlayerOnline(p)

	// Announce to friends
	gc.sendFriendStatus(p, FriendStatusOnline)

	// Send group update if in group
	if p.GroupID > 0 {
		gc.sendGroupUpdate(p)
	}

	gc.log.Info().Str("name", p.Name).Msg("player logged in")
}

// sendLoginVerifyWorld sends SMSG_LOGIN_VERIFY_WORLD to confirm the player's position.
func (gc *WorldSession) sendLoginVerifyWorld(p *player.Player) {
	pkt := wow.NewPacket(wow.ServerLoginVerifyWorld)

	_ = pkt.Write(p.Location.Map)
	_ = pkt.Write(p.Location.X)
	_ = pkt.Write(p.Location.Y)
	_ = pkt.Write(p.Location.Z)
	_ = pkt.Write(p.Location.O)

	gc.socket.Send(pkt)
}

// sendFeatureSystemStatus sends SMSG_FEATURE_SYSTEM_STATUS.
func (gc *WorldSession) sendFeatureSystemStatus() {
	pkt := wow.NewPacket(wow.ServerFeatureSystemStatus)

	_ = pkt.WriteOne(2) // COMPLAINT_ENABLED_WITH_AUTO_IGNORE
	_ = pkt.WriteOne(0) // voice chat disabled

	gc.socket.Send(pkt)
}

// sendMOTD sends the message of the day.
func (gc *WorldSession) sendMOTD() {
	// TODO: Implement MOTD system
	// For now, send empty message
}

// sendLearnedDanceMoves sends SMSG_LEARNED_DANCE_MOVES.
func (gc *WorldSession) sendLearnedDanceMoves() {
	pkt := wow.NewPacket(wow.ServerLearnedDanceMoves)

	_ = pkt.Write(uint32(0))
	_ = pkt.Write(uint32(0))

	gc.socket.Send(pkt)
}

// sendInitialPacketsBeforeAddToMap sends all required packets before adding player to map.
// This mirrors Player::SendInitialPacketsBeforeAddToMap in AzerothCore.
func (gc *WorldSession) sendInitialPacketsBeforeAddToMap(p *player.Player) {
	// Send social list
	// TODO: Implement social list

	// Send bind point update (homebind)
	gc.sendBindPointUpdate(p)

	// Send talents
	// TODO: Implement talents

	// Send instance difficulty
	gc.sendInstanceDifficulty(p)

	// Send initial spells
	// TODO: Implement spells

	// Send action buttons
	// TODO: Implement action buttons

	// Send reputations
	// TODO: Implement reputations

	// Send achievements
	// TODO: Implement achievements

	// Send equipment set list
	// TODO: Implement equipment sets

	// Send login set time speed
	gc.sendLoginSetTimeSpeed()

	// Send forced reactions
	// TODO: Implement forced reactions
}

// sendBindPointUpdate sends SMSG_BINDPOINTUPDATE.
func (gc *WorldSession) sendBindPointUpdate(p *player.Player) {
	pkt := wow.NewPacket(wow.ServerBindpointupdate)

	_ = pkt.Write(p.BindLocation.X)
	_ = pkt.Write(p.BindLocation.Y)
	_ = pkt.Write(p.BindLocation.Z)
	_ = pkt.Write(p.BindLocation.Map)
	_ = pkt.Write(p.BindLocation.Zone)

	gc.socket.Send(pkt)
}

// sendInstanceDifficulty sends SMSG_INSTANCE_DIFFICULTY.
func (gc *WorldSession) sendInstanceDifficulty(p *player.Player) {
	pkt := wow.NewPacket(wow.ServerInstanceDifficulty)

	_ = pkt.Write(uint32(0)) // Normal difficulty
	_ = pkt.Write(uint32(0)) // Not dynamic

	gc.socket.Send(pkt)
}

// sendLoginSetTimeSpeed sends SMSG_LOGIN_SETTIMESPEED.
func (gc *WorldSession) sendLoginSetTimeSpeed() {
	pkt := wow.NewPacket(wow.ServerLoginSettimespeed)

	// Game time (packed)
	_ = time.Now()

	_ = pkt.Write(uint32(0)) // Game time placeholder
	_ = pkt.Write(float32(0.01666667)) // Game speed (1/60)
	_ = pkt.Write(uint32(0)) // Added in 3.1.2

	gc.socket.Send(pkt)
}

// addPlayerToMap adds the player to the world map.
// This is a simplified version of Map::AddPlayerToMap.
func (gc *WorldSession) addPlayerToMap(p *player.Player) {
	// Mark player as in world
	p.IsInWorld = true

	// TODO: Add to grid/cell system
	// TODO: Load nearby grids
	// TODO: Send transport info
	// TODO: Send self update packet
	// TODO: Update object visibility

	gc.log.Debug().Str("name", p.Name).Msg("player added to map")
}

// sendInitialPacketsAfterAddToMap sends all required packets after adding player to map.
// This mirrors Player::SendInitialPacketsAfterAddToMap in AzerothCore.
func (gc *WorldSession) sendInitialPacketsAfterAddToMap(p *player.Player) {
	// Update visibility for player
	// TODO: Implement visibility system

	// Reset time sync
	gc.resetTimeSync()

	// Cast login effect spell (836)
	// TODO: Implement spell casting

	// Re-apply aura effects
	// TODO: Implement aura system

	// Update zone/area
	gc.sendInitWorldStates(p)

	// Send enchantment durations
	// TODO: Implement enchantments

	// Send item durations
	// TODO: Implement items

	// Send quest giver status
	// TODO: Implement quests

	// Send taxi node status
	// TODO: Implement taxi system

	// Send raid difficulty
	// TODO: Implement raid difficulty
}

// resetTimeSync resets the time sync counter.
func (gc *WorldSession) resetTimeSync() {
	gc.timeSyncCounter = 0
}

// sendInitWorldStates sends SMSG_INIT_WORLD_STATES.
func (gc *WorldSession) sendInitWorldStates(p *player.Player) {
	pkt := wow.NewPacket(wow.ServerInitWorldStates)

	_ = pkt.Write(p.Location.Map)
	_ = pkt.Write(p.Location.Zone)
	_ = pkt.Write(uint32(0)) // Area

	// World states count
	_ = pkt.Write(uint32(0))

	gc.socket.Send(pkt)
}

// setPlayerOnline sets the player as online in the database.
func (gc *WorldSession) setPlayerOnline(p *player.Player) {
	// TODO: Update database
	gc.log.Debug().Str("name", p.Name).Msg("player marked as online")
}

// sendFriendStatus notifies friends about player online status.
func (gc *WorldSession) sendFriendStatus(p *player.Player, status uint8) {
	// TODO: Implement friend list
	_ = p
	_ = status
}

// sendGroupUpdate sends group update to the player.
func (gc *WorldSession) sendGroupUpdate(p *player.Player) {
	// TODO: Implement group system
	_ = p
}

// sendCharacterLoginFailed sends SMSG_CHARACTER_LOGIN_FAILED.
func (gc *WorldSession) sendCharacterLoginFailed() {
	pkt := wow.NewPacket(wow.ServerCharacterLoginFailed)

	_ = pkt.WriteOne(0) // Login failed

	gc.socket.Send(pkt)
}
