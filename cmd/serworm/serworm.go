package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/rs/zerolog/log"
)

var serverAddress, username, password string

func init() {
	flag.StringVar(&serverAddress, "server", "localhost:5000", "address of the logon server in host:port format")
	flag.StringVar(&username, "user", "test", "username")
	flag.StringVar(&password, "pass", "test", "password")
}

func main() {
	flag.Parse()

	br := serworm.NewBridge(serverAddress, username, password)

	// TODO: create a game client and a binary packet dumper
	br.SetGameClient(nil)

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
}
