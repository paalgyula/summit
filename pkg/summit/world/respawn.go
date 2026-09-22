package world

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Default respawn delay in seconds if not specified by creature data.
const DefaultRespawnDelaySecs = 300 // 5 minutes

// Default corpse decay time in seconds.
const DefaultCorpseDecaySecs = 60 // 1 minute

// RespawnManager drives a killed NPC through corpse decay and respawn using
// one goroutine per dead NPC. The NPC object is kept and reused: the corpse
// stays in the map until it decays, then leaves the map, and after the
// respawn delay it is reset and added back, so the grid visibility pass
// sends the create packets to whoever is near.
type RespawnManager struct {
	mu     sync.Mutex
	spawns *SpawnManager
	server *Server
	cancel map[uint64]context.CancelFunc // spawnID → cancel function
	dead   map[uint64]bool               // spawnID → is dead and waiting
}

// NewRespawnManager creates a new respawn manager.
func NewRespawnManager(spawns *SpawnManager) *RespawnManager {
	return &RespawnManager{
		spawns: spawns,
		cancel: make(map[uint64]context.CancelFunc),
		dead:   make(map[uint64]bool),
	}
}

// SetServer sets the server reference used to reach the maps.
func (rm *RespawnManager) SetServer(server *Server) {
	rm.server = server
}

// ScheduleRespawn starts the corpse decay → respawn cycle of a killed NPC.
// Calling CancelRespawn aborts it.
func (rm *RespawnManager) ScheduleRespawn(npc *NPC) {
	rm.mu.Lock()

	// Cancel any existing respawn for this NPC
	if cancel, exists := rm.cancel[npc.SpawnID]; exists {
		cancel()
	}

	respawnDelay := time.Duration(DefaultRespawnDelaySecs) * time.Second
	if npc.SpawnTimeSecs > 0 {
		respawnDelay = time.Duration(npc.SpawnTimeSecs) * time.Second
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

	go rm.respawnLoop(ctx, npc, DefaultCorpseDecaySecs*time.Second, respawnDelay)
}

// respawnLoop waits out the corpse, despawns it, waits the respawn delay and
// brings the NPC back.
func (rm *RespawnManager) respawnLoop(ctx context.Context, npc *NPC, corpseDecay, respawnDelay time.Duration) {
	if !sleepCtx(ctx, corpseDecay) {
		return
	}

	rm.despawnNPC(npc)

	if !sleepCtx(ctx, respawnDelay) {
		return
	}

	rm.respawnNPC(npc)
}

// sleepCtx waits for d; false when ctx was cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// despawnNPC removes the corpse from the world.
func (rm *RespawnManager) despawnNPC(npc *NPC) {
	if rm.server != nil && rm.server.mapManager != nil {
		rm.server.mapManager.CreateBaseMap(npc.Map).RemoveNPC(npc.GUID())
	}

	rm.spawns.RemoveNPC(npc.SpawnID)

	log.Debug().
		Uint64("spawn", npc.SpawnID).
		Str("name", npc.Name).
		Msg("NPC corpse despawned")
}

// respawnNPC resets the NPC and puts it back into the world.
func (rm *RespawnManager) respawnNPC(npc *NPC) {
	rm.mu.Lock()
	delete(rm.dead, npc.SpawnID)
	delete(rm.cancel, npc.SpawnID)
	rm.mu.Unlock()

	npc.Reset()
	rm.spawns.SpawnNPC(npc)

	if rm.server != nil && rm.server.mapManager != nil {
		rm.server.mapManager.CreateBaseMap(npc.Map).AddNPC(npc)
	}

	log.Debug().
		Uint64("spawn", npc.SpawnID).
		Str("name", npc.Name).
		Msg("NPC respawned")
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
