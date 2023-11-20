//nolint:all
package main

import (
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/paalgyula/summit/docs"
	"github.com/paalgyula/summit/internal/store/localdb"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("server")        // name of config file (without extension)
	viper.SetConfigType("yaml")          // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/summit/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.summit") // call multiple times to add many search paths
	viper.AddConfigPath(".")             // optionally look for config in the working directory

	viper.SetDefault("log.level", -1)      // -1 Trace
	viper.SetDefault("log.format", "json") // Pretty log

	viper.SetDefault("world.listen", "127.0.0.1:5002")

	viper.SetDefault("auth.listen", "127.0.0.1:5000")
	viper.SetDefault("auth.management.enabled", false)
	viper.SetDefault("auth.management.listen", "127.0.0.1:4999")

	// TODO: gRPC transport authentication
	// viper.SetDefault("auth.management.user", "root")
	// viper.SetDefault("auth.management.pass", "oauth_token_here")

	// We have some defaults, so we can ignore the config read error.
	_ = viper.ReadInConfig() // Find and read the config file

	viper.SetEnvPrefix("summit")
	viper.AutomaticEnv()
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

	store := localdb.InitYamlDatabase("summit-store.yaml")
	defer store.SaveAll()

	rp := &auth.StaticRealmProvider{
		RealmList: []*auth.Realm{
			{
				Icon:          6,
				Lock:          0,
				Flags:         auth.RealmFlagRecommended,
				Name:          "The Highest Summit",
				Address:       "127.0.0.1:5002",
				Population:    3,
				NumCharacters: 1,
				Timezone:      8,
			},
		},
	}

	authServerOpts := []auth.ServerOption{
		auth.WithRealmProvider(rp),
	}

	if viper.GetBool("auth.management.enabled") {
		l, err := net.Listen("tcp", viper.GetString("auth.management.listen"))
		if err != nil {
			log.Fatal().Err(err).Msg("cannot create management listener")
		}

		authServerOpts = append(authServerOpts, auth.WithManagement(l))
	}

	ams := auth.NewManagementService(store)

	authListenAddr := viper.GetString("auth.listen")
	server, err := auth.NewServer(authListenAddr, ams, authServerOpts...)
	if err != nil {
		panic(err)
	}
	defer server.Close()

	// *
	// * World Server
	// *

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	worldSrv, err := world.NewServer(
		world.WithEndpoint(viper.GetString("world.listen")),
		world.WithAuthManagement(ams),
	)
	if err != nil {
		log.Fatal().Err(err).
			Msgf("cannot start world server: %s", err.Error())
	}

	if err := worldSrv.StartServer(store, store); err != nil {
		panic(err)
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
