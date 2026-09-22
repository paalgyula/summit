package world

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/paalgyula/summit/pkg/summit/world/packets"
	"io"
	"math/big"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/protocol"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ServerPacketHeaderSize size of the server's packet header in bytes.
// 2 bytes of length in big endian and 4bytes of opcode in little endian
// byte order.
const ServerPacketHeaderSize = 6

var ErrCannotReadHeader = errors.New("cannot read opcode")

type WorldSession struct {
	ID  string
	n   net.Conn
	log zerolog.Logger

	closeOnce sync.Once

	// opcodes maps client opcodes to this session's handlers. Handlers are
	// closures over the session, so the table cannot be shared between sessions.
	opcodes packets.Opcodes

	// Server side generated seed for authentication proofing
	serverSeed []byte

	// crypt *crypt.WowCrypt

	// These is comes from the login (auth) server
	AccountName string
	SessionKey  *big.Int

	ws SessionManager

	// External packet handler connection
	// bs *babysocket.Server

	socket *protocol.WoWSocket

	// Player currently logged in through this session
	player *player.Player

	// questLog mirrors the client quest log slots (PLAYER_QUEST_LOG_n_1);
	// 0 means the slot is free.
	questLog [maxQuestLogSlots]uint32

	// Time sync counter
	timeSyncCounter uint32
}

func NewWorldSession(n net.Conn, ws SessionManager, handlers ...PacketHandler) *WorldSession {
	wowsocket := protocol.NewWoWSocket(n)

	//nolint:exhaustruct
	gc := &WorldSession{
		ID: xid.New().String(),
		n:  n,
		log: log.With().
			Caller().
			Str("server", "world").
			Str("addr", n.RemoteAddr().String()).
			Logger(),
		socket: wowsocket,
		ws:     ws,
	}

	// New server seed on connection
	gc.serverSeed = make([]byte, 4)
	_, _ = rand.Read(gc.serverSeed)

	// Register opcode handlers from handlers.go
	gc.opcodes = packets.NewOpcodeTable()
	gc.RegisterHandlers(handlers...)

	go gc.handleConnection()
	ws.AddClient(gc)

	return gc
}

// LoginCharacter loads and initializes the player character into the world
// sending all initial world packets directly over the active session.
func (gc *WorldSession) LoginCharacter(p *player.Player) {
	gc.player = p
	p.Sender = gc
	p.Init()

	// Sync XP bar and talent points derived from current level.
	p.Object.SetUInt32Value(object.PlayerXp, p.XP)
	p.Object.SetUInt32Value(object.PlayerNextLevelXp, NextLevelXP(p.Level))

	// Talent points: 1 per level starting at level 10 (up to 71 at level 80).
	if p.Level >= MinTalentLevel {
		talentPoints := uint32(p.Level) - uint32(MinTalentLevel) + 1
		// Cap at 71 (levels 10–80)
		if talentPoints > 71 {
			talentPoints = 71
		}
		p.Object.SetUInt32Value(object.PlayerCharacterPoints1, talentPoints)
	}

	gc.sendLoginVerifyWorld(p)
	gc.sendFeatureSystemStatus()
	gc.sendMOTD()
	gc.sendLearnedDanceMoves()
	gc.sendInitialPacketsBeforeAddToMap(p)
	gc.addPlayerToMap(p)
	gc.sendInitialPacketsAfterAddToMap(p)
	gc.setPlayerOnline(p)
	gc.sendFriendStatus(p, FriendStatusOnline)

	if p.GroupID > 0 {
		gc.sendGroupUpdate(p)
	}

	gc.log.Info().Str("name", p.Name).Msg("player logged into world via websocket")
}

func (gc *WorldSession) recover() {
	a := recover()
	if a == nil { // No recover needed
		return
	}

	gc.log.Error().Interface("reason", a).Msgf("panic occurred, dropping client")

	r := bufio.NewReader(bytes.NewBuffer(debug.Stack()))
	for i := 0; i < 5; i++ {
		_, _, _ = r.ReadLine()
	}

	stack, _ := io.ReadAll(r)

	fmt.Fprintf(os.Stderr,
		"unhandled client error: \n%s",
		string(stack),
	)

	// Close connection
	gc.Close()
}

func (gc *WorldSession) handleConnection() {
	defer gc.recover() // Panic handler

	time.Sleep(time.Millisecond * 500)
	gc.log.Trace().Msg("sending auth challenge")
	gc.sendAuthChallenge()

	// Handle packets from the channel; the socket closes it when the
	// connection is gone, which must take the player out of the world too.
	for pkt := range gc.socket.Packets() {
		gc.Handle(pkt)
	}

	gc.close("connection closed")
}

// Close drops the client: saves and removes the player, then closes the socket.
func (gc *WorldSession) Close() error {
	gc.close("closing GameClient")

	return nil
}

