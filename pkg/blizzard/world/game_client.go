package world

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	. "github.com/paalgyula/summit/pkg/blizzard/world/packets"
	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type SessionManager interface {
	AddClient(*GameClient)
	Disconnected(string)
}

type GameClient struct {
	ID  string
	n   net.Conn
	log zerolog.Logger

	seed uint32

	input *wow.Reader

	readLock  sync.Mutex
	writeLock sync.Mutex

	crypt *crypt.WowCrypt
	acc   *db.Account

	ws SessionManager
}

func NewGameClient(n net.Conn, ws SessionManager, handlers ...PacketHandler) *GameClient {
	gc := &GameClient{
		ID: xid.New().String(),
		n:  n,
		log: log.With().
			Caller().
			Str("server", "world").
			Str("addr", n.RemoteAddr().String()).
			Logger(),

		readLock: sync.Mutex{},
		input:    wow.NewConnectionReader(n),

		writeLock: sync.Mutex{},
		ws:        ws,
	}

	// Register opcode handlers from handlers.go
	gc.RegisterHandlers(handlers...)

	go gc.handleConnection()
	ws.AddClient(gc)

	return gc
}

func (gc *GameClient) handleConnection() {
	defer gc.ws.Disconnected(gc.ID)

	time.Sleep(time.Millisecond * 500)
	gc.log.Error().Msg("sending auth challenge")
	gc.sendAuthChallenge()

	for {
		err := gc.handlePacket()
		if err != nil {
			gc.log.Error().Err(err).Msg("cannot handle packet(s)")
			return
		}
	}
}

func (gc *GameClient) SendPacket(opcode OpCode, payload []byte) {
	size := len(payload)
	header, err := gc.makeHeader(size, opcode)
	if err != nil {
		gc.log.Error().Err(err).Msg("cannot make packet header, dropping client")
		gc.Close()
	}

	gc.writeLock.Lock()
	defer gc.writeLock.Unlock()

	packet := append(header, payload...)
	_, err = gc.n.Write(packet)

	gc.log.Trace().Err(err).
		Msgf(">> sending packet 0x%04x (%s), payload size: %d packet size: %d", opcode.Int(), opcode.String(), size, len(packet))
}

func (gc *GameClient) makeHeader(packetLen int, opCode OpCode) ([]byte, error) {
	w := wow.NewPacketWriter()
	w.WriteB(uint16(packetLen + 2))
	w.WriteL(uint16(opCode))
	header := w.Bytes()

	if gc.crypt != nil {
		header = gc.crypt.Encrypt(w.Bytes())
	}

	return header, nil
}

func (gc *GameClient) handlePacket() error {
	gc.readLock.Lock()
	defer gc.readLock.Unlock()

	opCode, length, err := gc.readHeader()
	if err != nil && length < 0 {
		return err
	}

	data, err := gc.input.ReadBytes(length)
	if err != nil {
		return fmt.Errorf("with opcode: %0X, %w", opCode, err)
	}

	gc.log.Trace().Msgf("packet received 0x%04x (%s) size: %d", opCode.Int(), opCode.String(), len(data))

	return gc.Handle(opCode, data)
}

func (gc *GameClient) readHeader() (OpCode, int, error) {
	header, err := gc.input.ReadBytes(6)
	if err != nil {
		return 0, -1, errors.New("cannot read opcode")
	}

	if gc.crypt != nil {
		header = gc.crypt.Decrypt(header)
	}

	r := wow.NewPacketReader(header)

	var length uint16
	var opcode uint32

	r.ReadB(&length)
	r.ReadL(&opcode)

	opCode := OpCode(opcode)

	// fmt.Printf("world: decoded opcode: %02x, %s len: %d encrypted: %t\n",
	// 	opCode.Int(), opCode.String(), length, gc.crypt != nil)

	return opCode, int(length) - 4, nil
}

func (gc *GameClient) Close() error {
	return gc.n.Close()
}
