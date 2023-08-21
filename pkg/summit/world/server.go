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
	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//nolint:gochecknoinits
func init() {
	log.Logger = log.Output(zerolog.NewConsoleWriter())
}

type Server struct {
	clients sync.Map

	ctx context.Context
	db  *db.Database
	l   net.Listener
	log zerolog.Logger

	bs *babysocket.Server
}

func StartServer(ctx context.Context, listenAddress string) error {
	db := db.GetInstance()

	//nolint:exhaustruct
	ws := &Server{
		db:  db,
		log: log.With().Str("server", "world").Caller().Logger(),
		ctx: ctx,

		clients: sync.Map{},
	}

	var err error

	ws.l, err = net.Listen("tcp4", listenAddress)
	if err != nil {
		return fmt.Errorf("world.StartServer: %w", err)
	}

	bs, err := babysocket.NewServer(ctx, "babysocket", ws)
	if err != nil {
		return fmt.Errorf("world.StartServer: %w", err)
	}

	ws.bs = bs

	ws.log.Info().Msgf("world server is listening on: %s", listenAddress)

	go ws.listenConnections()
	go ws.Run()

	return nil
}

func (ws *Server) Clients() map[string]wow.PayloadSender {
	ret := map[string]wow.PayloadSender{}

	ws.clients.Range(func(key, value any) bool {
		v, _ := value.(*GameClient)
		ck, _ := key.(string)

		ret[ck] = v

		return true
	})

	return ret
}

func (ws *Server) listenConnections() {
	for {
		conn, err := ws.l.Accept()
		if err != nil {
			log.Error().Err(err).Msg("listener error")

			continue
		}

		NewGameClient(conn, ws, ws.bs)
	}
}

func (ws *Server) AddClient(gc *GameClient) {
	ws.clients.Store(gc.ID, gc)

	count := 0

	ws.clients.Range(func(key, value any) bool {
		count++

		return true
	})

	ws.log.Debug().Int("clients", count).
		Msgf("client added to set with id: %s", gc.ID)
}

func (ws *Server) Disconnected(id string) {
	ws.clients.Delete(id)
	ws.log.Debug().Msgf("client disconnected: %s", id)
}

func (ws *Server) Stats() {
	ws.log.Debug().Msgf(MemUsage())
}

func (ws *Server) Run() {
	ticker := time.NewTicker(time.Second * 20)

	defer ws.db.SaveAll()
	defer ws.l.Close()
	defer log.Warn().Msg("world server stopped")

	for {
		select {
		case <-ticker.C:
			// log.Info().Msg("Garbage collector timer: unimplemented")
			// ws.Stats()
			ws.db.SaveAll()
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
