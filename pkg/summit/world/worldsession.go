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
