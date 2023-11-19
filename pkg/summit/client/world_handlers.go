package client

import (
	"bytes"
	"encoding/binary"

	"github.com/paalgyula/summit/pkg/wow"
)

// ! Packet handler definitions
func (wc *WorldClient) handleMessage(msg *ServerMessage) {
	switch msg.Opcode {
	case wow.ServerAuthChallenge:
		wc.handleAuthChallenge(msg)
	case wow.ServerAuthResponse:
		wc.handleAuthResponse(msg)
	case wow.ServerCharEnum:
		wc.handleCharEnum(msg)
	default:
		wc.log.Warn().
			Str("packet", msg.Opcode.String()).
			Int("size", len(msg.Data)).
			Msgf("unhandled packet: %s", msg.Opcode.String())
	}
}

func (wc *WorldClient) makeHeader(opcode wow.OpCode, dataSize int) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, uint16(dataSize+4)) // +4 is the header length
	_ = binary.Write(buf, binary.LittleEndian, uint32(opcode))

	header := buf.Bytes()

	if wc.cryptEnable {
		header = wc.crypt.Encrypt(header)
	}

	if len(header) != 6 {
		wc.log.Fatal().Msgf("header must be 6 bytes long, got: %d", len(header))
	}

	return header
}

// Send data to the client
func (wc *WorldClient) Send(pkt *wow.Packet) {
	wc.clientMessages <- pkt
}
