package world

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	clients sync.Map

	gameListener net.Listener
	log          zerolog.Logger

	// Database access
	charStore  store.CharacterRepo
	worldStore store.WorldRepo

	bs *babysocket.Server

	// Management client for auth. Can be direct, or gRPC based.
	authManagement auth.ManagementService

	baseData *basedata.Store
}

func NewServer(opts ...ServerOption) (*Server, error) {
	worldServer := new(Server)

	worldServer.log = log.With().
		Str("service", "world").
		Caller().Logger()
	worldServer.clients = sync.Map{}

	// Apply options
	for _, so := range opts {
		if err := so(worldServer); err != nil {
			return nil, err
		}
	}

	if worldServer.gameListener == nil {
		//nolint:gosec
		l, err := net.Listen("tcp", ":8129") // Create default listener
		if err != nil {
			return nil, fmt.Errorf("gameserver listen error: %w", err)
		}

		worldServer.gameListener = l
	}

	return worldServer, nil
}

func (ws *Server) StartServer(worldStore store.WorldRepo, charStore store.CharacterRepo) error {
	ws.log.Info().Msgf("world server is listening on: %s", ws.gameListener.Addr().String())

	ws.charStore = charStore
	ws.worldStore = worldStore

	go ws.startListener()
	go ws.Run()

	return nil
}

func (ws *Server) Clients() map[string]wow.PayloadSender {
	ret := map[string]wow.PayloadSender{}

	ws.clients.Range(func(key, value any) bool {
		// ! FIXME: babysocket clients should be re-enabled
		// v, _ := value.(*WorldSession)
		// ck, _ := key.(string)

		// ret[ck] = v

		return true
	})

	return ret
}

func (ws *Server) startListener() {
	for {
		conn, err := ws.gameListener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			log.Error().Err(err).Msg("listener error")

			continue
		}

		NewWorldSession(conn, ws)
	}
}

func (ws *Server) AddClient(gc *WorldSession) {
	ws.clients.Store(gc.ID, gc)

	count := 0

	ws.clients.Range(func(key, value any) bool {
		count++

		return true
	})

	ws.log.Debug().Int("clients", count).
		Str("acc", gc.AccountName).
		Msgf("client added to set with id: %s", gc.ID)
}

func (ws *Server) Disconnected(gc *WorldSession, reason string) {
	ws.clients.Delete(gc.ID)
}

func (ws *Server) Stats() {
	ws.log.Debug().Msgf(MemUsage())
}

func (ws *Server) Run() {
	ticker := time.NewTicker(time.Second * 20)

	defer ws.gameListener.Close()
	defer ws.log.Warn().Msg("world server stopped")

	for {
		select {
		case <-ticker.C:
			// log.Info().Msg("Garbage collector timer: unimplemented")
			// ws.Stats()
			// ws.SaveAll()
			// ! TODO: shutdown with another channel
			// case <-ws.ctx.Done():
			// 	return
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
