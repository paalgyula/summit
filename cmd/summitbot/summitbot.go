package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/paalgyula/summit/pkg/summit/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	account     string
	password    string
	logonServer string
	worldServer string
	timeout     int
)

func main() {
	flag.StringVar(&account, "account", "test", "wow account name")
	flag.StringVar(&password, "password", "test", "wow account name")
	flag.StringVar(&logonServer, "logon", "logon.warmane.com:3724", "logon server address")
	flag.StringVar(&worldServer, "world", "51.178.64.97:8091", "world server's address")

	flag.IntVar(&timeout, "timeout", 15, "sets the idle timeout of the bot")

	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: ""})

	conn, err := net.Dial("tcp", logonServer)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to server")
	}

	rc := client.NewRealmClient(conn, 12340)

	realms, err := rc.Authenticate(account, password)
	if err != nil {
		log.Fatal().Err(err).Msg("auth failed")
	}

	log.Logger = log.With().Str("acc", account).Logger()

	log.Debug().Msg("login success")

	// List out realms
	for i, r := range realms {
		characters := ""
		if r.NumCharacters > 0 {
			characters = fmt.Sprintf("(%d)", r.NumCharacters)
		}

		log.Info().
			Str("name", r.Name).
			Str("address", r.Address).
			Int("characters", int(r.NumCharacters)).
			Msgf("#%02d - %s %s %s", i, r.Name, characters, r.Address)
	}

	log.Info().Str("server", worldServer).
		Msg("Trying to login to the world server")

	wc, err := client.NewWorldClient(account, rc.SessionKey.Text(16), worldServer)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to world server")
	}

	time.Sleep(time.Duration(timeout) * time.Second)

	err = wc.Disconnect()
	if err != nil {
		log.Fatal().Err(err).Msg("disconnected")
	}
}
