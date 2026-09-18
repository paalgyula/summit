package serworm

import (
	"net"
	"strconv"

	"github.com/paalgyula/summit/pkg/summit/client"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// WorldBridge proxies packets between a game client and an upstream world server.
type WorldBridge struct {
	serverAddr string
	log        zerolog.Logger

	// Proxy credentials for upstream authentication
	accountName string
	sessionKey  string
}

func (wb *WorldBridge) Start(listener net.Listener, sessionManager world.SessionManager) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			wb.log.Error().Err(err).Msg("cannot accept connection")

			continue
		}

		gc := world.NewWorldSession(conn, sessionManager)

		wc, err := client.NewWorldClient(wb.accountName, wb.sessionKey, wb.serverAddr)
		if err != nil {
			wb.log.Error().Err(err).Msg("cannot connect to upstream world server")
			gc.Close()

			continue
		}

		wb.log.Info().
			Str("account", wb.accountName).
			Str("upstream", wb.serverAddr).
			Msg("bridge connection established")

		// Forward all upstream packets to the game client
		wc.SetForwardHandler(func(opcode wow.OpCode, data []byte) {
			wow.GetPacketDumper().Write(opcode, data)
			gc.Send(wow.NewPacketWithData(opcode, data))
		})

		// Forward all client packets to upstream
		handlers := make([]world.PacketHandler, wow.NumMsgTypes)
		for i := 0; i < int(wow.NumMsgTypes); i++ {
			handlers[i] = world.PacketHandler{
				Opcode: wow.OpCode(i),
				Handler: world.ExternalPacketFunc(func(_ *world.WorldSession, oc wow.OpCode, data []byte) {
					wow.GetPacketDumper().Write(oc, data)
					wc.Send(wow.NewPacketWithData(oc, data))
				}),
			}
		}

		gc.RegisterHandlers(handlers...)
	}
}

func NewWorldBridge(listenPort int, serverAddr string, serverName string, ws world.SessionManager, accountName, sessionKey string) *WorldBridge {
	//nolint:exhaustruct
	b := &WorldBridge{
		serverAddr:  serverAddr,
		accountName: accountName,
		sessionKey:  sessionKey,
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
