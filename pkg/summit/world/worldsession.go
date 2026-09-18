package world

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"runtime/debug"
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
	gc.RegisterHandlers(handlers...)

	go gc.handleConnection()
	ws.AddClient(gc)

	return gc
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

	// Handle packets from the channel.
	for pkt := range gc.socket.Packets() {
		gc.Handle(pkt)
	}
}

func (gc *WorldSession) Close() error {
	gc.ws.Disconnected(gc, "closing GameClient")

	return gc.n.Close() //nolint:wrapcheck
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

	// Don't regen while in combat (TODO: track combat state properly)
	// For now, always regen

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

	// Mana regen (1% of max mana per tick for mana users)
	powerType := gc.player.GetPrimaryPowerType()
	if powerType == wow.PowerTypeMana && gc.player.GetPower(powerType) < gc.player.GetMaxPower(powerType) {
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

	// Energy regen (1 energy per tick for rogues/feral druids)
	if powerType == wow.PowerTypeEnergy && gc.player.GetPower(powerType) < gc.player.GetMaxPower(powerType) {
		newEnergy := gc.player.GetPower(powerType) + 1
		if newEnergy > gc.player.GetMaxPower(powerType) {
			newEnergy = gc.player.GetMaxPower(powerType)
		}

		gc.player.SetPower(powerType, newEnergy)
		regened = true
	}

	// Rage does not regen out of combat (handled separately)

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
		mask.SetBit(uint32(object.UpdateField(int(object.UnitFieldPower1)+int(powerType))))
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
func (gc *WorldSession) sendLevelUpInfo(levelsGained uint32) {
	pkt := wow.NewPacket(wow.ServerLevelupInfo)

	_ = pkt.Write(uint32(levelsGained)) // levels gained
	_ = pkt.Write(uint32(gc.player.Level))
	_ = pkt.Write(uint32(0)) // bonus health
	_ = pkt.Write(uint32(0)) // bonus mana
	_ = pkt.Write(uint32(0)) // bonus talent points

	// Stat gains (str, agi, sta, int, spi)
	_ = pkt.Write(uint32(0)) // str
	_ = pkt.Write(uint32(0)) // agi
	_ = pkt.Write(uint32(0)) // sta
	_ = pkt.Write(uint32(0)) // int
	_ = pkt.Write(uint32(0)) // spi

	gc.socket.Send(pkt)
}

// sendDeath sends death notification to the client.
func (gc *WorldSession) sendDeath() {
	// Send release spirit dialog
	pkt := wow.NewPacket(wow.ServerDeathReleaseLoc)

	_ = pkt.Write(uint32(0)) // release location map
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

	// Process health/mana regen
	gc.processRegen(now)
}
