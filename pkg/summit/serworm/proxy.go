package serworm

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ProxyServer struct {
	client *world.GameClient

	config LoginServerConfig

	ctx context.Context
	db  *db.Database
	log zerolog.Logger

	// bridge     *Bridge
	authServer *auth.Server

	realms []*auth.Realm
}

type LoginServerConfig struct {
	ServerAddress string
	User          string
	Pass          string
}

func StartProxy(ctx context.Context, listenAddress string, config LoginServerConfig) error {
	db := db.GetInstance()

	//nolint:exhaustruct
	srv := &ProxyServer{
		db:     db,
		log:    log.With().Str("server", "proxy").Caller().Logger(),
		ctx:    ctx,
		config: config,
	}

	as, err := auth.NewServer(listenAddress, srv)
	if err != nil {
		return fmt.Errorf("cannot start auth server: %w", err)
	}

	srv.authServer = as

	srv.log.Info().Msgf("proxy server is listening on: %s", listenAddress)

	go srv.Run()

	return nil
}

func (ws *ProxyServer) Realms(string) ([]*auth.Realm, error) {
	ws.InitFakeClient()

	return ws.realms, nil
}

func (ws *ProxyServer) InitFakeClient() {
	if ws.realms == nil {
		loginConn, err := net.Dial("tcp4", ws.config.ServerAddress)
		if err != nil {
			panic(err)
		}

		client := NewRealmClient(loginConn, 0x08)

		realms, err := client.Authenticate(ws.config.User, ws.config.Pass)
		if err != nil {
			ws.log.Fatal().Msg("cannot authenticate client")
		}

		ws.log.Debug().Msgf("starting %d bridge realms", len(realms))
		ws.startServers(realms)
	}
}

func (ws *ProxyServer) startServers(realms []*auth.Realm) {
	ws.realms = make([]*auth.Realm, len(realms))

	portBase := 5983
	for i, realm := range realms {
		_ = NewBridge(portBase+i, realm.Address, realm.Name, ws)
		realm.Address = fmt.Sprintf("127.0.0.1:%d", portBase+i)

		ws.realms[i] = realm
	}
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
	defer log.Warn().Msg("proxy server stopped")

	for range ws.ctx.Done() {
		return
	}
}
