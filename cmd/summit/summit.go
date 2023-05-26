package main

import (
	"context"
	"os"

	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	log.Info().Msg("Starting summit wow server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := world.StartServer(ctx, "0.0.0.0:5002"); err != nil {
		panic(err)
	}

	err := auth.StartServer("0.0.0.0:5000")
	panic(err)
}