// close runs the disconnect sequence once, whichever path triggers it first
// (logout, read error, panic, server-side kick).
func (gc *WorldSession) close(reason string) {
	gc.closeOnce.Do(func() {
		gc.ws.Disconnected(gc, reason)
		_ = gc.socket.Close()
		_ = gc.n.Close()
	})
}

// HandleLogoutRequest handles CMSG_LOGOUT_REQUEST: saves the character and
// sends the logout response. The client will disconnect after receiving the
// response.
func (gc *WorldSession) HandleLogoutRequest(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	// Save the character before logout
	if err := gc.ws.(*Server).charStore.UpdateCharacter(gc.player); err != nil {
		gc.log.Error().Err(err).Str("name", gc.player.Name).
			Msg("failed to save character on logout")
	} else {
		gc.log.Info().Str("name", gc.player.Name).Msg("character saved on logout")
	}

	// Send SMSG_LOGOUT_RESPONSE (uint32: 0 = OK, uint8: 0 = not instant)
	pkt := wow.NewPacket(wow.ServerLogoutResponse)
	_ = pkt.Write(uint32(0)) // LOGOUT_RESPONSE_OK
	_ = pkt.WriteOne(0)      // not instant logout
	gc.socket.Send(pkt)

	// Send SMSG_LOGOUT_COMPLETE after a short delay to let the client process
	go func() {
		time.Sleep(100 * time.Millisecond)
		pkt := wow.NewPacket(wow.ServerLogoutComplete)
		gc.socket.Send(pkt)
		gc.Close()
	}()
}

// Send sends a packet to the game client.
func (gc *WorldSession) Send(pkt *wow.Packet) {
	gc.socket.Send(pkt)
}

// SendPayload sends a packet with the given opcode and payload to the game client.
// This satisfies the wow.PayloadSender interface for babysocket integration.
func (gc *WorldSession) SendPayload(opcode int, payload []byte) {
	pkt := wow.NewPacketWithData(wow.OpCode(opcode), payload)
	gc.socket.SendPayload(pkt)
}

// sendDestroyObject sends SMSG_DESTROY_OBJECT to remove an object from the client.
func (gc *WorldSession) sendDestroyObject(guid wow.GUID) {
	pkt := wow.NewPacket(wow.ServerDestroyObject)
	_ = pkt.Write(guid)
	_ = pkt.WriteOne(0) // not despawn animation
	gc.socket.Send(pkt)
}

// sendDestroyObjectWithDeath sends SMSG_DESTROY_OBJECT with the death flag,
// triggering the death animation on the client.
func (gc *WorldSession) sendDestroyObjectWithDeath(target *player.Player) {
	pkt := BuildDestroyObject(target, true)
	gc.socket.Send(pkt)
}

// Regen interval in milliseconds.
const RegenIntervalMS = 2000

// processRegen handles health and mana regeneration.
func (gc *WorldSession) processRegen(now time.Time) {
	if gc.player == nil {
		return
	}

	if now.UnixMilli() < gc.player.NextRegenTime {
		return
	}

	regened := false

	// Health regen (1% of max health per tick, simplified)
	if gc.player.Health < gc.player.MaxHealth {
		healthGain := gc.player.MaxHealth / 100
		if healthGain < 1 {
			healthGain = 1
		}

		newHealth := gc.player.Health + healthGain
		if newHealth > gc.player.MaxHealth {
			newHealth = gc.player.MaxHealth
		}

		gc.player.SetHealth(newHealth)
		regened = true
	}

	powerType := gc.player.GetPrimaryPowerType()

	switch powerType {
	case wow.PowerTypeMana:
		// Mana regen: 1% of max mana per tick (out of combat)
		// In WoW, mana regen is based on Spirit and only works when not casting
		if gc.player.GetPower(powerType) < gc.player.GetMaxPower(powerType) {
			manaGain := gc.player.GetMaxPower(powerType) / 100
			if manaGain < 1 {
				manaGain = 1
			}

			newMana := gc.player.GetPower(powerType) + manaGain
			if newMana > gc.player.GetMaxPower(powerType) {
				newMana = gc.player.GetMaxPower(powerType)
			}

			gc.player.SetPower(powerType, newMana)
			regened = true
		}

	case wow.PowerTypeEnergy:
		// Energy regen: 10 energy per second (5 per 2s tick)
		// In WoW, energy regens at 10/sec for rogues/hunters
		if gc.player.GetPower(powerType) < gc.player.GetMaxPower(powerType) {
			energyGain := uint32(5) // 5 per 2s tick = 10/sec
			newEnergy := gc.player.GetPower(powerType) + energyGain
			if newEnergy > gc.player.GetMaxPower(powerType) {
				newEnergy = gc.player.GetMaxPower(powerType)
			}

			gc.player.SetPower(powerType, newEnergy)
			regened = true
		}

	case wow.PowerTypeFocus:
		// Focus regen: 10 focus per second (5 per 2s tick)
		// In WoW, focus regens at 10/sec for hunters
		if gc.player.GetPower(powerType) < gc.player.GetMaxPower(powerType) {
			focusGain := uint32(5) // 5 per 2s tick = 10/sec
			newFocus := gc.player.GetPower(powerType) + focusGain
			if newFocus > gc.player.GetMaxPower(powerType) {
				newFocus = gc.player.GetMaxPower(powerType)
			}

			gc.player.SetPower(powerType, newFocus)
			regened = true
		}

	case wow.PowerTypeRage:
		// Rage does not regen passively
		// Rage is gained from:
		// - Taking damage (rageGain = damage / 60, roughly)
		// - Dealing damage (rageGain = 5 per melee hit, more for crits)
		// - Abilities like Bloodrage
		// We handle rage gain in combat.go when damage is dealt/taken
	}

	if regened {
		gc.broadcastPlayerStats()
	}

	// Set next regen time
	gc.player.NextRegenTime = now.UnixMilli() + RegenIntervalMS
}

