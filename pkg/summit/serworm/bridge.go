package serworm

import (
	"net"
	"strconv"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/summit/world/packets"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Bridge struct {
	server string
	user   string
	pass   string

	client *world.GameClient

	socket    *RealmClient
	worldConn net.Conn

	crypt *crypt.WowCrypt
	srp   *crypt.SRP6

	log zerolog.Logger
}

// HandleExternalPacket handles an external packet received from the client by writing
// it to the packet dumper and sending the packet to the bridge.
//
// client: the game client sending the packet.
// oc: the op opcode of the packet.
// data: the data block of the packet.
func (b *Bridge) HandleExternalPacket(client *world.GameClient, oc wow.OpCode, data []byte) {
	wow.GetPacketDumper().Write(oc, data)

	b.Send2Bridge(oc, data)
}

//nolint:godox,wsl
func (b *Bridge) Send2Bridge(oc wow.OpCode, data []byte) {
	wow.GetPacketDumper().Write(oc, data)

	// TODO: send to server
}

func (b *Bridge) Send2Client(oc wow.OpCode, data []byte) {
	w := wow.NewPacket(oc)

	// w.WriteB(uint16(len(data) + 2))
	// w.Write(uint16(oc))
	// header := w.Bytes()

	// if b.crypt != nil {
	// 	header = b.crypt.Encrypt(w.Bytes())
	// } else {
	// 	b.log.Error().Msg("no encryption?!")
	// }

	// data = append(header, data...)

	wow.NewPacket(oc)
	w.WriteBytes(data)

	b.client.Send(w)
}

func (b *Bridge) Start(listener net.Listener, ws world.SessionManager) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			b.log.Error().Err(err).Msg("cannot accept connection")

			continue
		}

		handlers := make([]world.PacketHandler, wow.NumMsgTypes)
		for i := 0; i < int(wow.NumMsgTypes); i++ {
			handlers[i] = world.PacketHandler{
				Opcode:  wow.OpCode(i),
				Handler: b.HandleExternalPacket,
			}
		}

		b.client = world.NewGameClient(conn, ws, nil, handlers...)
		packets.OpcodeTable.Handle(wow.ClientAuthSession, b.client.AuthSessionHandler)
	}
}

func NewBridge(listenPort int, serverAddr, serverName string, ws world.SessionManager) *Bridge {
	b := &Bridge{
		log: log.With().
			Str("name", serverName).
			Str("service", "bridge").Logger(),
	}

	listenAddr := "127.0.0.1:" + strconv.Itoa(listenPort)
	b.log.Info().Msgf("starting bridge on address: %s", listenAddr)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		b.log.Fatal().Err(err).Msg("cannot listen")
	}

	go b.Start(listener, ws)

	return b
}
