package mapmanager

import (
	"sync"

	"github.com/paalgyula/summit/pkg/summit/world/areatrigger"
	"github.com/rs/zerolog/log"
)

// MapManager manages all maps in the world.
type MapManager struct {
	// Base maps (non-instanced)
	baseMaps map[uint32]*Map

	// Instanced maps: mapId -> instanceId -> Map
	instances map[uint32]map[uint32]*Map

	// Next instance ID
	nextInstanceID uint32

	// AreaTrigger manager reference
	areaTriggerMgr *areatrigger.Manager

	mutex sync.RWMutex
}

var (
	globalMapManager *MapManager
	mapManagerOnce   sync.Once
)

// GetMapManager returns the global MapManager instance.
func GetMapManager() *MapManager {
	mapManagerOnce.Do(func() {
		globalMapManager = &MapManager{
			baseMaps:       make(map[uint32]*Map),
			instances:      make(map[uint32]map[uint32]*Map),
			nextInstanceID: 1,
		}
	})

	return globalMapManager
}

// SetAreaTriggerManager sets the areatrigger manager for map integration.
func (mm *MapManager) SetAreaTriggerManager(mgr *areatrigger.Manager) {
	mm.areaTriggerMgr = mgr
}

// CreateBaseMap creates or returns the base map for the given map ID.
func (mm *MapManager) CreateBaseMap(mapID uint32) *Map {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if m, ok := mm.baseMaps[mapID]; ok {
		return m
	}

	// Create new base map
	entry := mm.getMapEntry(mapID)
	m := NewMap(mapID, 0, entry)

	mm.baseMaps[mapID] = m

	// Load areatriggers for this map
	if mm.areaTriggerMgr != nil {
		triggers := mm.areaTriggerMgr.GetTriggersForMap(mapID)
		for _, at := range triggers {
			m.AddAreaTrigger(at)
		}
	}

	log.Info().Uint32("mapID", mapID).Msg("Created base map")

	return m
}

// FindBaseMap returns the base map for the given map ID, or nil if not found.
func (mm *MapManager) FindBaseMap(mapID uint32) *Map {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return mm.baseMaps[mapID]
}

// FindMap finds a specific map instance.
func (mm *MapManager) FindMap(mapID, instanceID uint32) *Map {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	if instanceID == 0 {
		return mm.baseMaps[mapID]
	}

	if instances, ok := mm.instances[mapID]; ok {
		return instances[instanceID]
	}

	return nil
}

// CreateMap creates a map for a player (handles instancing).
func (mm *MapManager) CreateMap(mapID uint32, p interface{}) *Map {
	// For non-instanced maps, return/create base map
	entry := mm.getMapEntry(mapID)
	if entry == nil || !entry.IsInstanceable() {
		return mm.CreateBaseMap(mapID)
	}

	// For instanced maps, create a new instance
	return mm.createInstance(mapID, entry)
}

// createInstance creates a new instance of a map.
func (mm *MapManager) createInstance(mapID uint32, entry *MapEntry) *Map {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	// Get next instance ID
	instanceID := mm.nextInstanceID
	mm.nextInstanceID++

	// Create instance map
	m := NewMap(mapID, instanceID, entry)
	m.SpawnMode = 0 // Normal difficulty

	// Initialize instance map
	if mm.instances[mapID] == nil {
		mm.instances[mapID] = make(map[uint32]*Map)
	}
	mm.instances[mapID][instanceID] = m

	log.Info().
		Uint32("mapID", mapID).
		Uint32("instanceID", instanceID).
		Msg("Created map instance")

	return m
}

// Update updates all maps.
func (mm *MapManager) Update(diff uint32) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	// Update base maps
	for _, m := range mm.baseMaps {
		m.Update(diff)
	}

	// Update instances
	for _, instances := range mm.instances {
		for _, m := range instances {
			m.Update(diff)
		}
	}
}

// DoForAllMaps iterates over all maps and executes the callback.
func (mm *MapManager) DoForAllMaps(callback func(*Map)) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	for _, m := range mm.baseMaps {
		callback(m)
	}

	for _, instances := range mm.instances {
		for _, m := range instances {
			callback(m)
		}
	}
}

// GetMapCount returns the total number of maps (base + instances).
func (mm *MapManager) GetMapCount() int {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	count := len(mm.baseMaps)
	for _, instances := range mm.instances {
		count += len(instances)
	}

	return count
}

// getMapEntry returns the map entry from DBC data.
// This is a placeholder - actual implementation would load from Map.dbc.
func (mm *MapManager) getMapEntry(mapID uint32) *MapEntry {
	// TODO: Load from DBC or database
	return &MapEntry{
		ID:           mapID,
		InstanceType: 0,
		Name:         "Map " + string(rune(mapID)),
	}
}

// IsInstanceable returns true if the map supports instancing.
func (e *MapEntry) IsInstanceable() bool {
	return e.InstanceType != 0 // Non-world maps are instanceable
}
