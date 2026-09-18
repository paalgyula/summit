package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
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

	// Send existing players to the new player, and the new player to existing players
	gc.sendVisiblePlayers(p)

	gc.log.Debug().Str("name", p.Name).Msg("player added to map")
}

// sendVisiblePlayers handles visibility: sends CreateObject for existing players
// to the new player, and sends CreateObject for the new player to existing players.
// Also sends all NPCs to the new player.
func (gc *WorldSession) sendVisiblePlayers(p *player.Player) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Send existing players to the new player, and the new player to existing players
	for _, other := range server.GetOtherSessions(gc) {
		if other.player == nil || !other.player.IsInWorld {
			continue
		}

		// Send the existing player to the new player
		gc.sendCreateObjectForPlayer(other.player, p)

		// Send the new player to the existing player
		other.sendCreateObjectForPlayer(p, other.player)
	}

	// Send all NPCs to the new player
	for _, npc := range server.spawns.GetNPCsInMap(p.Location.Map) {
		gc.sendCreateObjectForNPC(npc)
	}

	// Send all game objects to the new player
	for _, gobj := range server.gameObjects.GetObjectsInMap(p.Location.Map) {
		gc.sendCreateObjectForGameObject(gobj)
	}
}

// sendCreateObjectForPlayer sends SMSG_UPDATE_OBJECT with a create block
// for the given player to the target session.
func (gc *WorldSession) sendCreateObjectForPlayer(source *player.Player, target *player.Player) {
	upd := &Updater{}

	// Build update flags for the source player
	flags := uint8(wow.UpdateFlagLowGUID | wow.UpdateFlagHighGUID | wow.UpdateFlagLiving | wow.UpdateFlagHasPosition)
	if source.GUID() == target.GUID() {
		flags |= wow.UpdateFlagSelf
	}
	upd.updateFlags = flags

	pkt := upd.BuildUpdateObject(source)
	gc.socket.Send(pkt)
}

// sendCreateObjectForNPC sends SMSG_UPDATE_OBJECT with a create block for an NPC.
func (gc *WorldSession) sendCreateObjectForNPC(npc *NPC) {
	pkt := wow.NewPacket(wow.ServerUpdateObject)

	_ = pkt.WriteUint32(1) // block count
	_ = pkt.WriteOne(0)    // has transport

	// Update type
	_ = pkt.WriteOne(wow.UpdateTypeCreateObject)

	// GUID
	_ = pkt.Write(npc.GetGUID())

	// Object type ID
	_ = pkt.WriteOne(int(wow.TypeIDUnit))

	// Update flags
	flags := uint8(wow.UpdateFlagLowGUID | wow.UpdateFlagHighGUID | wow.UpdateFlagLiving | wow.UpdateFlagHasPosition)
	_ = pkt.Write(flags)

	// Movement flags
	_ = pkt.Write(wow.MovementFlagNone)
	_ = pkt.WriteOne(0)                           // extra movement flags
	_ = pkt.Write(uint32(0))                      // time
	_ = pkt.Write(float32(npc.X))                 // X
	_ = pkt.Write(float32(npc.Y))                 // Y
	_ = pkt.Write(float32(npc.Z))                 // Z
	_ = pkt.Write(float32(npc.O))                 // O

	// Unit speeds
	_ = pkt.Write(float32(2.5))  // walk
	_ = pkt.Write(float32(7.0))  // run
	_ = pkt.Write(float32(4.5))  // run back
	_ = pkt.Write(float32(4.7))  // swim
	_ = pkt.Write(float32(2.5))  // swim back
	_ = pkt.Write(float32(7.0))  // flight
	_ = pkt.Write(float32(4.5))  // flight back
	_ = pkt.Write(float32(7.0))  // turn rate

	// Low GUID
	_ = pkt.WriteUint32(0x0B) // unk for units

	// High GUID
	_ = pkt.WriteUint32(0x00) // unk

	// Values update
	mask := npc.Object.BuildFullUpdateMask()
	blockCount := mask.GetUpdateBlockCount()

	// Write mask
	for i := uint32(0); i < blockCount; i++ {
		val := uint32(0)
		for b := uint32(0); b < 32; b++ {
			idx := i*32 + b
			if mask.GetBit(idx) {
				val |= 1 << b
			}
		}

		_ = pkt.Write(val)
	}

	// Write values
	for i := uint32(0); i < blockCount*32; i++ {
		if mask.GetBit(i) && int(i) < npc.Object.ValuesCount() {
			_ = pkt.Write(npc.Object.GetUInt32Value(object.UpdateField(i)))
		}
	}

	gc.socket.Send(pkt)
}

