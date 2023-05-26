package world

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.NewConsoleWriter())
}

type WorldServer struct {
	clients sync.Map

	ctx context.Context
	db  *db.Database
	l   net.Listener
	log zerolog.Logger
}

func StartServer(ctx context.Context, listenAddress string) (err error) {
	db := db.GetInstance()

	ws := WorldServer{
		db:  db,
		log: log.With().Str("server", "world").Caller().Logger(),
		ctx: ctx,

		clients: sync.Map{},
	}

	ws.l, err = net.Listen("tcp4", listenAddress)
	if err != nil {
		return fmt.Errorf("world.StartServer: %w", err)
	}

	ws.log.Info().Msgf("world server is listening on: %s", listenAddress)

	go ws.listenConnections()
	go ws.Run()

	return nil
}

func (ws *WorldServer) listenConnections() {
	for {
		conn, err := ws.l.Accept()
		if err != nil {
			log.Error().Err(err).Msg("cannot accept connection")

			continue
		}

		NewGameClient(conn, ws)
	}
}

func (ws *WorldServer) AddClient(gc *GameClient) {
	ws.clients.Store(gc.ID, gc)

	count := 0
	ws.clients.Range(func(key, value any) bool {
		count++
		return true
	})

	ws.log.Debug().Msgf("client added to set with id: %s", gc.ID)
}

func (ws *WorldServer) Disconnected(id string) {
	ws.clients.Delete(id)
	ws.log.Debug().Msgf("client disconnected: %s", id)
}

func (ws *WorldServer) Stats() {
	ws.log.Debug().Msgf(MemUsage())
}

func (ws *WorldServer) Run() {
	ticker := time.NewTicker(time.Second * 20)

	defer ws.db.SaveAll()
	defer ws.l.Close()
	defer log.Warn().Msg("world server stopped")

	for {
		select {
		case <-ticker.C:
			// log.Info().Msg("Garbage collector timer: unimplemented")
			// ws.Stats()
		case <-ws.ctx.Done():
			return
		}
	}
}

func MemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var bb bytes.Buffer

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Fprintf(&bb, "Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Fprintf(&bb, " TotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Fprintf(&bb, " Sys = %v MiB", bToMb(m.Sys))
	fmt.Fprintf(&bb, " NumGC = %v", m.NumGC)

	return bb.String()
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}