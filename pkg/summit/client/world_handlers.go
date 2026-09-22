package client

import (
	"bytes"
	"encoding/binary"

	"github.com/paalgyula/summit/pkg/wow"
)

// ! Packet handler definitions
func (wc *WorldClient) handleMessage(msg *ServerMessage) {
	// If forward handler is set, forward all packets instead of handling locally
	if wc.forwardHandler != nil {
		wc.forwardHandler(msg.Opcode, msg.Data)

		return
	}

	switch msg.Opcode {
	case wow.ServerAuthChallenge:
		wc.handleAuthChallenge(msg)
	case wow.ServerAuthResponse:
		wc.handleAuthResponse(msg)
	case wow.ServerCharEnum:
		wc.handleCharEnum(msg)
	case wow.ServerLoginVerifyWorld:
		wc.handleLoginVerifyWorld(msg)
	case wow.ServerCharacterLoginFailed:
		wc.handleCharacterLoginFailed(msg)
	case wow.ServerInitialSpells:
		wc.handleInitialSpells(msg)
	case wow.ServerActionButtons:
		wc.handleActionButtons(msg)
	case wow.ServerBindpointupdate:
		wc.handleBindPointUpdate(msg)
	case wow.ServerTimeSyncReq:
		wc.handleTimeSyncReq(msg)
	case wow.ServerUpdateObject:
		wc.handleUpdateObject(msg)
	case wow.ServerDestroyObject:
		wc.handleDestroyObject(msg)
	case wow.ServerNameQueryResponse:
		wc.handleNameQueryResponse(msg)
	case wow.ServerCreatureQueryResponse:
		wc.handleCreatureQueryResponse(msg)
	case wow.ServerMonsterMove:
		wc.handleMonsterMove(msg)
	case wow.ServerAttackstart:
		wc.handleAttackStart(msg)
	case wow.ServerAttackstop:
		wc.handleAttackStop(msg)
	case wow.ServerAttackerstateupdate:
		wc.handleAttackerStateUpdate(msg)
	case wow.ServerLogXpgain:
		wc.handleXpGain(msg)
	case wow.ServerLevelupInfo:
		wc.handleLevelUp(msg)
	case wow.ServerLootResponse:
		wc.handleLootResponse(msg)
	default:
		wc.log.Debug().
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

	if wc.cryptEnable.Load() {
		header = wc.crypt.Encrypt(header)
	}

	if len(header) != 6 {
		wc.log.Fatal().Msgf("header must be 6 bytes long, got: %d", len(header))
	}

	return header
}

// Send data to the client. It is a no-op once the connection is closed.
func (wc *WorldClient) Send(pkt *wow.Packet) {
	select {
	case wc.clientMessages <- pkt:
	case <-wc.closed:
	}
}
