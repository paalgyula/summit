//nolint:all
package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/paalgyula/summit/docs"
	"github.com/paalgyula/summit/internal/store/mongostore"
	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/auth"
	authws "github.com/paalgyula/summit/pkg/summit/auth/ws"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	// godotenv loads .env into os.Environ — viper picks it up via AutomaticEnv
	_ = godotenv.Load()

	viper.SetConfigName("server")        // name of config file (without extension)
	viper.SetConfigType("yaml")          // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/summit/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.summit") // call multiple times to add many search paths
	viper.AddConfigPath(".")             // optionally look for config in the working directory

	viper.SetDefault("debug", false)
	viper.SetDefault("log.level", -1)      // -1 Trace
	viper.SetDefault("log.format", "json") // Pretty log

	viper.SetDefault("world.listen", "127.0.0.1:8129")
	viper.SetDefault("world.ws_listen", "127.0.0.1:5002")
	viper.SetDefault("world.dbc_path", "dbc")

	viper.SetDefault("auth.listen", "127.0.0.1:5000")
	viper.SetDefault("auth.ws_listen", "127.0.0.1:5001")
	viper.SetDefault("auth.management.enabled", false)
	viper.SetDefault("auth.management.listen", "127.0.0.1:4999")
	// Public base URL the web client reaches the servers through (ingress),
	// e.g. "wss://summit.dev.pilab.hu". Empty: realms advertise their own listen address.
	viper.SetDefault("auth.public_url", "")
	// Shared secret for the gRPC management API (world servers, summitctl).
	viper.SetDefault("auth.management.token", "")
	// Seconds a realm stays "online" in the list without a status report.
	viper.SetDefault("auth.realm_ttl", 30)

	// MongoDB store configuration
	viper.SetDefault("store.mongodb.uri", "mongodb://admin:admin@localhost:27017")
	viper.SetDefault("store.mongodb.database", "summit")

	// Roles: the same binary runs the login server, the world server, or both.
	viper.SetDefault("auth.enabled", true)
	viper.SetDefault("world.enabled", true)
	// A world server in its own process validates sessions and reports its
	// realm status through the auth server's management API.
	viper.SetDefault("world.auth_management.address", "")
	viper.SetDefault("world.auth_management.token", "")
	// This world server's realm-list identity.
	viper.SetDefault("world.realm.name", "The Highest Summit")
	viper.SetDefault("world.realm.slug", "highest-summit")
	viper.SetDefault("world.realm.address", "")
	viper.SetDefault("world.realm.max_players", 1000)

	// We have some defaults, so we can ignore the config read error.
	_ = viper.ReadInConfig() // Find and read the config file

	viper.SetEnvPrefix("summit")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // SUMMIT_AUTH_PUBLIC_URL -> auth.public_url
	viper.AutomaticEnv()

	// Bind DEBUG env var (no prefix — reads plain DEBUG from .env)
	viper.BindEnv("debug", "DEBUG") //nolint:errcheck

	setupLogger()
}

func setupLogger() {
	debug := viper.GetBool("debug")
	if !debug {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

		return
	}

	// Pretty colored console output for humans
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05.000"}
	log.Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(zerolog.DebugLevel)
}

// realmConfig is one entry of the "realms" list in summit.yaml.
type realmConfig struct {
	Name    string `mapstructure:"name"`
	Slug    string `mapstructure:"slug"`
	Address string `mapstructure:"address"` // world server address (host:port), for direct connections
}

// loadRealms builds the realm list from configuration, falling back to a
// single local realm pointing at this process's world WebSocket listener.
func loadRealms() []*auth.Realm {
	var cfgs []realmConfig
	if err := viper.UnmarshalKey("realms", &cfgs); err != nil {
		log.Warn().Err(err).Msg("invalid realms configuration")
	}

	if len(cfgs) == 0 {
		cfgs = []realmConfig{{Name: "The Highest Summit", Slug: "highest-summit", Address: viper.GetString("world.ws_listen")}}
	}

	realms := make([]*auth.Realm, 0, len(cfgs))
	for i, c := range cfgs {
		flags := auth.RealmFlagNone
		if i == 0 {
			flags = auth.RealmFlagRecommended
		}
		realms = append(realms, &auth.Realm{
			Icon:          6,
			Flags:         flags,
			Name:          c.Name,
			Slug:          c.Slug,
			Address:       c.Address,
			Population:    1,
			NumCharacters: 1,
			Timezone:      8,
		})
	}
	return realms
}

