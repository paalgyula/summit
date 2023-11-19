package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/packets"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

type (
	handlePacket  = func(wow.PacketData)
	handlePacket2 = func(*wow.Packet)
	handleCommand = func()
)

// ExternalPacketFunc register packet for external processing.
type ExternalPacketFunc = func(*WorldSession, wow.OpCode, []byte)

type PacketHandler struct {
	Opcode  wow.OpCode
	Handler any
}

func (gc *WorldSession) RegisterHandlers(handlers ...PacketHandler) {
	if len(handlers) < int(wow.NumMsgTypes) {
		origSize := len(handlers)

		additional := int(wow.NumMsgTypes) - len(handlers)
		for i := 0; i < additional; i++ {
			handlers = append(handlers, PacketHandler{
				Opcode:  wow.OpCode(origSize + i + 1),
				Handler: "none",
			})
		}
	}

	for _, oh := range handlers {
		if len(packets.OpcodeTable) <= int(oh.Opcode) {
			log.Printf("opcode table too short: 0x%03x", oh.Opcode)

			continue
		}

		packets.OpcodeTable.Handle(oh.Opcode, oh.Handler)
	}

	packets.OpcodeTable.Handle(wow.ClientPing, gc.PingHandler)
	packets.OpcodeTable.Handle(wow.ClientAuthSession, gc.AuthSessionHandler)
	packets.OpcodeTable.Handle(wow.ClientCharEnum, gc.SendCharacterEnum)
	packets.OpcodeTable.Handle(wow.ClientCharCreate, gc.CreateCharacter)
	packets.OpcodeTable.Handle(wow.ClientRealmSplit, gc.HandleRealmSplit)
}

func (gc *WorldSession) Handle(pkt *wow.Packet) {
	wow.GetPacketDumper().Write(pkt.Opcode(), pkt.Bytes())

	handle := packets.OpcodeTable.Get(pkt.Opcode())
	if handle == nil {
		// return errors.New("no handler record found")
		gc.log.Warn().Msgf("no handler record found: 0x%04x", pkt.OpCode())

		return
	}

	switch t := handle.Handler.(type) {
	case string:
		gc.log.Warn().
			Str("packet", pkt.Opcode().String()).
			Str("handler", t).
			Msg("handler defined as string")
	case handlePacket:
		t(pkt.Bytes())
	case handlePacket2: // func(*wow.Packet)
		t(pkt)
	case handleCommand:
		t()
	case ExternalPacketFunc:
		t(gc, pkt.Opcode(), pkt.Bytes())
	default:
		gc.log.Error().Msgf("handler function is not defined: %s", t)
		gc.log.Error().
			Type("packet", pkt.Opcode().String()).
			Str("opcode", fmt.Sprintf("0x%04x", pkt.OpCode())).
			Str("handler", fmt.Sprintf("%+v %T", handle, handle.Handler)).
			Msgf("handler type not handled")
	}

	// switch oc {
	// case ClientPing:
	// 	gc.PingHandler()
	// case ClientAuthSession:
	// 	gc.AuthSessionHandler(data)
	// case ClientCharEnum:
	// 	gc.ListCharacters()
	// default:

	// }
}
