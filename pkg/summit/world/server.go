package world

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world/areatrigger"
	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	mapmanager "github.com/paalgyula/summit/pkg/summit/world/map"
	"github.com/paalgyula/summit/pkg/summit/world/worldstate"
	"github.com/paalgyula/summit/pkg/summit/world/wsconn"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	clients sync.Map

	gameListener net.Listener
	echoServer   *echo.Echo
	log          zerolog.Logger

	// Database access
	accountStore store.AccountRepo
	charStore    store.CharacterRepo
	worldStore   store.WorldRepo

	bs *babysocket.Server

	// Management client for auth. Can be direct, or gRPC based.
	authManagement auth.ManagementService

	// Realm-list identity reported to the auth server (nil: not advertised)
	realm     *RealmIdentity
	realmLock uint8
	done      chan struct{}

	baseData *basedata.Store

	// NPC spawn manager
	spawns *SpawnManager

	// Game object manager
	gameObjects *GameObjectManager

	// Spell data manager (DBC-loaded spell data)
	spellMgr *SpellMgr

	// Map manager
	mapManager *mapmanager.MapManager

	// AreaTrigger manager
	areaTriggerMgr *areatrigger.Manager

	// WorldState manager
	worldStateManager *worldstate.Manager
}

func NewServer(opts ...ServerOption) (*Server, error) {
	worldServer := new(Server)

	worldServer.log = log.With().
		Str("service", "world").
		Caller().Logger()
	worldServer.clients = sync.Map{}
	worldServer.done = make(chan struct{})
	worldServer.spawns = NewSpawnManager()
	worldServer.gameObjects = NewGameObjectManager()
	worldServer.spellMgr = NewSpellMgr()

	// Initialize map management system
	worldServer.mapManager = mapmanager.GetMapManager()

	// Initialize areatrigger system
	worldServer.areaTriggerMgr = areatrigger.NewManager()
	worldServer.mapManager.SetAreaTriggerManager(worldServer.areaTriggerMgr)

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
	if accRepo, ok := charStore.(store.AccountRepo); ok {
		ws.accountStore = accRepo
	}

	// Initialize worldstate manager with database
	ws.worldStateManager = worldstate.NewManager(nil) // TODO: pass actual DB

	go ws.startListener()
	go ws.Run()
	go ws.reportRealmStatus()

	return nil
}

// StartWebSocketServer starts a dedicated Echo WebSocket server for realm/world connections.
func (ws *Server) StartWebSocketServer(listenAddr string) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{"*"},
	}))

	h := wsconn.NewHandler(ws)
	e.GET(wsconn.RealmPath, h.HandleWS)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	ws.echoServer = e
	ws.log.Info().Str("listen", listenAddr).Msg("realm websocket server starting")

	go func() {
		if err := e.Start(listenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ws.log.Error().Err(err).Msg("realm websocket server failed")
		}
	}()

	return nil
}

// NewWorldSessionConn implements wsconn.SessionFactory: a browser connection
// gets the same session, handshake and encryption as an accepted TCP client.
func (ws *Server) NewWorldSessionConn(conn net.Conn) {
	NewWorldSession(conn, ws)
}

func (ws *Server) startListener() {
	for {
		conn, err := ws.gameListener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			log.Error().Err(err).Msg("tcp listener error")

			continue
		}

		NewWorldSession(conn, ws)
	}
}

func (ws *Server) Clients() map[string]wow.PayloadSender {
	ret := map[string]wow.PayloadSender{}

	ws.clients.Range(func(key, value any) bool {
		v, _ := value.(*WorldSession)
		ck, _ := key.(string)

		ret[ck] = v

		return true
	})

	return ret
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
	// Send DestroyObject to other players before removing
	if gc.player != nil && gc.player.IsInWorld {
		for _, other := range ws.GetOtherSessions(gc) {
			if other.player != nil && other.player.IsInWorld {
				other.sendDestroyObject(gc.player.GUID())
			}
		}

		// Remove player from map
		if ws.mapManager != nil {
			m := ws.mapManager.FindBaseMap(gc.player.Location.Map)
			if m != nil {
				m.RemovePlayer(gc.player.ID)
			}
		}
	}

	ws.clients.Delete(gc.ID)
}

// GetOnlineSessions returns all active sessions.
func (ws *Server) GetOnlineSessions() []*WorldSession {
	var sessions []*WorldSession

	ws.clients.Range(func(_, value any) bool {
		gc, ok := value.(*WorldSession)
		if ok {
			sessions = append(sessions, gc)
		}

		return true
	})

	return sessions
}

// GetOtherSessions returns all sessions except the given one.
func (ws *Server) GetOtherSessions(exclude *WorldSession) []*WorldSession {
	var sessions []*WorldSession

	ws.clients.Range(func(_, value any) bool {
		gc, ok := value.(*WorldSession)
		if ok && gc.ID != exclude.ID {
			sessions = append(sessions, gc)
		}

		return true
	})

	return sessions
}

func (ws *Server) Stats() {
	ws.log.Debug().Msg(MemUsage())
}

func (ws *Server) Run() {
	// World update tick: 50ms = 20 updates per second (matches C++ world update rate)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	defer ws.gameListener.Close()
	defer ws.log.Warn().Msg("world server stopped")

	lastSave := time.Now()
	saveInterval := 5 * time.Minute

	for {
		select {
		case now := <-ticker.C:
			ws.update(now)

			// Periodic save
			if now.Sub(lastSave) >= saveInterval {
				ws.saveAll()
				lastSave = now
			}
		}
	}
}

// update runs one world tick. It processes all connected sessions.
func (ws *Server) update(now time.Time) {
	// Update all maps
	if ws.mapManager != nil {
		ws.mapManager.Update(50) // 50ms tick
	}

	ws.clients.Range(func(key, value any) bool {
		gc, ok := value.(*WorldSession)
		if !ok {
			return true
		}

		if gc.player == nil {
			return true
		}

		// Periodic player save
		gc.updatePeriodic(now)

		return true
	})
}

// saveAll forces a save of all online players.
func (ws *Server) saveAll() {
	ws.clients.Range(func(key, value any) bool {
		gc, ok := value.(*WorldSession)
		if !ok {
			return true
		}

		if gc.player != nil {
			gc.log.Debug().Str("name", gc.player.Name).Msg("periodic save")
		}

		return true
	})
}

// GetOnlinePlayerCount returns the number of connected players.
func (ws *Server) GetOnlinePlayerCount() int {
	count := 0

	ws.clients.Range(func(key, value any) bool {
		gc, ok := value.(*WorldSession)
		if ok && gc.player != nil {
			count++
		}

		return true
	})

	return count
}

// GetAllSessions returns a snapshot of all active sessions.
func (ws *Server) GetAllSessions() []*WorldSession {
	var sessions []*WorldSession

	ws.clients.Range(func(key, value any) bool {
		gc, ok := value.(*WorldSession)
		if ok {
			sessions = append(sessions, gc)
		}

		return true
	})

	return sessions
}

// GetMapManager returns the map manager.
func (ws *Server) GetMapManager() *mapmanager.MapManager {
	return ws.mapManager
}

// GetAreaTriggerManager returns the areatrigger manager.
func (ws *Server) GetAreaTriggerManager() *areatrigger.Manager {
	return ws.areaTriggerMgr
}

// GetWorldStateManager returns the worldstate manager.
func (ws *Server) GetWorldStateManager() *worldstate.Manager {
	return ws.worldStateManager
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