func main() {
	logLevel := viper.GetInt("log.level")
	zerolog.SetGlobalLevel(zerolog.Level(logLevel))

	// Setup pretty console logger if enabled
	if strings.ToLower(viper.GetString("log.format")) == "pretty" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().Caller().Logger()
	}

	log.Info().
		Str("branch", docs.Branch).
		Str("version", docs.Version).
		Msg("Starting summit wow server")

	// Initialize MongoDB store
	uri := viper.GetString("store.mongodb.uri")
	dbName := viper.GetString("store.mongodb.database")

	log.Info().Str("uri", uri).Str("db", dbName).Msg("connecting to MongoDB")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ms, err := mongostore.Connect(ctx, uri, dbName)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to MongoDB")
	}

	accRepo := store.AccountRepo(ms)
	charRepo := store.CharacterRepo(ms)
	worldRepo := store.WorldRepo(mongostore.NewWorldStore(ms.DB()))

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ms.Close(ctx)
	}()

	ams := auth.NewManagementService(accRepo)
	// Static realms are listed as offline until their world server reports in.
	ams.SetRealmRegistry(auth.NewRealmRegistry(loadRealms(), time.Duration(viper.GetInt("auth.realm_ttl"))*time.Second))

	// *
	// * Login (auth) server
	// *
	if viper.GetBool("auth.enabled") {
		authServerOpts := []auth.ServerOption{
			auth.WithRealmProvider(ams.RealmRegistry()),
			auth.WithManagementToken(viper.GetString("auth.management.token")),
		}

		if viper.GetBool("auth.management.enabled") {
			l, err := net.Listen("tcp", viper.GetString("auth.management.listen"))
			if err != nil {
				log.Fatal().Err(err).Msg("cannot create management listener")
			}

			authServerOpts = append(authServerOpts, auth.WithManagement(l))
		}

		server, err := auth.NewServer(viper.GetString("auth.listen"), ams, authServerOpts...)
		if err != nil {
			panic(err)
		}
		defer server.Close()

		authWSServer := authws.NewServer(ams, ams.RealmRegistry(), authws.WithPublicURL(viper.GetString("auth.public_url")))
		go func() {
			if err := authWSServer.Start(viper.GetString("auth.ws_listen")); err != nil {
				log.Warn().Err(err).Msg("auth websocket server closed")
			}
		}()
		defer authWSServer.Close()
	}

	// *
	// * World Server
	// *
	if viper.GetBool("world.enabled") {
		// In-process auth by default; a remote auth server over gRPC when configured
		var management auth.ManagementService = ams
		if addr := viper.GetString("world.auth_management.address"); addr != "" {
			client, err := auth.NewManagementClient(addr, viper.GetString("world.auth_management.token"))
			if err != nil {
				log.Fatal().Err(err).Str("address", addr).Msg("cannot connect to auth management API")
			}
			defer client.Close()
			management = client
			log.Info().Str("address", addr).Msg("validating sessions through the auth management API")
		}

		realmAddress := viper.GetString("world.realm.address")
		if realmAddress == "" {
			realmAddress = viper.GetString("world.ws_listen")
		}

		worldOptions := []world.ServerOption{
			world.WithEndpoint(viper.GetString("world.listen")),
			world.WithAuthManagement(management),
			world.WithRealmIdentity(world.RealmIdentity{
				Name:       viper.GetString("world.realm.name"),
				Slug:       viper.GetString("world.realm.slug"),
				Address:    realmAddress,
				MaxPlayers: uint32(viper.GetInt("world.realm.max_players")),
				Icon:       6,
				Timezone:   8,
			}),
			world.WithBabySocket(),
			world.WithStaticBaseData(),
		}

		// Spell.dbc & friends: without them no spell can be cast
		if dbcPath := viper.GetString("world.dbc_path"); dbcPath != "" {
			worldOptions = append(worldOptions, world.WithSpellDBC(dbcPath))
		} else {
			log.Warn().Msg("world.dbc_path is not set: spell data unavailable, casts will fail")
		}

		worldSrv, err := world.NewServer(worldOptions...)
		if err != nil {
			log.Fatal().Err(err).
				Msgf("cannot start world server: %s", err.Error())
		}

		if err := worldSrv.StartServer(worldRepo, charRepo); err != nil {
			panic(err)
		}

		// Start Realm WebSocket server
		if err := worldSrv.StartWebSocketServer(viper.GetString("world.ws_listen")); err != nil {
			log.Error().Err(err).Msg("cannot start realm websocket server")
		}
	}

	done := make(chan bool, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Info().Msgf("signal received: %s - graceful shutdown initiated", sig.String())

		done <- true
	}()

	<-done

	log.Info().Msg("closing assets and shutting down the emulator")
}
