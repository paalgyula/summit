package worldstate

import (
	"database/sql"
	"sync"

	"github.com/rs/zerolog/log"
)

// WorldState manages global world state values.
type WorldState struct {
	states map[uint32]uint64
	mutex  sync.RWMutex
}

var (
	globalWorldState *WorldState
	worldStateOnce   sync.Once
)

// GetWorldState returns the global WorldState instance.
func GetWorldState() *WorldState {
	worldStateOnce.Do(func() {
		globalWorldState = &WorldState{
			states: make(map[uint32]uint64),
		}
	})

	return globalWorldState
}

// LoadFromDB loads world states from the database.
func (ws *WorldState) LoadFromDB(db *sql.DB) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	rows, err := db.Query("SELECT entry, value FROM worldstates")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load world states, using empty table")
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var entry uint32
		var value uint64
		if err := rows.Scan(&entry, &value); err != nil {
			log.Error().Err(err).Msg("Failed to scan world state")
			continue
		}

		ws.states[entry] = value
		count++
	}

	log.Info().Int("count", count).Msg("Loaded world states")

	return nil
}

// SetState sets a world state value and persists to database.
func (ws *WorldState) SetState(db *sql.DB, index uint32, value uint64) {
	ws.mutex.Lock()
	ws.states[index] = value
	ws.mutex.Unlock()

	// Persist to database
	if db != nil {
		// Try update first
		result, err := db.Exec("UPDATE worldstates SET value = ? WHERE entry = ?", value, index)
		if err != nil {
			log.Error().Err(err).Uint32("entry", index).Msg("Failed to update world state")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// Insert if not exists
			_, err = db.Exec("INSERT INTO worldstates (entry, value) VALUES (?, ?)", index, value)
			if err != nil {
				log.Error().Err(err).Uint32("entry", index).Msg("Failed to insert world state")
			}
		}
	}
}

// GetState returns a world state value.
func (ws *WorldState) GetState(index uint32) uint64 {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	return ws.states[index]
}

// GetAllStates returns all world states.
func (ws *WorldState) GetAllStates() map[uint32]uint64 {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	result := make(map[uint32]uint64, len(ws.states))
	for k, v := range ws.states {
		result[k] = v
	}

	return result
}

// DeleteState removes a world state.
func (ws *WorldState) DeleteState(db *sql.DB, index uint32) {
	ws.mutex.Lock()
	delete(ws.states, index)
	ws.mutex.Unlock()

	if db != nil {
		_, err := db.Exec("DELETE FROM worldstates WHERE entry = ?", index)
		if err != nil {
			log.Error().Err(err).Uint32("entry", index).Msg("Failed to delete world state")
		}
	}
}

// GetStatesForMap returns world states relevant to a specific map.
// This is a simplified version - actual implementation would filter based on map/zone.
func (ws *WorldState) GetStatesForMap(mapID uint32) map[uint32]uint64 {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	// For now, return all states
	// TODO: Filter based on map/zone requirements
	result := make(map[uint32]uint64)
	for k, v := range ws.states {
		result[k] = v
	}

	return result
}
