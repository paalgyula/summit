package world

import (
	"fmt"

	"github.com/rs/zerolog/log"

	// . "github.com/paalgyula/summit/pkg/summit/world/packets"
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
type PacketHandleTable [wow.NumMsgTypes]*PacketHandler

func (pPht *PacketHandleTable) Get(code wow.OpCode) (*PacketHandler, error) {
	if code >= wow.NumMsgTypes {
		return nil, fmt.Errorf("Out of bounds code: %s", code.String())
	}

	return pPht[code], nil
}

func (pPht *PacketHandleTable) Set(code wow.OpCode, handler any) {

	packet_handler, err := pPht.Get(code)

	if err != nil {
		log.Fatal().Msgf("Attempting to set out of bounds code: %s", code.String())
		return
	}

	if packet_handler == nil {
		packet_handler = &PacketHandler{Opcode: code, Handler: nil}
		pPht[code] = packet_handler
	}

	packet_handler.Handler = handler
}

func (gc *GameClient) RegisterHandlers(custom_handlers ...PacketHandler) {

	// Default Handlers

	gc.pht.Set(wow.ClientPing, gc.PingHandler)
	gc.pht.Set(wow.ClientAuthSession, gc.AuthSessionHandler)
	gc.pht.Set(wow.ClientCharEnum, gc.ListCharacters)
	gc.pht.Set(wow.ClientCharCreate, gc.CreateCharacter)
	gc.pht.Set(wow.ClientRealmSplit, gc.HandleRealmSplit)

	// Then Custom Overrides

	for _, ch := range custom_handlers {
		gc.pht.Set(ch.Opcode, ch.Handler)
	}

}

func (gc *GameClient) RegisterHandler(custom_handler PacketHandler) {
	gc.pht.Set(custom_handler.Opcode, custom_handler.Handler)
}

func (gc *GameClient) Handle(oc wow.OpCode, data []byte) error {
	wow.GetPacketDumper().Write(oc, data)

	handle, err := gc.pht.Get(oc)
	if handle == nil || err != nil {
		// return errors.New("no handler record found")
		gc.log.Warn().Msgf("no handler record found: 0x%04x", int(oc))
		return nil
	}

	switch t := handle.Handler.(type) {
	case string:
		gc.log.Warn().
			Type("pkt", oc).
			Str("id", oc.String()).
			Str("name", fmt.Sprintf("%+v", handle)).
			Msgf("handler defined as string: %s", t)
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
