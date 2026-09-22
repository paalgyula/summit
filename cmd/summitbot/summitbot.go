package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/paalgyula/summit/pkg/summit/bot"
	"github.com/paalgyula/summit/pkg/summit/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	loginTimeout = 30 * time.Second
	worldTimeout = 30 * time.Second
)

var (
	account     string
	password    string
	logonServer string
	worldServer string
	character   string
	timeout     int
)

func main() {
	flag.StringVar(&account, "account", "test", "wow account name")
	flag.StringVar(&password, "password", "test", "wow account password")
	flag.StringVar(&logonServer, "logon", "logon.warmane.com:3724", "logon server address")
	flag.StringVar(&worldServer, "world", "51.178.64.97:8091", "world server's address")
	flag.StringVar(&character, "character", "", "character name to enter the world with (default: first)")

	flag.IntVar(&timeout, "timeout", 15, "how long the bot stays in the world, in seconds")

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
		Msg("connecting to the world server")

	wc, err := client.NewWorldClient(account, rc.SessionKey.Text(16), worldServer)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to world server")
	}

	defer func() {
		if err := wc.Disconnect(); err != nil {
			log.Warn().Err(err).Msg("cannot disconnect from world server")
		}
	}()

	loginCtx, cancelLogin := context.WithTimeout(context.Background(), loginTimeout)
	defer cancelLogin()

	chars, err := wc.WaitForCharacters(loginCtx)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot fetch characters")
	}

	for i, c := range chars {
		log.Info().
			Str("name", c.Name).
			Uint8("level", c.Level).
			Str("location", fmt.Sprintf("%d (%.2f, %.2f, %.2f)", c.Location.Map, c.Location.X, c.Location.Y, c.Location.Z)).
			Msgf("#%02d - %s", i, c.Name)
	}

	worldCtx, cancelWorld := context.WithTimeout(context.Background(), worldTimeout)
	defer cancelWorld()

	player, err := wc.EnterWorld(worldCtx, character)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot enter the world")
	}

	// Wait for the rest of the login sequence (spellbook, action bar) so the
	// snapshot below is complete. Fall back to what EnterWorld gave us if the
	// server never sends it.
	if snapshot, waitErr := wc.WaitForInitialState(worldCtx); waitErr == nil {
		player = snapshot
	} else {
		log.Warn().Err(waitErr).Msg("initial world state incomplete")
	}

	log.Info().
		Str("name", player.Name).
		Uint32("map", player.Location.Map).
		Float32("x", player.Location.X).
		Float32("y", player.Location.Y).
		Float32("z", player.Location.Z).
		Int("spells", len(player.KnownSpells)).
		Int("actions", len(player.Actions)).
		Msg("player is in the world")

	playCtx, cancelPlay := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancelPlay()

	ai := bot.New(wc, bot.DefaultConfig())

	log.Info().Int("seconds", timeout).Msg("bot is playing the game")

	if err := ai.Run(playCtx); err != nil {
		log.Warn().Err(err).Msg("bot stopped")
	}

	log.Info().Int("kills", ai.Kills()).Msg("bot finished")
}
