package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const ServerPacketHeaderSize = 6

// NewWoWSocket initializes a new socket reader with an easy-to-use
// channel based packet handler.
func NewWoWSocket(connection io.ReadWriteCloser) *WoWSocket {
	ws := &WoWSocket{
		crypt:        nil,
		cryptEnabled: false,
		log:          log.With().Logger(),

		sendChan:    make(chan *wow.Packet),
		receiveChan: make(chan *wow.Packet),

		input: wow.NewConnectionReader(connection),

		connection: connection,
	}

	// Start send/receive
	go ws.connectionReader()
	go ws.connectionWriter()

	return ws
}

type WoWSocket struct {
	crypt        *crypt.WowCrypt
	cryptEnabled bool

	sendChan    chan *wow.Packet
	receiveChan chan *wow.Packet

	input      *wow.PacketReader
	connection io.ReadWriteCloser

	log zerolog.Logger
}

// SetCrypt sets the WoWCrypt instance, and it can be turned on at the same time.
func (gc *WoWSocket) SetCrypt(crypt *crypt.WowCrypt, enable bool) {
	gc.crypt = crypt
	gc.cryptEnabled = enable
}

// EnableCrypt enables the packet header crypt.
func (gc *WoWSocket) EnableCrypt() {
	gc.cryptEnabled = true
}

// Close closes the socket, and the send/receive channels.
func (gc *WoWSocket) Close() error {
	gc.connection.Close()

	close(gc.receiveChan)
	close(gc.sendChan)

	return nil
}

// Returns a receive channel of packet stream.
func (gc *WoWSocket) Packets() <-chan *wow.Packet {
	return gc.receiveChan
}

// Send sends out the packet.
func (gc *WoWSocket) Send(pkt *wow.Packet) {
	gc.sendChan <- pkt
}

func (gc *WoWSocket) SendPayload(pkt *wow.Packet) {
	header, err := gc.makeHeader(pkt.Opcode(), pkt.Len())
	if err != nil {
		gc.log.Error().Err(err).Msg("cannot make packet header, dropping client")
		gc.Close()
	}

	gc.connection.Write(header)
	gc.connection.Write(pkt.Bytes())

	gc.log.Trace().Err(err).
		Str("packet", pkt.Opcode().String()).
		Str("opcode", fmt.Sprintf("0x%04x", pkt.OpCode())).
		Int("size", pkt.Len()).
		Msgf(">>> %s", pkt.Opcode().String())
}

// func (gc *WoWSocket) Send(packet *wow.Packet) {
// 	size := packet.Len()

// 	payload, err := gc.makeHeader(size, packet.OpCode())
// 	if err != nil {
// 		gc.log.Error().Err(err).Msg("cannot make packet header, dropping client")
// 		gc.Close()
// 	}

// 	gc.writeLock.Lock()
// 	defer gc.writeLock.Unlock()

// 	payload = append(payload, packet.Bytes()...)
// 	_, err = gc.n.Write(payload)

// 	oc := wow.OpCode(packet.OpCode())

// 	gc.log.Trace().Err(err).
// 		Str("packet", oc.String()).
// 		Int("size", packet.Len()).
// 		Msgf(">> sending packet 0x%04x", int(oc))
// }

func (gc *WoWSocket) makeHeader(opcode wow.OpCode, dataSize int) ([]byte, error) {
	w := wow.NewPacket(opcode)

	_ = w.Write(uint16(dataSize+2), binary.BigEndian) // Packet length
	_ = w.Write(uint16(opcode), binary.LittleEndian)  // OpCode

	header := w.Bytes()

	if gc.crypt != nil && gc.cryptEnabled {
		header = gc.crypt.Encrypt(header)
	}

	return header, nil
}

// goroutine for sending out the packets from the receive channel.
func (gc *WoWSocket) connectionWriter() {
	for p := range gc.sendChan {
		gc.SendPayload(p)
	}
}

// goroutine for reading packets and send it to the
// receiveChan channel for processing.
func (gc *WoWSocket) connectionReader() {
	for {
		// Read the incoming packet data and
		// decode the header when crypt is enabled
		opCode, data, err := gc.readPacket()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				log.Error().Msg("connection closed")

				_ = gc.Close()

				return
			}

			log.Error().Err(err).Msg("packet read error occured")

			continue
		}

		gc.log.Trace().
			Str("opcode", fmt.Sprintf("0x%04x", int(opCode))).
			Str("packet", opCode.String()).
			Int("size", len(data)).
			Msgf("<<< %s", opCode.String())

		// TODO: re-implement babysockets
		// if gc.bs != nil {
		// 	gc.bs.SendPacketToBabies(gc.ID, int(opCode), data)
		// }

		gc.receiveChan <- wow.NewPacketWithData(opCode, data)
	}
}

func (gc *WoWSocket) readPacket() (wow.OpCode, []byte, error) {
	header, err := gc.input.ReadNBytes(ServerPacketHeaderSize)
	if err != nil {
		return 0, nil, fmt.Errorf("wowsocket.readPacket: %w", err)
	}

	if gc.crypt != nil {
		header = gc.crypt.Decrypt(header)
	}

	r := wow.NewPacketReader(header)

	var length uint16
	// Get the length first
	if err := r.ReadB(&length); err != nil {
		return 0, nil, fmt.Errorf("packet length: %w", err)
	}

	var code uint32
	// Then read the opcode
	if err := r.ReadL(&code); err != nil {
		return 0, nil, fmt.Errorf("reading opcode: %w", err)
	}

	// gc.log.Trace().Msgf("world: decoded opcode: %02x, %v len: %d encrypted: %t",
	// 	opcode, wow.OpCode(opcode), length, gc.crypt != nil)

	dataSize := int(length) + 2 - ServerPacketHeaderSize
	content := make([]byte, dataSize)

	if err := gc.input.Read(content); err != nil {
		return 0, nil, fmt.Errorf(
			"readPacket: content is too small: %w", err)
	}

	return wow.OpCode(code), content, nil
}
