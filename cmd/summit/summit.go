//nolint:all
package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/docs"
	"github.com/paalgyula/summit/pkg/store/localdb"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().
		Caller().
		Logger()

	log.Info().
		Str("build", docs.BuildInfo()).
		Msg("Starting summit wow server")

	store := localdb.InitYamlDatabase("summit.yaml")
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

	l, err := net.Listen("tcp", "127.0.0.1:4999")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create management listener")
	}

	ams := auth.NewManagementService(store)

	server, err := auth.NewServer("0.0.0.0:5000", ams,
		auth.WithRealmProvider(rp),
		auth.WithManagement(l),
	)
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
		world.WithEndpoint("127.0.0.1:5002"),
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
