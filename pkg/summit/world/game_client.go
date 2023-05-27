package world

import (
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	. "github.com/paalgyula/summit/pkg/summit/world/packets"
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

	// External packet handler connection
	bs *babysocket.Server
}

func NewGameClient(n net.Conn, ws SessionManager, bs *babysocket.Server, handlers ...PacketHandler) *GameClient {
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
		bs:        bs,
	}

	// Register opcode handlers from handlers.go
	gc.RegisterHandlers(handlers...)

	go gc.handleConnection()
	ws.AddClient(gc)

	return gc
}

func (gc *GameClient) recover() {
	a := recover()

	gc.log.Error().Msgf("panic occured, dropping client")
	fmt.Printf("Unhandled Error: %s\n%s",
		a,
		string(debug.Stack()),
	)

	// Close connection
	gc.n.Close()
}

func (gc *GameClient) handleConnection() {
	defer gc.recover() // Panic handler
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

func (gc *GameClient) SendPayload(opcode int, payload []byte) {
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

	oc := OpCode(opcode)
	gc.log.Trace().Err(err).
		Msgf(">> sending packet 0x%04x (%s), payload size: %d packet size: %d", oc.Int(), oc.String(), size, len(packet))
}

func (gc *GameClient) Send(packet wow.Packet) {
	size := packet.Len()
	header, err := gc.makeHeader(size, packet.OpCode())
	if err != nil {
		gc.log.Error().Err(err).Msg("cannot make packet header, dropping client")
		gc.Close()
	}

	gc.writeLock.Lock()
	defer gc.writeLock.Unlock()

	payload := append(header, packet.Bytes()...)
	_, err = gc.n.Write(payload)

	oc := OpCode(packet.OpCode())
	gc.log.Trace().Err(err).
		Msgf(">> sending packet 0x%04x (%s), payload size: %d packet size: %d", oc.Int(), oc.String(), size, packet.Len())
}

func (gc *GameClient) makeHeader(packetLen int, opCode int) ([]byte, error) {
	w := wow.NewPacket(0)
	w.WriteB(uint16(packetLen + 2))
	w.Write(uint16(opCode))
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

	data, err := gc.input.ReadNBytes(length)
	if err != nil {
		return fmt.Errorf("with opcode: %0X, %w", opCode, err)
	}

	gc.log.Trace().Msgf("packet received 0x%04x (%s) size: %d", opCode.Int(), opCode.String(), len(data))

	if gc.bs != nil {
		gc.bs.SendPacket(gc.ID, opCode.Int(), data)
	}

	return gc.Handle(opCode, data)
}

func (gc *GameClient) readHeader() (OpCode, int, error) {
	header, err := gc.input.ReadNBytes(6)
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
