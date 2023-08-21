//nolint:all
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/rs/zerolog/log"
)

//nolint:gochecknoglobals
var listenAddress, serverAddress, username, password string

//nolint:gochecknoinits
func init() {
	flag.StringVar(&listenAddress, "listen", "localhost:5000", "address where to listen")
	// Sorry, and thank you for your support guys!
	flag.StringVar(&serverAddress, "server", "logon.warmane.com:3724", "address of the logon server in host:port format")
	flag.StringVar(&username, "user", "test", "username")
	flag.StringVar(&password, "pass", "test", "password")
}

func main() {
	flag.Parse()

	ctx := context.Background()

	err := serworm.StartProxy(ctx, listenAddress, serworm.LoginServerConfig{
		ServerAddress: serverAddress,
		User:          username,
		Pass:          password,
	})
	if err != nil {
		panic(err)
	}

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
