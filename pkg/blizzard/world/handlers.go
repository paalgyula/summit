package world

import (
	"fmt"

	. "github.com/paalgyula/summit/pkg/blizzard/world/packets"
	"github.com/paalgyula/summit/pkg/wow"
)

type handlePacket = func([]byte)
type handleCommand = func()

// ExternalPacketFunc register packet for external processing.
type ExternalPacketFunc = func(*GameClient, OpCode, []byte)

type PacketHandler struct {
	Opcode  OpCode
	Handler any
}

func (gc *GameClient) RegisterHandlers(handlers ...PacketHandler) {
	OpcodeTable.Handle(ClientPing, gc.PingHandler)
	OpcodeTable.Handle(ClientAuthSession, gc.AuthSessionHandler)
	OpcodeTable.Handle(ClientCharEnum, gc.ListCharacters)
	OpcodeTable.Handle(ClientCharCreate, gc.CreateCharacter)

	for _, oh := range handlers {
		OpcodeTable.Handle(oh.Opcode, oh.Handler)
	}
}

func (gc *GameClient) Handle(oc OpCode, data []byte) error {
	wow.GetPacketDumper().Write(oc.Int(), data)

	handle := OpcodeTable.Get(oc.Int())
	if handle == nil {
		// return errors.New("no handler record found")
		gc.log.Warn().Msgf("no handler record found: 0x%04x", oc.Int())
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
			Str("pkt", oc.String()).
			Str("id", fmt.Sprintf("0x%04x", oc.Int())).
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
