package world

import (
	"context"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/rs/zerolog/log"
)

// Default respawn delay in seconds if not specified by creature data.
const DefaultRespawnDelaySecs = 300 // 5 minutes

// Default corpse decay time in seconds.
const DefaultCorpseDecaySecs = 60 // 1 minute

// RespawnManager handles NPC respawn timers using per-NPC goroutines.
type RespawnManager struct {
	mu      sync.Mutex
	spawns  *SpawnManager
	server  *Server
	cancel  map[uint64]context.CancelFunc // spawnID → cancel function
	dead    map[uint64]bool               // spawnID → is dead and waiting
}

// NewRespawnManager creates a new respawn manager.
func NewRespawnManager(spawns *SpawnManager) *RespawnManager {
	return &RespawnManager{
		spawns: spawns,
		cancel: make(map[uint64]context.CancelFunc),
		dead:   make(map[uint64]bool),
	}
}

// SetServer sets the server reference for broadcasting.
func (rm *RespawnManager) SetServer(server *Server) {
	rm.server = server
}

// ScheduleRespawn starts a goroutine that waits for the respawn delay
// then recreates the NPC. Calling CancelRespawn aborts it.
func (rm *RespawnManager) ScheduleRespawn(npc *NPC) {
	rm.mu.Lock()

	// Cancel any existing respawn for this NPC
	if cancel, exists := rm.cancel[npc.SpawnID]; exists {
		cancel()
	}

	respawnDelay := time.Duration(DefaultRespawnDelaySecs+DefaultCorpseDecaySecs) * time.Second
	if npc.SpawnTimeSecs > 0 {
		respawnDelay = time.Duration(npc.SpawnTimeSecs+DefaultCorpseDecaySecs) * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	rm.cancel[npc.SpawnID] = cancel
	rm.dead[npc.SpawnID] = true

	rm.mu.Unlock()

	log.Debug().
		Uint64("spawn", npc.SpawnID).
		Str("name", npc.Name).
		Dur("delay", respawnDelay).
		Msg("NPC respawn scheduled")

	// Snapshot the NPC data we need to recreate it
	snapshot := &deadNPC{
		EntryID:      npc.EntryID,
		SpawnID:      npc.SpawnID,
		Name:         npc.Name,
		DisplayID:    npc.DisplayID,
		Faction:      npc.Faction,
		Level:        npc.Level,
		X:            npc.X,
		Y:            npc.Y,
		Z:            npc.Z,
		O:            npc.O,
		MapID:        npc.Map,
		NpcFlags:     npc.NpcFlags,
		UnitFlags:    npc.UnitFlags,
		SpawnTimeSecs: npc.SpawnTimeSecs,
	}

	go rm.respawnLoop(ctx, snapshot, respawnDelay)
}

// respawnLoop waits for the delay then recreates the NPC.
func (rm *RespawnManager) respawnLoop(ctx context.Context, dead *deadNPC, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		// Cancelled — NPC was manually respawned or server shutting down
		return
	case <-timer.C:
		// Timer expired — respawn the NPC
		rm.respawnNPC(dead)
	}
}

// respawnNPC recreates an NPC from a dead snapshot and adds it to the world.
func (rm *RespawnManager) respawnNPC(dead *deadNPC) {
	rm.mu.Lock()

	// Check if already respawned (someone else did it)
	if rm.spawns.GetNPC(dead.SpawnID) != nil {
		delete(rm.dead, dead.SpawnID)
		delete(rm.cancel, dead.SpawnID)
		rm.mu.Unlock()

		return
	}

	delete(rm.dead, dead.SpawnID)
	delete(rm.cancel, dead.SpawnID)
	rm.mu.Unlock()

	npc := &NPC{
		Object:        object.NewObject(),
		Unit:          object.NewUnit(),
		SpawnID:       dead.SpawnID,
		EntryID:       dead.EntryID,
		Name:          dead.Name,
		DisplayID:     dead.DisplayID,
		Faction:       dead.Faction,
		Level:         dead.Level,
		Health:        100,
		MaxHealth:     100,
		X:             dead.X,
		Y:             dead.Y,
		Z:             dead.Z,
		O:             dead.O,
		Map:           dead.MapID,
		NpcFlags:      dead.NpcFlags,
		UnitFlags:     dead.UnitFlags,
		SpawnTimeSecs: dead.SpawnTimeSecs,
	}
	npc.init()

	rm.spawns.SpawnNPC(npc)

	log.Debug().
		Uint64("spawn", dead.SpawnID).
		Str("name", dead.Name).
		Msg("NPC respawned")

	// Notify all players in the map
	if rm.server != nil {
		for _, gc := range rm.server.GetOnlineSessions() {
			if gc.player != nil && gc.player.IsInWorld && gc.player.Location.Map == dead.MapID {
				gc.sendCreateObjectForNPC(npc)
			}
		}
	}
}

// CancelRespawn cancels a pending respawn for a spawn ID.
func (rm *RespawnManager) CancelRespawn(spawnID uint64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if cancel, exists := rm.cancel[spawnID]; exists {
		cancel()
		delete(rm.cancel, spawnID)
		delete(rm.dead, spawnID)
	}
}

// IsDead returns true if the NPC is waiting to respawn.
func (rm *RespawnManager) IsDead(spawnID uint64) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	return rm.dead[spawnID]
}

// DeadCount returns the number of NPCs waiting to respawn.
func (rm *RespawnManager) DeadCount() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	return len(rm.dead)
}

// Shutdown cancels all pending respawns (called on server stop).
func (rm *RespawnManager) Shutdown() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for id, cancel := range rm.cancel {
		cancel()
		delete(rm.cancel, id)
	}

	rm.dead = make(map[uint64]bool)
}

// deadNPC holds the snapshot of a dead NPC needed to recreate it.
type deadNPC struct {
	EntryID       uint32
	SpawnID       uint64
	Name          string
	DisplayID     uint32
	Faction       uint32
	Level         uint8
	X, Y, Z, O   float32
	MapID         uint32
	NpcFlags      uint32
	UnitFlags     uint32
	SpawnTimeSecs uint32
}