// broadcastPlayerStats sends health/mana updates to all nearby players.
func (gc *WorldSession) broadcastPlayerStats() {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Build update mask with health and power
	mask := &object.UpdateMask{}
	mask.SetCount(uint32(gc.player.Object.ValuesCount()))

	powerType := gc.player.GetPrimaryPowerType()

	// Always update health
	mask.SetBit(uint32(object.UnitFieldHealth))

	// Update primary power
	if powerType >= 0 && int(powerType) < wow.MaxPowerTypes {
		mask.SetBit(uint32(object.UpdateField(int(object.UnitFieldPower1) + int(powerType))))
	}

	// Build values block
	blockCount := mask.GetUpdateBlockCount()

	// Create values update packet
	pkt := wow.NewPacket(wow.ServerUpdateObject)

	_ = pkt.WriteUint32(1) // block count
	_ = pkt.WriteOne(0)    // has transport

	// Update type
	_ = pkt.WriteOne(wow.UpdateTypeValues)
	_ = pkt.Write(gc.player.GUID())

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
		if mask.GetBit(i) && int(i) < gc.player.Object.ValuesCount() {
			_ = pkt.Write(gc.player.Object.GetUInt32Value(object.UpdateField(i)))
		}
	}

	// Send to all
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
}

// sendLevelUpInfo sends SMSG_LEVELUP_INFO when the player gains a level.
//
// WotLK 3.3.5a layout:
//
//	uint32  newLevel
//	uint32  healthGain
//	uint32  manaGain     (power[0])
//	uint32  rageGain     (power[1]) — always 0
//	uint32  focusGain    (power[2]) — always 0
//	uint32  energyGain   (power[3]) — always 0
//	uint32  runeGain     (power[4]) — always 0
//	uint32  strGain
//	uint32  agiGain
//	uint32  staGain
//	uint32  intGain
//	uint32  spiGain
func (gc *WorldSession) sendLevelUpInfo(newLevel uint8, hp, mana, str, agi, sta, int_, spi uint32) {
	pkt := wow.NewPacket(wow.ServerLevelupInfo)

	_ = pkt.Write(uint32(newLevel))
	_ = pkt.Write(hp)
	_ = pkt.Write(mana)      // power 0
	_ = pkt.Write(uint32(0)) // power 1 (rage)
	_ = pkt.Write(uint32(0)) // power 2 (focus)
	_ = pkt.Write(uint32(0)) // power 3 (energy)
	_ = pkt.Write(uint32(0)) // power 4 (runes)
	_ = pkt.Write(str)
	_ = pkt.Write(agi)
	_ = pkt.Write(sta)
	_ = pkt.Write(int_)
	_ = pkt.Write(spi)

	gc.Send(pkt)
}

// sendDeath sends death notification to the client.
func (gc *WorldSession) sendDeath() {
	// Send release spirit dialog
	pkt := wow.NewPacket(wow.ServerDeathReleaseLoc)

	_ = pkt.Write(uint32(0))  // release location map
	_ = pkt.Write(float32(0)) // release location x
	_ = pkt.Write(float32(0)) // release location y
	_ = pkt.Write(float32(0)) // release location z

	gc.socket.Send(pkt)
}

// sendResurrectRequest sends SMSG_RESURRECT_REQUEST to the client.
func (gc *WorldSession) sendResurrectRequest() {
	pkt := wow.NewPacket(wow.ServerResurrectRequest)

	_ = pkt.Write(uint64(0)) // resurrecter GUID (spirit healer)
	pkt.WriteString("Spirit Healer")
	_ = pkt.Write(uint32(0)) // hierarchy

	gc.socket.Send(pkt)
}

// updatePeriodic runs periodic updates for the player (regen, saves, etc).
func (gc *WorldSession) updatePeriodic(now time.Time) {
	if gc.player == nil || !gc.player.IsInWorld {
		return
	}

	// Don't regen while dead/ghost
	if gc.player.IsGhost {
		return
	}

	// Process combat (auto-attack swings)
	gc.ProcessCombatTick(now)

	// Process spell casts
	gc.ProcessSpellTick(now)

	// Process health/mana regen
	gc.processRegen(now)
}
