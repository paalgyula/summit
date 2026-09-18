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

// updatePeriodic runs periodic updates for the player (regen, saves, etc).
func (gc *WorldSession) updatePeriodic(now time.Time) {
	if gc.player == nil || !gc.player.IsInWorld {
		return
	}

	// Process combat (auto-attack swings)
	gc.ProcessCombatTick(now)

	// TODO: Implement health/mana regeneration
	// TODO: Implement aura tick
	// TODO: Implement save timer
}
