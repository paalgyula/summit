package world

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ErrCannotReadHeader = errors.New("cannot read opcode")

type GameClient struct {
	ID  string
	n   net.Conn
	log zerolog.Logger

	// Server side generated seed for authentication proofing
	serverSeed []byte

	input *wow.PacketReader

	readLock  sync.Mutex
	writeLock sync.Mutex

	crypt *crypt.WowCrypt

	// These is comes from the login (auth) server
	AccountName string
	sessionKey  *big.Int

	ws SessionManager

	// External packet handler connection
	bs *babysocket.Server
}

func NewGameClient(n net.Conn, ws SessionManager, bs *babysocket.Server, handlers ...PacketHandler) *GameClient {
	//nolint:exhaustruct
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

	// New server seed on connection
	gc.serverSeed = make([]byte, 4)
	_, _ = rand.Read(gc.serverSeed)

	// Register opcode handlers from handlers.go
	gc.RegisterHandlers(handlers...)

	go gc.handleConnection()
	ws.AddClient(gc)

	return gc
}

func (gc *GameClient) recover() {
	a := recover()

	gc.log.Error().Msgf("panic occurred, dropping client")
	log.Printf("Unhandled Error: %s\n%s",
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
	gc.log.Trace().Msg("sending auth challenge")
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

	_, err = gc.n.Write(append(header, payload...))

	oc := wow.OpCode(opcode)
	gc.log.Trace().Err(err).
		Str("packet", oc.String()).
		Str("opcode", fmt.Sprintf("0x%04x", oc)).
		Int("payloadSize", size).
		Msgf(">> sending packet 0x%04x (%v), packet size: %d",
			int(oc),
			oc.String(),
			len(header)+len(payload))
}

func (gc *GameClient) Send(packet *wow.Packet) {
	size := packet.Len()

	payload, err := gc.makeHeader(size, packet.OpCode())
	if err != nil {
		gc.log.Error().Err(err).Msg("cannot make packet header, dropping client")
		gc.Close()
	}

	gc.writeLock.Lock()
	defer gc.writeLock.Unlock()

	payload = append(payload, packet.Bytes()...)
	_, err = gc.n.Write(payload)

	oc := wow.OpCode(packet.OpCode())

	gc.log.Trace().Err(err).
		Str("packet", oc.String()).
		Int("size", packet.Len()).
		Msgf(">> sending packet 0x%04x", int(oc))
}

func (gc *GameClient) makeHeader(packetLen int, opCode int) ([]byte, error) {
	w := wow.NewPacket(0)
	if err := w.WriteB(uint16(packetLen + 2)); err != nil {
		return nil, fmt.Errorf("error while writing packet length: %w", err)
	}

	if err := w.Write(uint16(opCode)); err != nil {
		return nil, fmt.Errorf("error while writing opcode: %w", err)
	}

	header := w.Bytes()

	if gc.crypt != nil {
		header = gc.crypt.Encrypt(w.Bytes())
	}

	return header, nil
}

func (gc *GameClient) handlePacket() error {
	gc.readLock.Lock()
	defer gc.readLock.Unlock()

	// Read and decode the header when crypt is not nil
	opCode, length, err := gc.readHeader()
	if err != nil && length < 0 {
		return err
	}

	// Read the rest of the packet
	data, err := gc.input.ReadNBytes(length)
	if err != nil {
		return fmt.Errorf("with opcode: %0X, %w", opCode, err)
	}

	gc.log.Trace().
		Str("opcode", fmt.Sprintf("0x%04x", int(opCode))).
		Str("packet", opCode.String()).
		Int("pktSize", len(data)).
		Msgf("<< packet received 0x%04x", int(opCode))

	if gc.bs != nil {
		gc.bs.SendPacketToBabies(gc.ID, int(opCode), data)
	}

	return gc.Handle(opCode, data)
}

func (gc *GameClient) readHeader() (wow.OpCode, int, error) {
	header, err := gc.input.ReadNBytes(6)
	if err != nil {
		return 0, -1, ErrCannotReadHeader
	}

	if gc.crypt != nil {
		header = gc.crypt.Decrypt(header)
	}

	r := wow.NewPacketReader(header)

	var length uint16
	// Get the length first
	if err := r.ReadB(&length); err != nil {
		return 0, -1, fmt.Errorf("error while reading packet length: %w", err)
	}

	var opcode uint32
	// Then read the opcode
	if err := r.ReadL(&opcode); err != nil {
		return 0, -1, fmt.Errorf("error while reading opcode: %w", err)
	}

	// gc.log.Trace().Msgf("world: decoded opcode: %02x, %v len: %d encrypted: %t",
	// 	opcode, wow.OpCode(opcode), length, gc.crypt != nil)

	return wow.OpCode(opcode), int(length) - 4, nil
}

func (gc *GameClient) SessionKey() []byte {
	return gc.sessionKey.Bytes()
}

func (gc *GameClient) Close() error {
	return gc.n.Close() //nolint:wrapcheck
}
