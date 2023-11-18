package serworm

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/store/localdb"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ProxyServer struct {
	client *world.GameClient

	config LoginServerConfig

	ctx context.Context
	db  store.AccountRepo
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
	store := localdb.InitYamlDatabase("summit.yaml")

	//nolint:exhaustruct
	srv := &ProxyServer{
		db:     store,
		log:    log.With().Str("server", "proxy").Caller().Logger(),
		ctx:    ctx,
		config: config,
	}

	as, err := auth.NewServer(listenAddress, store, auth.WithRealmProvider(srv))
	if err != nil {
		return fmt.Errorf("cannot start auth server: %w", err)
	}

	srv.authServer = as

	srv.log.Info().Msgf("proxy server is listening on: %s", listenAddress)

	go srv.Run()

	return nil
}

func (proxy *ProxyServer) Realms(string) ([]*auth.Realm, error) {
	proxy.InitFakeRealmClient()

	return proxy.realms, nil
}

func (proxy *ProxyServer) InitFakeRealmClient() {
	if proxy.realms == nil {
		loginConn, err := net.Dial("tcp4", proxy.config.ServerAddress)
		if err != nil {
			panic(err)
		}

		client := NewRealmClient(loginConn, 0x08)

		realms, err := client.Authenticate(proxy.config.User, proxy.config.Pass)
		if err != nil {
			proxy.log.Fatal().Msg("cannot authenticate client")
		}

		proxy.log.Debug().Msgf("starting %d bridge realms", len(realms))
		proxy.startServers(realms)
	}
}

func (proxy *ProxyServer) startServers(realms []*auth.Realm) {
	proxy.realms = make([]*auth.Realm, len(realms))

	portBase := 5983
	for i, realm := range realms {
		_ = NewWorldBridge(portBase+i, realm.Address, realm.Name, proxy)
		realm.Address = fmt.Sprintf("127.0.0.1:%d", portBase+i)

		proxy.realms[i] = realm
	}
}

func (proxy *ProxyServer) AddClient(gc *world.GameClient) {
	proxy.client = gc
	proxy.log.Error().Msgf("client connected, opening bridge for: %s", gc.ID)
}

func (proxy *ProxyServer) Disconnected(id string) {
	proxy.log.Debug().Msgf("client disconnected: %s", id)
	os.Exit(0)
}

func (proxy *ProxyServer) Run() {
	defer log.Warn().Msg("proxy server stopped")

	for range proxy.ctx.Done() {
		return
	}
}

// !
// ! SessionManager methods
// !

// GetAuthSession retrives the auth session from login (auth) server.
func (ws *ProxyServer) GetAuthSession(account string) *auth.Session {
	panic("not implemented") // TODO: Implement
}

// GetCharacters fetches the character list (with full character info) from the store.
func (ws *ProxyServer) GetCharacters(account string, characters *player.Players) (err error) {
	// *characters, err = ws.characterStore.GetCharacters(account)
	// return err
	panic("not implemented") // TODO: Implement
}

// CreateCharacter saves a new character into the database.
func (ws *ProxyServer) CreateCharacter(account string, character *player.Player) error {
	// return ws.characterStore.CreateCharacter(account, character)
	panic("not implemented") // TODO: Implement
}
