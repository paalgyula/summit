package world

import (
	"fmt"

	. "github.com/paalgyula/summit/pkg/summit/world/packets"
	"github.com/paalgyula/summit/pkg/wow"
)

type handlePacket = func(wow.PacketData)
type handleCommand = func()

// ExternalPacketFunc register packet for external processing.
type ExternalPacketFunc = func(*GameClient, wow.OpCode, []byte)

type PacketHandler struct {
	Opcode  wow.OpCode
	Handler any
}

func (gc *GameClient) RegisterHandlers(handlers ...PacketHandler) {
	OpcodeTable.Handle(wow.ClientPing, gc.PingHandler)
	OpcodeTable.Handle(wow.ClientAuthSession, gc.AuthSessionHandler)
	OpcodeTable.Handle(wow.ClientCharEnum, gc.ListCharacters)
	OpcodeTable.Handle(wow.ClientCharCreate, gc.CreateCharacter)
	OpcodeTable.Handle(wow.ClientRealmSplit, gc.HandleRealmSplit)

	handle_count := len(handlers)

	if handle_count < int(wow.NumMsgTypes) {
		additional := int(wow.NumMsgTypes) - handle_count
		gc.log.Debug().Msgf("Adding %d empty WoW OpCode handlers", additional)
		for i := 0; i < additional; i++ {
			handlers = append(handlers, PacketHandler{
				Opcode:  wow.OpCode(handle_count + i + 1),
				Handler: "none",
			})
		}
	}

	for _, oh := range handlers {
		if len(OpcodeTable) <= int(oh.Opcode) {
			gc.log.Error().Msgf("Opcode Table missing Opcode: %v\n", oh.Opcode.String())
			// fmt.Println("opcode table too short")
			continue
		}
		OpcodeTable.Handle(oh.Opcode, oh.Handler)
	}
}

func (gc *GameClient) Handle(oc wow.OpCode, data []byte) error {
	wow.GetPacketDumper().Write(oc, data)

	handle := OpcodeTable.Get(oc)
	if handle == nil {
		// return errors.New("no handler record found")
		gc.log.Warn().Msgf("no handler record found: 0x%04x", int(oc))
		return nil
	}

	switch t := handle.Handler.(type) {
	case string:
		gc.log.Warn().Msgf("handler defined as string: %s", t)
	case handlePacket:
		t(data)
	case handleCommand:
		t()
	case ExternalPacketFunc:
		t(gc, oc, data)
	default:
		gc.log.Error().Msgf("handler function is not defined: %s", t)
		gc.log.Error().
			Type("pkt", oc).
			Str("id", fmt.Sprintf("0x%04x", oc)).
			Str("name", fmt.Sprintf("%+v", handle)).
			Msgf("no handler for the packet")
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

	return nil
}
