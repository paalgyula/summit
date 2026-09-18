package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

// NPC represents a spawned creature in the world.
type NPC struct {
	*object.Object
	*object.Unit

	ID           uint32
	EntryID      uint32
	Name         string
	DisplayID    uint32
	Faction      uint32
	Level        uint8
	Health       uint32
	MaxHealth    uint32
	BaseDamage   float32
	X, Y, Z, O  float32
	Map          uint32
}

// NewNPC creates a new NPC with the given parameters.
func NewNPC(entryID uint32, name string, displayID, faction uint32, level uint8, health uint32, x, y, z, o float32, mapID uint32) *NPC {
	n := &NPC{
		Object:    object.NewObject(),
		Unit:      object.NewUnit(),
		ID:        entryID,
		EntryID:   entryID,
		Name:      name,
		DisplayID: displayID,
		Faction:   faction,
		Level:     level,
		Health:    health,
		MaxHealth: health,
		X:         x,
		Y:         y,
		Z:         z,
		O:         o,
		Map:       mapID,
	}

	n.init()

	return n
}

// init sets up the NPC's update fields.
func (n *NPC) init() {
	n.Object.InitValues(int(object.UnitEnd))

	// GUID
	guid := wow.NewGUID(wow.UnitGUID, n.ID)
	n.Object.SetGUID(guid)

	// Object fields
	n.Object.SetUInt32Value(object.ObjectFieldGuid, uint32(guid))
	n.Object.SetUInt32Value(object.ObjectFieldGuid+1, uint32(uint64(guid)>>32))
	n.Object.SetUInt32Value(object.ObjectFieldType, uint32(wow.TypeIDUnit))
	n.Object.SetUInt32Value(object.ObjectFieldEntry, n.EntryID)
	n.Object.SetFloatValue(object.ObjectFieldScaleX, 1.0)

	// Unit fields
	n.Object.SetUInt32Value(object.UnitFieldDisplayid, n.DisplayID)
	n.Object.SetUInt32Value(object.UnitFieldNativedisplayid, n.DisplayID)
	n.Object.SetUInt32Value(object.UnitFieldFactiontemplate, n.Faction)
	n.Object.SetUInt32Value(object.UnitFieldLevel, uint32(n.Level))
	n.Object.SetUInt32Value(object.UnitFieldHealth, n.Health)
	n.Object.SetUInt32Value(object.UnitFieldMaxhealth, n.MaxHealth)

	// Unit flags: UNIT_FLAG_PVP_ATTACKABLE
	n.Object.SetUInt32Value(object.UnitFieldFlags, 0x08)

	// Bounding radius
	n.Object.SetFloatValue(object.UnitFieldBoundingradius, 0.388999998569489)
	n.Object.SetFloatValue(object.UnitFieldCombatreach, 1.5)

	// Base attack time (2000ms)
	n.Object.SetUInt32Value(object.UnitFieldBaseattacktime, 2000)
}

// GetGUID returns the NPC's GUID.
func (n *NPC) GetGUID() wow.GUID {
	return wow.NewGUID(wow.UnitGUID, n.ID)
}

// GetHealth returns the NPC's current health.
func (n *NPC) GetHealth() uint32 {
	return n.Health
}

// SetHealth sets the NPC's health, clamped to [0, MaxHealth].
func (n *NPC) SetHealth(v uint32) {
	if v > n.MaxHealth {
		v = n.MaxHealth
	}

	n.Health = v
	n.Object.SetUInt32Value(object.UnitFieldHealth, v)
}

// IsDead returns true if the NPC has 0 health.
func (n *NPC) IsDead() bool {
	return n.Health == 0
}

// SpawnManager holds all spawned NPCs.
type SpawnManager struct {
	npcs map[uint32]*NPC
}

// NewSpawnManager creates a new spawn manager with hardcoded test NPCs.
func NewSpawnManager() *SpawnManager {
	sm := &SpawnManager{
		npcs: make(map[uint32]*NPC),
	}

	// Spawn a training dummy at starting location
	sm.SpawnNPC(NewNPC(
		100001,              // entry ID
		"Training Dummy",    // name
		49,                  // displayID (human male)
		1,                   // faction (hostile)
		1,                   // level
		50,                  // health
		-8949.95,            // X (Elwynn Forest starting area)
		-132.66,             // Y
		83.53,               // Z
		1.0,                 // O
		0,                   // map (Eastern Kingdoms)
	))

	return sm
}

// SpawnNPC adds an NPC to the world.
func (sm *SpawnManager) SpawnNPC(npc *NPC) {
	sm.npcs[npc.ID] = npc
}

// RemoveNPC removes an NPC from the world.
func (sm *SpawnManager) RemoveNPC(id uint32) {
	delete(sm.npcs, id)
}

// GetNPC returns an NPC by ID.
func (sm *SpawnManager) GetNPC(id uint32) *NPC {
	return sm.npcs[id]
}

// GetNPCsInMap returns all NPCs in the given map.
func (sm *SpawnManager) GetNPCsInMap(mapID uint32) []*NPC {
	var result []*NPC

	for _, npc := range sm.npcs {
		if npc.Map == mapID {
			result = append(result, npc)
		}
	}

	return result
}
