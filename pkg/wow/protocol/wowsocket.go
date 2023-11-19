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
func NewWoWSocket(conn io.ReadWriteCloser) *WoWSocket {
	ws := &WoWSocket{
		crypt:        nil,
		cryptEnabled: false,
		log:          log.With().Logger(),

		sendChan:    make(chan *wow.Packet),
		receiveChan: make(chan *wow.Packet),

		input: wow.NewConnectionReader(conn),

		connection: conn,
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
func (ws *WoWSocket) SetCrypt(crypt *crypt.WowCrypt, enable bool) {
	ws.crypt = crypt
	ws.cryptEnabled = enable
}

// EnableCrypt enables the packet header crypt.
func (ws *WoWSocket) EnableCrypt() {
	ws.cryptEnabled = true
}

// Close closes the socket, and the send/receive channels.
func (ws *WoWSocket) Close() error {
	ws.connection.Close()

	close(ws.receiveChan)
	close(ws.sendChan)

	return nil
}

// Returns a receive channel of packet stream.
func (ws *WoWSocket) Packets() <-chan *wow.Packet {
	return ws.receiveChan
}

// Send sends out the packet.
func (ws *WoWSocket) Send(pkt *wow.Packet) {
	ws.sendChan <- pkt
}

func (ws *WoWSocket) SendPayload(pkt *wow.Packet) {
	header, err := ws.makeHeader(pkt.Opcode(), pkt.Len())
	if err != nil {
		ws.log.Error().Err(err).Msg("cannot make packet header, dropping client")
		ws.Close()
	}

	ws.connection.Write(header)
	ws.connection.Write(pkt.Bytes())

	ws.log.Trace().Err(err).
		Str("packet", pkt.Opcode().String()).
		Str("opcode", fmt.Sprintf("0x%04x", pkt.OpCode())).
		Int("size", pkt.Len()).
		Msgf(">>> %s", pkt.Opcode().String())
}

// func (ws *WoWSocket) Send(packet *wow.Packet) {
// 	size := packet.Len()

// 	payload, err := ws.makeHeader(size, packet.OpCode())
// 	if err != nil {
// 	 ws.log.Error().Err(err).Msg("cannot make packet header, dropping client")
// 	 ws.Close()
// 	}

//  ws.writeLock.Lock()
// 	defer ws.writeLock.Unlock()

// 	payload = append(payload, packet.Bytes()...)
// 	_, err = ws.n.Write(payload)

// 	oc := wow.OpCode(packet.OpCode())

//  ws.log.Trace().Err(err).
// 		Str("packet", oc.String()).
// 		Int("size", packet.Len()).
// 		Msgf(">> sending packet 0x%04x", int(oc))
// }

func (ws *WoWSocket) makeHeader(opcode wow.OpCode, dataSize int) ([]byte, error) {
	w := wow.NewPacket(opcode)

	_ = w.Write(uint16(dataSize+2), binary.BigEndian) // Packet length
	_ = w.Write(uint16(opcode), binary.LittleEndian)  // OpCode

	header := w.Bytes()

	if ws.crypt != nil && ws.cryptEnabled {
		header = ws.crypt.Encrypt(header)
	}

	return header, nil
}

// goroutine for sending out the packets from the receive channel.
func (ws *WoWSocket) connectionWriter() {
	for p := range ws.sendChan {
		ws.SendPayload(p)
	}
}

// goroutine for reading packets and send it to the
// receiveChan channel for processing.
func (ws *WoWSocket) connectionReader() {
	for {
		// Read the incoming packet data and
		// decode the header when crypt is enabled
		opCode, data, err := ws.readPacket()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				ws.log.Error().Msg("connection closed")

				_ = ws.Close()

				return
			}

			ws.log.Error().Err(err).Msg("packet read error occured")

			continue
		}

		ws.log.Trace().
			Str("opcode", fmt.Sprintf("0x%04x", int(opCode))).
			Str("packet", opCode.String()).
			Int("size", len(data)).
			Msgf("<<< %s", opCode.String())

		// TODO: re-implement babysockets
		// if ws.bs != nil {
		//  ws.bs.SendPacketToBabies(gc.ID, int(opCode), data)
		// }

		ws.receiveChan <- wow.NewPacketWithData(opCode, data)
	}
}

func (ws *WoWSocket) readPacket() (wow.OpCode, []byte, error) {
	header, err := ws.input.ReadNBytes(ServerPacketHeaderSize)
	if err != nil {
		return 0, nil, fmt.Errorf("wowsocket.readPacket: %w", err)
	}

	if ws.crypt != nil {
		header = ws.crypt.Decrypt(header)
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

	// ws.log.Trace().Msgf("world: decoded opcode: %02x, %v len: %d encrypted: %t",
	// 	opcode, wow.OpCode(opcode), length, ws.crypt != nil)

	dataSize := int(length) + 2 - ServerPacketHeaderSize
	content := make([]byte, dataSize)

	if err := ws.input.Read(content); err != nil {
		return 0, nil, fmt.Errorf(
			"readPacket: content is too small: %w", err)
	}

	return wow.OpCode(code), content, nil
}
