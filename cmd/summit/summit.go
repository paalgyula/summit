package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	log.Info().Msg("Starting summit wow server")
	db.InitYamlDatabase()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := world.StartServer(ctx, "0.0.0.0:5002"); err != nil {
		panic(err)
	}

	server, err := auth.NewServer("0.0.0.0:5000")
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
