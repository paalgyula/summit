package world

import (
	"time"

	mapmanager "github.com/paalgyula/summit/pkg/summit/world/map"
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
	p.Sender = gc

	// Set up broadcast function for health/power updates
	p.BroadcastPacket = func(pkt *wow.Packet) {
		server, ok := gc.ws.(*Server)
		if !ok {
			return
		}
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && other.player.IsInWorld {
				other.socket.Send(pkt)
			}
		}
	}

	// Initialize player values (health, power, display, update fields)
	p.Init()

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
	// Send bind point update (homebind)
	gc.sendBindPointUpdate(p)

	// Send instance difficulty
	gc.sendInstanceDifficulty(p)

	// Send initial spells (empty for now)
	gc.sendInitialSpells(p)

	// Send action buttons
	gc.sendActionButtons(p)

	// Send login set time speed
	gc.sendLoginSetTimeSpeed()

	// Send initial talents (empty for now)
	gc.sendInitialTalents(p)
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

	_ = pkt.Write(uint32(0))           // Game time placeholder
	_ = pkt.Write(float32(0.01666667)) // Game speed (1/60)
	_ = pkt.Write(uint32(0))           // Added in 3.1.2

	gc.socket.Send(pkt)
}

// addPlayerToMap adds the player to the world map.
// This mirrors Map::AddPlayerToMap in AzerothCore: adds to grid, then
// triggers a visibility pass so nearby objects are sent to the client.
func (gc *WorldSession) addPlayerToMap(p *player.Player) {
	// Mark player as in world
	p.IsInWorld = true

	// Add player to the map for update queue tracking and grid visibility
	server, ok := gc.ws.(*Server)
	if ok && server.mapManager != nil {
		m := server.mapManager.CreateBaseMap(p.Location.Map)
		m.AddPlayer(p)
		// Trigger initial visibility update through the grid system —
		// this sends create packets for nearby players, NPCs, and objects.
		m.UpdatePlayerVisibility(p)
	}

	// Send nearby players to the new player, and the new player to nearby players
	gc.sendNearbyPlayers(p)

	gc.log.Debug().Str("name", p.Name).Msg("player added to map")
}

// sendNearbyPlayers sends CreateObject for nearby players only.
// The new player receives create blocks for players within sight range,
// and existing players receive a create block for the new player.
// NPCs and game objects are handled by the map's grid-based visibility
// system (UpdatePlayerVisibility), not here.
func (gc *WorldSession) sendNearbyPlayers(p *player.Player) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	sightRange := mapmanager.DefaultVisibilityDistance

	for _, other := range server.GetOtherSessions(gc) {
		if other.player == nil || !other.player.IsInWorld {
			continue
		}

		// Only send players within visibility range
		dist := mapmanager.Distance2DPositions(
			p.Location.X, p.Location.Y,
			other.player.Location.X, other.player.Location.Y,
		)

		if dist <= sightRange {
			// Send the existing player to the new player
			gc.sendCreateObjectForPlayer(other.player, p)

			// Send the new player to the existing player
			other.sendCreateObjectForPlayer(p, other.player)
		}
	}
}

// sendCreateObjectForPlayer sends SMSG_UPDATE_OBJECT with a create block
// for the given player to the target session.
func (gc *WorldSession) sendCreateObjectForPlayer(source *player.Player, target *player.Player) {
	upd := &Updater{}
	pkt := upd.BuildCreateObject(source, target)
	gc.socket.Send(pkt)
}

// sendCreateObjectForNPC sends SMSG_UPDATE_OBJECT with a create block for an NPC.
func (gc *WorldSession) sendCreateObjectForNPC(npc *NPC) {
	pkt := BuildNPCCreateObject(npc)
	gc.socket.Send(pkt)
}

// sendInitialPacketsAfterAddToMap sends all required packets after adding player to map.
// This mirrors Player::SendInitialPacketsAfterAddToMap in AzerothCore.
func (gc *WorldSession) sendInitialPacketsAfterAddToMap(p *player.Player) {
	// Reset time sync
	gc.resetTimeSync()

	// Send initial world states
	gc.sendInitWorldStates(p)

	// Rebuild the client quest log from the character's persisted quests.
	gc.rebuildQuestLog()

	// Send player create update to self (SMSG_UPDATE_OBJECT with player values)
	gc.sendPlayerCreate(p)
}

// sendCreateObjectSelf sends the player's own create object update.
func (gc *WorldSession) sendCreateObjectSelf() {
	if gc.player == nil {
		return
	}

	gc.sendPlayerCreate(gc.player)
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

// sendInitialSpells sends SMSG_INITIAL_SPELLS with the player's known spells.
func (gc *WorldSession) sendInitialSpells(p *player.Player) {
	// 3.3.5a layout: u8 talent spec, u16 count, (u32 spell, u16 unk) × count, u16 cooldown count
	pkt := wow.NewPacket(wow.ServerInitialSpells)

	_ = pkt.WriteOne(0)
	_ = pkt.Write(uint16(len(p.KnownSpells)))

	for _, spellID := range p.KnownSpells {
		_ = pkt.Write(uint32(spellID))
		_ = pkt.Write(uint16(0))
	}

	_ = pkt.Write(uint16(0)) // no cooldowns

	gc.socket.Send(pkt)
}

// sendActionButtons sends SMSG_ACTION_BUTTONS with the player's action bar.
func (gc *WorldSession) sendActionButtons(p *player.Player) {
	// 3.3.5a layout: u8 packet type (1 = initial), then MAX_ACTION_BUTTONS (144) packed
	// buttons: action id in the low 24 bits, type in the top byte
	pkt := wow.NewPacket(wow.ServerActionButtons)

	_ = pkt.WriteOne(1)

	for i := 0; i < MaxActionButtons; i++ {
		var button uint32

		if i < len(p.Actions) {
			button = p.Actions[i]
		}

		_ = pkt.Write(button)
	}

	gc.socket.Send(pkt)
}

// sendInitialTalents sends SMSG_INITIALIZE_FACTIONS with empty talent data.
// A proper implementation would send SMSG_TALENT_INFO.
func (gc *WorldSession) sendInitialTalents(_ *player.Player) {
	// Talent info is sent separately; player create is handled in sendInitialPacketsAfterAddToMap.
}

// sendPlayerCreate sends the initial SMSG_UPDATE_OBJECT for the player's own creation.
func (gc *WorldSession) sendPlayerCreate(p *player.Player) {
	upd := &Updater{}
	pkt := upd.BuildSelfCreateObject(p)
	gc.socket.Send(pkt)
}

// sendCreateObjectForGameObject sends SMSG_UPDATE_OBJECT with a create block for a game object.
func (gc *WorldSession) sendCreateObjectForGameObject(gobj *GameObject) {
	pkt := BuildGameObjectCreateObject(gobj)
	gc.socket.Send(pkt)
}