// sendInitialPacketsAfterAddToMap sends all required packets after adding player to map.
// This mirrors Player::SendInitialPacketsAfterAddToMap in AzerothCore.
func (gc *WorldSession) sendInitialPacketsAfterAddToMap(p *player.Player) {
	// Reset time sync
	gc.resetTimeSync()

	// Send initial world states
	gc.sendInitWorldStates(p)

	// Send player create update to self (SMSG_UPDATE_OBJECT with player values)
	gc.sendPlayerCreate(p)
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
	pkt := wow.NewPacket(wow.ServerInitialSpells)

	_ = pkt.WriteOne(0) // Spell book (0 = spellbook)

	// Number of spells
	_ = pkt.Write(uint32(len(p.KnownSpells)))

	// Spell list (each: spell ID + slot)
	for i, spellID := range p.KnownSpells {
		_ = pkt.Write(uint32(spellID))
		_ = pkt.Write(uint16(i)) // slot index
	}

	// Cooldown count (0 = no cooldowns)
	_ = pkt.Write(uint32(0))

	gc.socket.Send(pkt)
}

// sendActionButtons sends SMSG_ACTION_BUTTONS with the player's action bar.
func (gc *WorldSession) sendActionButtons(p *player.Player) {
	pkt := wow.NewPacket(wow.ServerActionButtons)

	// Send all 120 action buttons (12 buttons x 3 bars + 12 stance buttons)
	// Each button is 4 bytes: uint32 packed (action | type << 24)
	for i := 0; i < 120; i++ {
		var button uint32

		if i < len(p.Actions) {
			button = p.Actions[i]
		}

		_ = pkt.Write(button)
	}

	// Knowned (client sends this back)
	_ = pkt.WriteOne(0)

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

	// Build update flags
	flags := uint8(wow.UpdateFlagSelf | wow.UpdateFlagLowGUID | wow.UpdateFlagHighGUID | wow.UpdateFlagLiving | wow.UpdateFlagHasPosition)
	upd.updateFlags = flags

	// Build update object
	pkt := upd.BuildUpdateObject(p)

	gc.socket.Send(pkt)
}

// sendCreateObjectForGameObject sends SMSG_UPDATE_OBJECT with a create block for a game object.
func (gc *WorldSession) sendCreateObjectForGameObject(gobj *GameObject) {
	pkt := wow.NewPacket(wow.ServerUpdateObject)

	_ = pkt.WriteUint32(1) // block count
	_ = pkt.WriteOne(0)    // has transport

	// Update type
	_ = pkt.WriteOne(wow.UpdateTypeCreateObject)

	// GUID
	_ = pkt.Write(gobj.GetGUID())

	// Object type ID
	_ = pkt.WriteOne(int(wow.TypeIDGameObject))

	// Update flags for game object
	flags := uint8(wow.UpdateFlagLowGUID | wow.UpdateFlagHighGUID | wow.UpdateFlagHasPosition)
	_ = pkt.Write(flags)

	// Stationary position (game objects don't move)
	_ = pkt.Write(float32(gobj.X))
	_ = pkt.Write(float32(gobj.Y))
	_ = pkt.Write(float32(gobj.Z))
	_ = pkt.Write(float32(gobj.O))

	// Rotation quaternion (0, 0, 0, 1 = no rotation)
	_ = pkt.Write(float32(0))
	_ = pkt.Write(float32(0))
	_ = pkt.Write(float32(0))
	_ = pkt.Write(float32(1))

	// Low GUID
	_ = pkt.WriteUint32(0x0B) // unk for game objects

	// High GUID
	_ = pkt.WriteUint32(0x00) // unk

	// Values update
	mask := gobj.Object.BuildFullUpdateMask()
	blockCount := mask.GetUpdateBlockCount()

	// Write mask
	for i := uint32(0); i < blockCount; i++ {
		val := uint32(0)
		for b := uint32(0); b < 32; b++ {
			idx := i*32 + b
			if mask.GetBit(idx) {
				val |= 1 << b
			}
		}

		_ = pkt.Write(val)
	}

	// Write values
	for i := uint32(0); i < blockCount*32; i++ {
		if mask.GetBit(i) && int(i) < gobj.Object.ValuesCount() {
			_ = pkt.Write(gobj.Object.GetUInt32Value(object.UpdateField(i)))
		}
	}

	gc.socket.Send(pkt)
}
