package serworm

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/paalgyula/summit/internal/store/mongostore"
	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/client"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ProxyServer struct {
	client *world.WorldSession

	config LoginServerConfig

	ctx context.Context
	db  store.AccountRepo
	log zerolog.Logger

	authServer     *auth.Server
	authManagement auth.ManagementService
	charStore      store.CharacterRepo

	// Proxy credentials for upstream auth
	accountName string
	sessionKey  string

	realms []*auth.Realm
}

type LoginServerConfig struct {
	ServerAddress string
	User          string
	Pass          string
}

// ProxyStoreConfig configures the MongoDB store for the proxy.
type ProxyStoreConfig struct {
	URI      string
	Database string
}

func StartProxy(ctx context.Context, listenAddress string, config LoginServerConfig, storeCfg ProxyStoreConfig) error {
	if storeCfg.URI == "" {
		storeCfg.URI = "mongodb://admin:admin@localhost:27017"
	}
	if storeCfg.Database == "" {
		storeCfg.Database = "summit"
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, err := mongostore.Connect(connectCtx, storeCfg.URI, storeCfg.Database)
	if err != nil {
		return fmt.Errorf("cannot connect to MongoDB: %w", err)
	}

	ms := auth.NewManagementService(db)

	//nolint:exhaustruct
	srv := &ProxyServer{
		db: db,
		log: log.With().
			Str("service", "proxy").
			Caller().
			Logger(),
		ctx:            ctx,
		config:         config,
		charStore:      db,
		authManagement: ms,
	}

	as, err := auth.NewServer(listenAddress, ms, auth.WithRealmProvider(srv))
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
	if proxy.realms != nil {
		return
	}

	loginConn, err := net.Dial("tcp4", proxy.config.ServerAddress)
	if err != nil {
		panic(err)
	}

	rc := client.NewRealmClient(loginConn, 0x08)

	realms, err := rc.Authenticate(proxy.config.User, proxy.config.Pass)
	if err != nil {
		proxy.log.Fatal().Msg("cannot authenticate client")
	}

	// Store session key and account name for world bridges
	proxy.sessionKey = rc.SessionKey.Text(16)
	proxy.accountName = strings.ToUpper(proxy.config.User)

	proxy.log.Info().
		Str("account", proxy.accountName).
		Msgf("proxy authenticated with upstream, starting %d bridge realms", len(realms))

	proxy.startServers(realms)
}

func (proxy *ProxyServer) startServers(realms []*auth.Realm) {
	proxy.realms = make([]*auth.Realm, len(realms))

	portBase := 5983
	for i, realm := range realms {
		_ = NewWorldBridge(portBase+i, realm.Address, realm.Name, proxy, proxy.accountName, proxy.sessionKey)
		realm.Address = fmt.Sprintf("127.0.0.1:%d", portBase+i)

		proxy.realms[i] = realm
	}
}

func (proxy *ProxyServer) AddClient(gc *world.WorldSession) {
	proxy.client = gc
	proxy.log.Info().Msgf("client connected: %s", gc.ID)
}

func (proxy *ProxyServer) Disconnected(_ *world.WorldSession, reason string) {
	proxy.log.Debug().Msgf("client disconnected: %s", reason)
	os.Exit(0)
}

func (proxy *ProxyServer) Run() {
	defer log.Warn().Msg("proxy server stopped")

	for range proxy.ctx.Done() {
		return
	}
}

// SessionManager interface implementation

func (proxy *ProxyServer) GetAuthSession(account string) *auth.Session {
	proxy.log.Trace().Msgf("requesting auth session for account: %s", account)

	sess := proxy.authManagement.GetSession(account)

	if sess != nil {
		proxy.log.Trace().
			Str("account", strings.ToLower(account)).
			Msg("session found")
	}

	return sess
}

func (proxy *ProxyServer) GetCharacters(account string, characters *player.Players) error {
	chars, err := proxy.charStore.GetCharacters(account)
	if err != nil {
		return err
	}

	for _, c := range chars {
		characters.Add(c)
	}

	return nil
}

func (proxy *ProxyServer) GetCharacter(guid uint32) (*player.Player, error) {
	return proxy.charStore.GetCharacter(guid)
}

func (proxy *ProxyServer) CreateCharacter(account string, character *player.Player) error {
	return proxy.charStore.CreateCharacter(account, character)
}
