package serworm

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ProxyServer struct {
	client *world.GameClient

	ctx context.Context
	db  *db.Database
	l   net.Listener
	log zerolog.Logger
}

func StartProxy(ctx context.Context, listenAddress string) (err error) {
	db := db.GetInstance()

	ws := ProxyServer{
		db:  db,
		log: log.With().Str("server", "proxy").Caller().Logger(),
		ctx: ctx,
	}

	ws.l, err = net.Listen("tcp4", listenAddress)
	if err != nil {
		return fmt.Errorf("world.StartProxy: %w", err)
	}

	ws.log.Info().Msgf("proxy server is listening on: %s TODO: regiser in realm server", listenAddress)

	go ws.listen()
	go ws.Run()

	return nil
}

func (ws *ProxyServer) listen() {
	conn, err := ws.l.Accept()
	if err != nil {
		log.Error().Err(err).Msg("cannot accept connection")
	}

	bridge := NewBridge("logon.warmane.com:3724", "gmgoofy", "0027472")

	handlers := make([]world.PacketHandler, 0xffff)
	for i := 0; i < 0xffff; i++ {
		handlers[i] = world.PacketHandler{
			Opcode:  wow.OpCode(i),
			Handler: bridge.HandleExternalPacket,
		}
	}

	gc := world.NewGameClient(conn, ws, nil, handlers...)
	bridge.SetGameClient(gc)

}

func (ws *ProxyServer) AddClient(gc *world.GameClient) {
	ws.client = gc
	ws.log.Error().Msgf("client connected, opening bridge for: %s", gc.ID)
}

func (ws *ProxyServer) Disconnected(id string) {
	ws.log.Debug().Msgf("client disconnected: %s", id)
	os.Exit(0)
}

func (ws *ProxyServer) Run() {
	defer ws.l.Close()
	defer log.Warn().Msg("proxy server stopped")

	for {
		select {
		case <-ws.ctx.Done():
			return
		}
	}
}
