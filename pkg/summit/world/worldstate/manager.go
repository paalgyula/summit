package worldstate

import (
	"database/sql"
	"sync"

	"github.com/rs/zerolog/log"
)

// Manager handles world state operations and player updates.
type Manager struct {
	worldState *WorldState
	db         *sql.DB
	mutex      sync.RWMutex
}

// NewManager creates a new world state manager.
func NewManager(db *sql.DB) *Manager {
	return &Manager{
		worldState: GetWorldState(),
		db:         db,
	}
}

// SetState sets a world state value.
func (m *Manager) SetState(index uint32, value uint64) {
	m.worldState.SetState(m.db, index, value)
}

// GetState returns a world state value.
func (m *Manager) GetState(index uint32) uint64 {
	return m.worldState.GetState(index)
}

// InitWorldStates is a placeholder for sending initial world states to a player.
// In AzerothCore, this sends SMSG_INIT_WORLD_STATES with all relevant states.
func (m *Manager) InitWorldStates(playerID uint32, mapID, zoneID, areaID uint32) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Get states for this map
	states := m.worldState.GetStatesForMap(mapID)

	// Build and send SMSG_INIT_WORLD_STATES packet
	// This would be implemented in the packet system
	log.Debug().
		Uint32("player", playerID).
		Uint32("map", mapID).
		Uint32("zone", zoneID).
		Uint32("area", areaID).
		Int("states", len(states)).
		Msg("Would send InitWorldStates")
}

// SendWorldStateUpdate sends a world state update to a player.
func (m *Manager) SendWorldStateUpdate(playerID uint32, variableID, value int32) {
	// Send SMSG_UPDATE_WORLD_STATE packet
	log.Debug().
		Uint32("player", playerID).
		Int32("variable", variableID).
		Int32("value", value).
		Msg("Would send UpdateWorldState")
}

// BroadcastWorldState sends a world state update to all players.
func (m *Manager) BroadcastWorldState(variableID, value int32) {
	// This would iterate over all online players and send the update
	log.Debug().
		Int32("variable", variableID).
		Int32("value", value).
		Msg("Would broadcast WorldState")
}

// common world state IDs (from AzerothCore)
const (
	// Time states
	WORLD_STATE_CUSTOM_WEEKLY_QUEST_RESET_TIME                 = 36001
	WORLD_STATE_CUSTOM_MONTHLY_QUEST_RESET_TIME                = 36002
	WORLD_STATE_CUSTOM_BG_DAILY_RESET_TIME                     = 36003
	WORLD_STATE_CUSTOM_GUILD_DAILY_RESET_TIME                  = 36004
	WORLD_STATE_CUSTOM_DAILY_CALENDAR_DELETION_OLD_EVENTS_TIME = 36005

	// Scourge Invasion states
	WORLD_STATE_SCOURGE_INVASION_AZSHARA                        = 3960
	WORLD_STATE_SCOURGE_INVASION_BLASTED_LANDS                  = 3961
	WORLD_STATE_SCOURGE_INVASION_BURNING_STEPPES                = 3962
	WORLD_STATE_SCOURGE_INVASION_EASTERN_PLAGUELANDS            = 3963
	WORLD_STATE_SCOURGE_INVASION_TANARIS                        = 3964
	WORLD_STATE_SCOURGE_INVASION_WINTERSPRING                   = 3965
	WORLD_STATE_SCOURGE_INVASION_VICTORIES                      = 3966
	WORLD_STATE_SCOURGE_INVASION_NECROPOLIS_AZSHARA             = 3967
	WORLD_STATE_SCOURGE_INVASION_NECROPOLIS_BLASTED_LANDS       = 3968
	WORLD_STATE_SCOURGE_INVASION_NECROPOLIS_BURNING_STEPPES     = 3969
	WORLD_STATE_SCOURGE_INVASION_NECROPOLIS_EASTERN_PLAGUELANDS = 3970
	WORLD_STATE_SCOURGE_INVASION_NECROPOLIS_TANARIS             = 3971
	WORLD_STATE_SCOURGE_INVASION_NECROPOLIS_WINTERSPRING        = 3972

	// World PvP states
	WORLD_STATE_WSG_ENABLED        = 4248
	WORLD_STATE_WSG_ALLIANCE_SCORE = 4249
	WORLD_STATE_WSG_HORDE_SCORE    = 4250
	WORLD_STATE_WSG_MAX_SCORE      = 4251

	// Wintergrasp states
	WORLD_STATE_WINTERGRASP_ACTIVE       = 31366
	WORLD_STATE_WINTERGRASP_FOR_DEFENSE  = 31372
	WORLD_STATE_WINTERGRASP_FOR_ATTACKER = 31373
	WORLD_STATE_WINTERGRASP_TIME         = 31376
	WORLD_STATE_WINTERGRASP_VICTORY_A    = 31384
	WORLD_STATE_WINTERGRASP_VICTORY_H    = 31385
)
