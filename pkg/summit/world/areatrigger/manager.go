package areatrigger

import (
	"database/sql"
	"sync"

	"github.com/rs/zerolog/log"
)

// Manager manages all areatriggers and their scripts.
type Manager struct {
	// All areatriggers indexed by entry
	triggers map[uint32]*AreaTrigger

	// Teleport destinations indexed by entry
	teleports map[uint32]*AreaTriggerTeleport

	// Scripts indexed by trigger entry
	scripts map[uint32]AreaTriggerScript

	// OnlyOnce triggers that have been activated
	// Key: instanceKey (mapID:instanceID), Value: set of trigger entries
	activatedTriggers map[string]map[uint32]bool

	mutex sync.RWMutex
}

// NewManager creates a new areatrigger manager.
func NewManager() *Manager {
	return &Manager{
		triggers:          make(map[uint32]*AreaTrigger),
		teleports:         make(map[uint32]*AreaTriggerTeleport),
		scripts:           make(map[uint32]AreaTriggerScript),
		activatedTriggers: make(map[string]map[uint32]bool),
	}
}

// LoadFromDB loads areatriggers from the database.
func (m *Manager) LoadFromDB(db *sql.DB) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Load areatriggers
	rows, err := db.Query("SELECT entry, map, x, y, z, radius, length, width, height, orientation FROM areatrigger")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load areatriggers, using empty table")
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var at AreaTrigger
		if err := rows.Scan(&at.Entry, &at.MapID, &at.X, &at.Y, &at.Z,
			&at.Radius, &at.Length, &at.Width, &at.Height, &at.Orientation); err != nil {
			log.Error().Err(err).Msg("Failed to scan areatrigger")
			continue
		}

		m.triggers[at.Entry] = &at
		count++
	}

	log.Info().Int("count", count).Msg("Loaded areatriggers")

	// Load areatrigger teleports
	teleRows, err := db.Query("SELECT ID, target_map, target_position_x, target_position_y, target_position_z, target_orientation FROM areatrigger_teleport")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load areatrigger teleports")
		return nil
	}
	defer teleRows.Close()

	teleCount := 0
	for teleRows.Next() {
		var id uint32
		var at AreaTriggerTeleport
		if err := teleRows.Scan(&id, &at.TargetMapID, &at.TargetX, &at.TargetY, &at.TargetZ, &at.TargetO); err != nil {
			log.Error().Err(err).Msg("Failed to scan areatrigger teleport")
			continue
		}

		m.teleports[id] = &at
		teleCount++
	}

	log.Info().Int("count", teleCount).Msg("Loaded areatrigger teleports")

	return nil
}

// GetTrigger returns an areatrigger by entry.
func (m *Manager) GetTrigger(entry uint32) *AreaTrigger {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.triggers[entry]
}

// GetTeleport returns a teleport destination by trigger entry.
func (m *Manager) GetTeleport(entry uint32) *AreaTriggerTeleport {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.teleports[entry]
}

// GetTriggersForMap returns all areatriggers for a specific map.
func (m *Manager) GetTriggersForMap(mapID uint32) []*AreaTrigger {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*AreaTrigger
	for _, at := range m.triggers {
		if at.MapID == mapID {
			result = append(result, at)
		}
	}

	return result
}

// RegisterScript registers a script for an areatrigger.
func (m *Manager) RegisterScript(entry uint32, script AreaTriggerScript) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.scripts[entry] = script
}

// CheckTrigger checks if a player has entered an areatrigger and executes scripts.
func (m *Manager) CheckTrigger(playerID uint32, entry uint32, x, y, z float32) bool {
	m.mutex.RLock()
	at := m.triggers[entry]
	script := m.scripts[entry]
	m.mutex.RUnlock()

	if at == nil {
		return false
	}

	// Check if player is inside the trigger
	if !at.IsInside(x, y, z) {
		return false
	}

	// Execute script if registered
	if script != nil {
		return script.OnTrigger(playerID, at)
	}

	return false
}

// IsTriggerDone checks if a once-only trigger has been activated.
func (m *Manager) IsTriggerDone(instanceKey string, entry uint32) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if triggers, ok := m.activatedTriggers[instanceKey]; ok {
		return triggers[entry]
	}

	return false
}

// MarkTriggerDone marks a once-only trigger as activated.
func (m *Manager) MarkTriggerDone(instanceKey string, entry uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.activatedTriggers[instanceKey] == nil {
		m.activatedTriggers[instanceKey] = make(map[uint32]bool)
	}

	m.activatedTriggers[instanceKey][entry] = true
}

// ResetTriggerDone resets a once-only trigger.
func (m *Manager) ResetTriggerDone(instanceKey string, entry uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if triggers, ok := m.activatedTriggers[instanceKey]; ok {
		delete(triggers, entry)
	}
}

// MakeInstanceKey creates an instance key from map and instance IDs.
func MakeInstanceKey(mapID, instanceID uint32) string {
	return string(rune(mapID)) + ":" + string(rune(instanceID))
}

// AddTrigger adds a new areatrigger (for runtime creation).
func (m *Manager) AddTrigger(at *AreaTrigger) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.triggers[at.Entry] = at
}

// RemoveTrigger removes an areatrigger.
func (m *Manager) RemoveTrigger(entry uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.triggers, entry)
}
