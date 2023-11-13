package serworm

import (
	"net"
	"strconv"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//nolint:unused
type WorldBridge struct {
	serverAddr string
	user       string
	pass       string

	// client *world.GameClient

	// socket    *RealmClient
	// worldConn net.Conn

	// crypt *crypt.WowCrypt

	log zerolog.Logger
}

// HandleProxy handles an external packet received from the client by writing
// it to the packet dumper and sending the packet to the upstream.
//
// client: the game client sending the packet.
// oc: the op opcode of the packet.
// data: the data block of the packet.
func (wb *WorldBridge) HandleProxy(_ *world.GameClient, oc wow.OpCode, data []byte) {
	wow.GetPacketDumper().Write(oc, data)
}

//nolint:godox,wsl
func (wb *WorldBridge) Start(listener net.Listener, sessionManager world.SessionManager) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			wb.log.Error().Err(err).Msg("cannot accept connection")

			continue
		}

		handlers := make([]world.PacketHandler, wow.NumMsgTypes)
		for i := 0; i < int(wow.NumMsgTypes); i++ {
			handlers[i] = world.PacketHandler{
				Opcode:  wow.OpCode(i),
				Handler: wb.HandleProxy,
			}
		}

		gc := world.NewGameClient(conn, sessionManager, nil, handlers...)

		_, err = NewWorldClient(gc, wb.serverAddr)
		if err != nil {
			log.Error().Err(err).Send()
		}

		// TODO: re-activate AuthSessionHandler
		// packets.OpcodeTable.Handle(wow.ClientAuthSession, wb.client.AuthSessionHandler)
	}
}

func NewWorldBridge(listenPort int, serverAddr string, serverName string, ws world.SessionManager) *WorldBridge {
	//nolint:exhaustruct
	b := &WorldBridge{
		serverAddr: serverAddr,
		log: log.With().
			Str("name", serverName).
			Str("service", "bridge").Logger(),
	}

	listenAddr := "127.0.0.1:" + strconv.Itoa(listenPort)
	b.log.Info().Msgf("starting world bridge on address: %s", listenAddr)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		b.log.Fatal().Err(err).Msg("cannot listen")
	}

	go b.Start(listener, ws)

	return b
}
