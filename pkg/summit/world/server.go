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
	"github.com/paalgyula/summit/pkg/summit/world/channel"
	"github.com/paalgyula/summit/pkg/summit/world/lfg"
	mapmanager "github.com/paalgyula/summit/pkg/summit/world/map"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/summit/world/quest"
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

	// Quest manager (quest templates, relations, completion)
	questMgr *quest.Manager

	// Respawn manager (handles NPC respawn timers)
	respawnMgr *RespawnManager

	// Map manager
	mapManager *mapmanager.MapManager

	// AreaTrigger manager
	areaTriggerMgr *areatrigger.Manager

	// WorldState manager
	worldStateManager *worldstate.Manager

	// Group manager
	groupMu sync.RWMutex
	groups  map[uint32]*Group

	// Channel managers (one per faction + neutral)
	channelAlliance *channel.Manager
	channelHorde    *channel.Manager
	channelNeutral  *channel.Manager

	// LFG manager
	lfgMgr *lfg.Manager

	// Chat configuration
	chatConfig ChatConfig
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
	worldServer.mapManager.SetGameObjectDynamicFlagsFunc(worldServer.gameObjectDynamicFlags)
	worldServer.groups = make(map[uint32]*Group)
	worldServer.channelAlliance = channel.NewManager(1)
	worldServer.channelHorde = channel.NewManager(2)
	worldServer.channelNeutral = channel.NewManager(0)
	worldServer.chatConfig = DefaultChatConfig()

	// Initialize LFG with sample dungeons (real data should come from DBC)
	worldServer.lfgMgr = lfg.NewManager([]*lfg.Dungeon{
		{ID: 1, Name: "Ragefire Chasm", MapID: 389, MinLevel: 8, MaxLevel: 13},
		{ID: 2, Name: "Wailing Caverns", MapID: 43, MinLevel: 10, MaxLevel: 18},
		{ID: 3, Name: "The Deadmines", MapID: 36, MinLevel: 10, MaxLevel: 18},
		{ID: 4, Name: "Shadowfang Keep", MapID: 33, MinLevel: 11, MaxLevel: 20},
		{ID: 5, Name: "Stormwind Stockade", MapID: 34, MinLevel: 15, MaxLevel: 22},
		{ID: 6, Name: "Gnomeregan", MapID: 90, MinLevel: 17, MaxLevel: 24},
		{ID: 7, Name: "Razorfen Kraul", MapID: 17, MinLevel: 23, MaxLevel: 30},
		{ID: 8, Name: "The Scarlet Monastery", MapID: 189, MinLevel: 26, MaxLevel: 36},
		{ID: 9, Name: "Uldaman", MapID: 70, MinLevel: 31, MaxLevel: 40},
		{ID: 10, Name: "Zul'Farrak", MapID: 117, MinLevel: 35, MaxLevel: 43},
		{ID: 11, Name: "Maraudon", MapID: 138, MinLevel: 36, MaxLevel: 44},
		{ID: 12, Name: "Temple of Atal'Hakkar", MapID: 109, MinLevel: 41, MaxLevel: 50},
		{ID: 13, Name: "Blackrock Depths", MapID: 158, MinLevel: 48, MaxLevel: 56},
		{ID: 14, Name: "Lower Blackrock Spire", MapID: 229, MinLevel: 53, MaxLevel: 60},
		{ID: 15, Name: "Upper Blackrock Spire", MapID: 229, MinLevel: 55, MaxLevel: 60},
		{ID: 16, Name: "Dire Maul", MapID: 429, MinLevel: 54, MaxLevel: 60},
		{ID: 17, Name: "Stratholme", MapID: 329, MinLevel: 55, MaxLevel: 60},
		{ID: 18, Name: "Scholomance", MapID: 289, MinLevel: 55, MaxLevel: 60},
		{ID: 19, Name: "Ragefire Chasm (Heroic)", MapID: 389, Difficulty: 1, MinLevel: 70, MaxLevel: 80},
		{ID: 20, Name: "Deadmines (Heroic)", MapID: 36, Difficulty: 1, MinLevel: 70, MaxLevel: 80},
	})

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

	// Full item templates (names, stats, prices) come from the world store;
	// summit.dat only carries what Item.dbc knows.
	if worldStore != nil {
		if items, err := worldStore.GetItemTemplates(); err != nil {
			ws.log.Warn().Err(err).Msg("cannot load item templates from the world store")
		} else {
			basedata.GetInstance().MergeItemTemplates(items)
			ws.log.Info().Int("items", len(items)).Msg("loaded item templates from store")
		}
	}

	// Initialize quest manager with world data
	ws.questMgr = quest.NewManager(worldStore)

	// Load creature spawns from database (falls back to hardcoded if no world store)
	ws.spawns = NewSpawnManagerFromDB(worldStore)

	// Register all spawned NPCs into their respective maps for grid visibility
	ws.registerSpawnsWithMaps()

	// Initialize respawn manager
	ws.respawnMgr = NewRespawnManager(ws.spawns)
	ws.respawnMgr.SetServer(ws)

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
	// Save and clean up the player before removing
	if gc.player != nil && gc.player.IsInWorld {
		// Persist the character to the database
		if err := ws.charStore.UpdateCharacter(gc.player); err != nil {
			gc.log.Error().Err(err).Str("name", gc.player.Name).
				Msg("failed to save character on disconnect")
		} else {
			gc.log.Info().Str("name", gc.player.Name).Msg("character saved on disconnect")
		}

		// Send DestroyObject to other players before removing
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
	gc.log.Info().Str("reason", reason).Msg("client disconnected")
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

	// Cancel all pending respawn goroutines on shutdown
	if ws.respawnMgr != nil {
		defer ws.respawnMgr.Shutdown()
	}

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

	// Update spawned NPCs (aggro scans, chase, and combat ticks)
	ws.updateNPCs(now)

	// Update spawned game objects (loot/respawn state machine)
	ws.updateGameObjects()

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

// updateNPCs processes AI, aggro scans, and combat ticks for all spawned creatures.
func (ws *Server) updateNPCs(now time.Time) {
	if ws.spawns == nil {
		return
	}

	// Snapshot online, in-world, alive players
	var players []*player.Player
	ws.clients.Range(func(_, value any) bool {
		if gc, ok := value.(*WorldSession); ok && gc.player != nil && gc.player.IsInWorld && gc.player.IsAlive() {
			players = append(players, gc.player)
		}
		return true
	})

	npcs := ws.spawns.GetNPCs()
	for _, npc := range npcs {
		if !npc.IsAlive() {
			continue
		}

		// Interpolate active spline movement
		npc.UpdatePositionFromSpline(now)

		if !npc.InCombat {
			// Proximity aggro scan
			for _, p := range players {
				if p.Location.Map != npc.Map {
					continue
				}
				dx := p.Location.X - npc.X
				dy := p.Location.Y - npc.Y
				dz := p.Location.Z - npc.Z
				distSq := dx*dx + dy*dy + dz*dz

				aggroRadius := npc.AggroRadius
				if aggroRadius <= 0 {
					aggroRadius = 20.0
				}
				if distSq <= aggroRadius*aggroRadius {
					if IsHostileToCheck(npc, p) {
						npc.Attack(p, true)
						npc.AddThreat(p, 100.0)
						break
					}
				}
			}

			// Idle wander if not in combat
			if !npc.InCombat {
				// Create a send function that only sends to players who can see this NPC
				sendToVisible := func(pkt *wow.Packet) {
					if ws.mapManager != nil {
						m := ws.mapManager.FindBaseMap(npc.Map)
						if m != nil {
							m.SendToNPCVisiblePlayers(npc.GUID(), pkt)
							return
						}
					}
					// Fallback to broadcast if map not found
					ws.BroadcastPacket(pkt)
				}

				// Waypoint movement takes priority over random wander
				if npc.MovementType == store.MotionTypeWaypoint && npc.WaypointPath != nil {
					ProcessNPCWaypoint(npc, now, sendToVisible)
				} else if npc.MovementType == store.MotionTypeRandom {
					ProcessNPCWander(npc, now, sendToVisible)
				}
			}
		} else {
			// In combat: process chase, attack swings, and combat exit timer
			sendToVisible := func(pkt *wow.Packet) {
				if ws.mapManager != nil {
					m := ws.mapManager.FindBaseMap(npc.Map)
					if m != nil {
						m.SendToNPCVisiblePlayers(npc.GUID(), pkt)
						return
					}
				}
				ws.BroadcastPacket(pkt)
			}
			ProcessNPCChase(npc, now, sendToVisible)
			ProcessNPCCombatTick(npc, now, sendToVisible)
			ProcessCombatTimer(npc, now)
		}
	}
}

// updateGameObjects advances the game object state machine for all spawned
// game objects. Runs on the 50ms world tick.
func (ws *Server) updateGameObjects() {
	if ws.gameObjects == nil {
		return
	}

	ws.gameObjects.Update(50, GameObjectUseContext{Server: ws})
}

// gameObjectDynamicFlags computes GAMEOBJECT_DYNAMIC from a viewer's
// perspective: a game object that satisfies an incomplete kill/use-objective of
// one of the player's quests sparkles and becomes interactable.
// Mirrors GameObject::ActivateToQuest / BuildValuesUpdate's dynamic flags.
func (ws *Server) gameObjectDynamicFlags(gobj interface{}, p *player.Player) uint16 {
	g, ok := gobj.(*GameObject)
	if !ok || g == nil || p == nil || ws.questMgr == nil {
		return 0
	}

	quests, ok := p.QuestStatus.(map[uint32]*quest.QuestStatusData)
	if !ok {
		return 0
	}

	for questID, status := range quests {
		if status == nil || status.Status != quest.QuestStatusIncomplete {
			continue
		}

		q := ws.questMgr.GetQuest(questID)
		if q == nil {
			continue
		}

		for i := 0; i < 4; i++ {
			if q.RequiredNpcOrGo[i] < 0 && uint32(-q.RequiredNpcOrGo[i]) == g.Entry {
				return uint16(basedata.GODynFlagActivate | basedata.GODynFlagSparkle)
			}
		}
	}

	return 0
}

// BroadcastPacket sends a packet to all sessions that have a player actively
// in the world (i.e. past auth handshake and map placement).
// Pre-auth sessions must never receive world packets — doing so injects bytes
// before the header cipher is initialised and desyncs the WoW framing.
func (ws *Server) BroadcastPacket(pkt *wow.Packet) {
	if pkt == nil {
		return
	}
	ws.clients.Range(func(_, value any) bool {
		gc, ok := value.(*WorldSession)
		if !ok {
			return true
		}
		// Skip sessions that have not yet authenticated or are not in the world.
		if gc.player == nil || !gc.player.IsInWorld {
			return true
		}
		gc.Send(pkt)
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

		if gc.player != nil && gc.player.IsInWorld {
			if err := ws.charStore.UpdateCharacter(gc.player); err != nil {
				gc.log.Error().Err(err).Str("name", gc.player.Name).
					Msg("periodic save failed")
			} else {
				gc.log.Debug().Str("name", gc.player.Name).Msg("periodic save")
			}
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

// GetGroup returns the group with the given ID, or nil.
func (ws *Server) GetGroup(id uint32) *Group {
	ws.groupMu.RLock()
	defer ws.groupMu.RUnlock()

	return ws.groups[id]
}

// SetGroup stores a group in the server's group map.
func (ws *Server) SetGroup(g *Group) {
	ws.groupMu.Lock()
	defer ws.groupMu.Unlock()

	ws.groups[g.ID] = g
}

// RemoveGroup removes a group from the server.
func (ws *Server) RemoveGroup(id uint32) {
	ws.groupMu.Lock()
	defer ws.groupMu.Unlock()

	delete(ws.groups, id)
}

// SessionByGUID returns the WorldSession whose player has the given GUID, or nil.
func (ws *Server) SessionByGUID(guid wow.GUID) *WorldSession {
	var result *WorldSession

	ws.clients.Range(func(_, value any) bool {
		gc, ok := value.(*WorldSession)
		if ok && gc.player != nil && gc.player.GUID() == guid {
			result = gc

			return false
		}

		return true
	})

	return result
}

// GetChannelManager returns the channel manager for the given team.
func (ws *Server) GetChannelManager(team int) *channel.Manager {
	switch team {
	case 1:
		return ws.channelAlliance
	case 2:
		return ws.channelHorde
	default:
		return ws.channelNeutral
	}
}

// GetLfgManager returns the LFG manager.
func (ws *Server) GetLfgManager() *lfg.Manager {
	return ws.lfgMgr
}

// GetChatConfig returns the chat configuration.
func (ws *Server) GetChatConfig() ChatConfig {
	return ws.chatConfig
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

// registerSpawnsWithMaps registers all spawned NPCs and game objects into
// their respective Map instances so the grid-based visibility system can
// find them. This must be called after SpawnManager is loaded.
func (ws *Server) registerSpawnsWithMaps() {
	if ws.mapManager == nil {
		return
	}

	// Register NPCs into their maps
	if ws.spawns != nil {
		npcsByMap := make(map[uint32][]*NPC)
		for _, npc := range ws.spawns.GetNPCs() {
			npcsByMap[npc.Map] = append(npcsByMap[npc.Map], npc)
		}

		for mapID, npcs := range npcsByMap {
			m := ws.mapManager.CreateBaseMap(mapID)
			for _, npc := range npcs {
				m.AddNPC(npc)
			}
		}

		ws.log.Info().
			Int("totalNPCs", ws.spawns.Count()).
			Msg("registered NPCs into maps")
	}

	// Register game objects into their maps
	if ws.gameObjects != nil {
		gobsByMap := make(map[uint32][]*GameObject)
		for _, gobj := range ws.gameObjects.GetObjects() {
			gobsByMap[gobj.Map] = append(gobsByMap[gobj.Map], gobj)
		}

		for mapID, gobs := range gobsByMap {
			m := ws.mapManager.CreateBaseMap(mapID)
			for _, gobj := range gobs {
				m.AddGameObject(gobj)
			}
		}

		ws.log.Info().
			Int("totalGameObjects", len(ws.gameObjects.GetObjects())).
			Msg("registered game objects into maps")
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
