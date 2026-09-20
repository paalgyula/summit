package mapmanager

import (
	"sync"

	"github.com/paalgyula/summit/pkg/summit/world/areatrigger"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/rs/zerolog"
)

// Map represents a game map (world, dungeon, battleground, etc.).
type Map struct {
	ID         uint32
	InstanceID uint32
	SpawnMode  uint8

	Name  string
	Entry *MapEntry

	players map[uint32]*player.Player
	npcs    map[uint32]interface{} // NPC interface
	objects map[uint32]interface{} // GameObject interface

	// AreaTriggers on this map
	areaTriggers []*areatrigger.AreaTrigger

	mutex sync.RWMutex
	log   zerolog.Logger
}

// MapEntry holds basic map data from DBC.
type MapEntry struct {
	ID           uint32
	InstanceType uint32
	Flags        uint32
	Name         string
	LinkedZone   uint32
	MultimapID   uint32
	EntranceMap  uint32
	EntranceX    float32
	EntranceY    float32
	ExpansionID  uint32
	MaxPlayers   uint32
}

// NewMap creates a new map instance.
func NewMap(id, instanceID uint32, entry *MapEntry) *Map {
	return &Map{
		ID:         id,
		InstanceID: instanceID,
		Entry:      entry,
		players:    make(map[uint32]*player.Player),
		npcs:       make(map[uint32]interface{}),
		objects:    make(map[uint32]interface{}),
		log:        zerolog.Logger{},
	}
}

// AddPlayer adds a player to the map.
func (m *Map) AddPlayer(p *player.Player) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.players[p.ID] = p
	p.IsInWorld = true
}

// RemovePlayer removes a player from the map.
func (m *Map) RemovePlayer(guid uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if p, ok := m.players[guid]; ok {
		p.IsInWorld = false
		delete(m.players, guid)
	}
}

// GetPlayer returns a player by GUID.
func (m *Map) GetPlayer(guid uint32) *player.Player {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.players[guid]
}

// GetPlayers returns all players on the map.
func (m *Map) GetPlayers() []*player.Player {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]*player.Player, 0, len(m.players))
	for _, p := range m.players {
		result = append(result, p)
	}

	return result
}

// GetPlayersCount returns the number of players on the map.
func (m *Map) GetPlayersCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.players)
}

// HavePlayers returns true if there are players on the map.
func (m *Map) HavePlayers() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.players) > 0
}

// Update processes map updates.
func (m *Map) Update(diff uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Process player updates
	for _, p := range m.players {
		if p.IsInWorld {
			// Player update logic would go here
			_ = p
		}
	}
}

// AddAreaTrigger adds an areatrigger to this map.
func (m *Map) AddAreaTrigger(at *areatrigger.AreaTrigger) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.areaTriggers = append(m.areaTriggers, at)
}

// GetAreaTriggers returns all areatriggers on this map.
func (m *Map) GetAreaTriggers() []*areatrigger.AreaTrigger {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.areaTriggers
}

// IsDungeon returns true if this is a dungeon map.
func (m *Map) IsDungeon() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 1 // INSTANCE_MULTIMAP
}

// IsRaid returns true if this is a raid map.
func (m *Map) IsRaid() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 4 // INSTANCE_RAID
}

// IsBattleground returns true if this is a battleground map.
func (m *Map) IsBattleground() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 3 // INSTANCE_BATTLEGROUND
}

// IsBattleArena returns true if this is an arena map.
func (m *Map) IsBattleArena() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 2 // INSTANCE_ARENA
}

// IsWorldMap returns true if this is a world map (open world).
func (m *Map) IsWorldMap() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 0 // INSTANCE_NONE
}

// IsInstanceable returns true if this map supports instancing.
func (m *Map) IsInstanceable() bool {
	return m.IsDungeon() || m.IsRaid() || m.IsBattleground() || m.IsBattleArena()
}

// GetMapDifficulty returns the map difficulty for the current spawn mode.
// This is a simplified version - actual implementation would look up MapDifficulty.dbc.
func (m *Map) GetMapDifficulty() uint32 {
	return uint32(m.SpawnMode)
}

// SendToPlayers sends a packet to all players on the map.
func (m *Map) SendToPlayers(data []byte) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, p := range m.players {
		if p.IsInWorld {
			// Would send packet via session - needs integration with WorldSession
			_ = data
			_ = p
		}
	}
}
