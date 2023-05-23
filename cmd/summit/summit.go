package main

import (
	"os"

	"github.com/paalgyula/summit/pkg/blizzard/auth"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	log.Info().Msg("Starting summit wow server")

	err := auth.StartServer("0.0.0.0:5000")
	panic(err)
}
