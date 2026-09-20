package babysocket

import (
	"encoding/gob"
	"net"

	"github.com/rs/zerolog/log"
)

type socketClient struct {
	conn net.Conn
	s    *Server
	id   string
}

func (sc *socketClient) Listen() {
	defer sc.disconnected()

	var data DataPacket

	dec := gob.NewDecoder(sc.conn)

	for {
		err := dec.Decode(&data)
		if err != nil {
			log.Err(err).Msg("babysocket listener error")

			return
		}

		sc.s.log.Trace().
			Str("id", sc.id).
			Str("command", data.Command.String()).
			Str("source", data.Source).
			Str("target", data.Target).
			Int("opcode", data.Opcode).
			Int("size", data.Size).
			Msg("received packet from baby client")

		switch data.Command {
		case CommandPacket:
			sc.handlePacket(&data)
		case CommandInstruction:
			sc.handleInstruction(&data)
		case CommandResponse:
			sc.handleResponse(&data)
		default:
			sc.s.log.Warn().Msgf("unknown command type: %d", data.Command)
		}
	}
}

func (sc *socketClient) handlePacket(data *DataPacket) {
	switch {
	case data.Target == "*":
		// Broadcast to all game clients
		sc.s.log.Debug().
			Int("opcode", data.Opcode).
			Msg("broadcasting packet to all game clients")
		sc.s.SendToAll(data.Opcode, data.Data)
	case data.Target == "":
		// Broadcast to all baby clients (except sender)
		sc.s.log.Debug().
			Int("opcode", data.Opcode).
			Msg("broadcasting packet to all baby clients")
		sc.s.SendPacketToBabies(sc.id, data.Opcode, data.Data)
	default:
		// Send to specific baby client by ID
		sc.s.log.Debug().
			Str("target", data.Target).
			Int("opcode", data.Opcode).
			Msg("sending packet to specific baby client")
		sc.s.SendToBaby(data.Target, data.Opcode, data.Data)
	}
}

func (sc *socketClient) handleInstruction(data *DataPacket) {
	sc.s.log.Warn().
		Str("source", data.Source).
		Str("target", data.Target).
		Msg("received instruction command (not implemented)")
}

func (sc *socketClient) handleResponse(data *DataPacket) {
	sc.s.log.Warn().
		Str("source", data.Source).
		Str("target", data.Target).
		Msg("received response command (not implemented)")
}

func (sc *socketClient) disconnected() {
	sc.conn.Close()
	sc.s.removeClient(sc.id)
}
