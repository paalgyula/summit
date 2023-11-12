//nolint:all
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/docs"
	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	log.Info().
		Str("build", docs.BuildInfo()).
		Msg("Starting summit wow server")
	db.InitYamlDatabase()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := world.StartServer(ctx, "0.0.0.0:5002"); err != nil {
		panic(err)
	}

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

	server, err := auth.NewServer("0.0.0.0:5000", auth.WithRealmProvider(rp))
	if err != nil {
		panic(err)
	}
	defer server.Close()

	done := make(chan bool, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh

		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	<-done

	log.Info().Msg("Shutting down")
	db.GetInstance().SaveAll()
}
