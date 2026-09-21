package world_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGOStore() *basedata.Store {
	return &basedata.Store{
		GOTemplates: []*basedata.GameObjectTemplate{
			{
				Entry: 1001, Type: 3, DisplayID: 31, Name: "Chest", Size: 1.0,
				// Chest: Data[0]=lockId, Data[1]=lootId
				Data: [24]int32{0, 5001},
			},
			{
				Entry: 1002, Type: 0, DisplayID: 57, Name: "Door", Size: 1.0,
				// Door: Data[0]=startOpen, Data[1]=lockId, Data[2]=autoCloseTime
				Data: [24]int32{0, 0, 5000},
			},
			{
				Entry: 1003, Type: 1, DisplayID: 57, Name: "Button", Size: 1.0,
				// Button: Data[0]=startOpen, Data[1]=lockId, Data[2]=autoCloseTime, Data[3]=linkedTrap
				Data: [24]int32{0, 1234, 3000, 0},
			},
			{
				Entry: 1004, Type: 10, DisplayID: 293, Name: "Goober", Size: 1.0,
				// Goober: Data[0]=lockId, Data[1]=questId, Data[2]=eventId, Data[3]=autoCloseTime
				Data: [24]int32{5678, 0, 0, 2000},
			},
			{
				Entry: 1005, Type: 6, DisplayID: 293, Name: "Trap", Size: 1.0,
				// Trap: Data[0]=lockId, Data[1]=level, Data[2]=diameter, Data[3]=spellId
				Data: [24]int32{0, 60, 10, 8001},
			},
			{
				Entry: 1006, Type: 9, DisplayID: 293, Name: "Sign", Size: 1.0,
				// Text: Data[0]=pageID
				Data: [24]int32{42},
			},
			{
				Entry: 1007, Type: 17, DisplayID: 31, Name: "FishingHole", Size: 1.0,
				// FishingHole: Data[0]=lockId, Data[1]=radius, Data[2]=lootId
				Data: [24]int32{0, 20, 6001},
			},
		},
		GOSpawns: []*basedata.GameObjectSpawn{
			{GUID: 1, Entry: 1001, MapID: 0, PhaseMask: 1, PosX: -8940, PosY: -140, PosZ: 83, State: 0},
			{GUID: 2, Entry: 1002, MapID: 0, PhaseMask: 1, PosX: -8935, PosY: -135, PosZ: 83, O: 1.57, State: 0},
			{GUID: 3, Entry: 1003, MapID: 1, PhaseMask: 1, PosX: 100, PosY: 200, PosZ: 300, State: 0},
		},
	}
}

func TestGameObjectTemplate_ChestData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 3,
		Data: [24]int32{0, 5001, 60, 1},
	}

	assert.Equal(t, uint32(0), tpl.GetChestLockID())
	assert.Equal(t, uint32(5001), tpl.GetChestLootID())
	assert.Equal(t, uint32(60), tpl.GetChestRestockTime())
	assert.True(t, tpl.IsChestConsumable())
}

func TestGameObjectTemplate_DoorData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 0,
		Data: [24]int32{0, 1234, 5000},
	}

	assert.Equal(t, uint32(0), tpl.GetDoorStartOpen())
	assert.Equal(t, uint32(1234), tpl.GetDoorLockID())
	assert.Equal(t, uint32(5000), tpl.GetDoorAutoClose())
}

func TestGameObjectTemplate_ButtonData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 1,
		Data: [24]int32{0, 1234, 3000, 5678},
	}

	assert.Equal(t, uint32(0), tpl.GetButtonStartOpen())
	assert.Equal(t, uint32(1234), tpl.GetButtonLockID())
	assert.Equal(t, uint32(3000), tpl.GetButtonAutoClose())
	assert.Equal(t, uint32(5678), tpl.GetButtonLinkedTrap())
}

func TestGameObjectTemplate_GooberData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 10,
		Data: [24]int32{5678, 100, 200, 3000, 1, 1, 60, 0, 0, 0, 8001},
	}

	assert.Equal(t, uint32(5678), tpl.GetGooberLockID())
	assert.Equal(t, int32(100), tpl.GetGooberQuestID())
	assert.Equal(t, uint32(200), tpl.GetGooberEventID())
	assert.Equal(t, uint32(3000), tpl.GetGooberAutoClose())
	assert.Equal(t, uint32(1), tpl.GetGooberCustomAnim())
	assert.True(t, tpl.IsGooberConsumable())
	assert.Equal(t, uint32(60), tpl.GetGooberCooldown())
	assert.Equal(t, uint32(8001), tpl.GetGooberSpellID())
}

func TestGameObjectTemplate_TrapData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 6,
		Data: [24]int32{0, 60, 10, 8001, 1, 30},
	}

	assert.Equal(t, uint32(0), tpl.GetTrapLockID())
	assert.Equal(t, uint32(8001), tpl.GetTrapSpellID())
	assert.Equal(t, uint32(30), tpl.GetTrapCooldown())
}

func TestGameObjectTemplate_LockID(t *testing.T) {
	tests := []struct {
		name     string
		typeID   uint8
		data     [24]int32
		expected uint32
	}{
		{"door", 0, [24]int32{0, 1234, 0}, 1234},
		{"button", 1, [24]int32{0, 5678, 0}, 5678},
		{"chest", 3, [24]int32{9999, 0}, 9999},
		{"trap", 6, [24]int32{1111, 0}, 1111},
		{"goober", 10, [24]int32{2222, 0}, 2222},
		{"text", 9, [24]int32{}, 0}, // no lock
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := &basedata.GameObjectTemplate{Type: tt.typeID, Data: tt.data}
			assert.Equal(t, tt.expected, tpl.GetLockID())
		})
	}
}

func TestGameObjectTemplate_LootID(t *testing.T) {
	tests := []struct {
		name     string
		typeID   uint8
		data     [24]int32
		expected uint32
	}{
		{"chest", 3, [24]int32{0, 5001}, 5001},
		{"fishing_hole", 17, [24]int32{0, 20, 6001}, 6001},
		{"door", 0, [24]int32{}, 0}, // no loot
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := &basedata.GameObjectTemplate{Type: tt.typeID, Data: tt.data}
			assert.Equal(t, tt.expected, tpl.GetLootID())
		})
	}
}

func TestGameObjectTemplate_AutoCloseTime(t *testing.T) {
	tests := []struct {
		name     string
		typeID   uint8
		data     [24]int32
		expected uint32
	}{
		{"door", 0, [24]int32{0, 0, 5000}, 5000},
		{"button", 1, [24]int32{0, 0, 3000}, 3000},
		{"goober", 10, [24]int32{0, 0, 0, 2000}, 2000},
		{"chest", 3, [24]int32{}, 0}, // no auto-close
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := &basedata.GameObjectTemplate{Type: tt.typeID, Data: tt.data}
			assert.Equal(t, tt.expected, tpl.GetAutoCloseTime())
		})
	}
}

func TestNewGameObject_Fields(t *testing.T) {
	basedata.SetInstance(testGOStore())
	defer basedata.SetInstance(nil)

	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: basedata.GetInstance().GameObjectTemplates[1001],
	}
	data := &world.GameObjectData{
		GameObjectSpawn: basedata.GetInstance().GameObjectSpawns[1],
	}

	g := world.NewGameObject(tpl, data)

	assert.Equal(t, uint32(1), g.ID)
	assert.Equal(t, uint32(1001), g.Entry)
	assert.Equal(t, "Chest", g.Name)
	assert.Equal(t, world.GameObjectTypeChest, g.Type)
	assert.Equal(t, uint32(31), g.DisplayID)
	assert.Equal(t, float32(1.0), g.Size)
	assert.Equal(t, float32(-8940), g.X)
	assert.Equal(t, float32(-140), g.Y)
	assert.Equal(t, float32(83), g.Z)
	assert.Equal(t, uint32(0), g.Map)
	assert.Equal(t, world.GOStateReady, g.GOState)
}

func TestNewGameObject_UpdateFields(t *testing.T) {
	basedata.SetInstance(testGOStore())
	defer basedata.SetInstance(nil)

	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: basedata.GetInstance().GameObjectTemplates[1001],
	}
	data := &world.GameObjectData{
		GameObjectSpawn: basedata.GetInstance().GameObjectSpawns[1],
	}

	g := world.NewGameObject(tpl, data)

	// Check GUID type is set correctly
	guid := g.GetGUID()
	assert.Equal(t, wow.GameObjectGUID, guid.High())

	// Check object type (at index 2)
	assert.Equal(t, uint32(wow.TypeIDGameObject), g.Object.GetUInt32Value(2))
	// Check entry (at index 3)
	assert.Equal(t, uint32(1001), g.Object.GetUInt32Value(3))
}

func TestGameObjectUse_Chest(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1001, Type: 3, Name: "Chest", Size: 1.0,
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 1, Entry: 1001},
	}

	g := world.NewGameObject(tpl, data)

	assert.False(t, g.IsLooted())
	g.Use(nil)
	assert.True(t, g.IsLooted())
}

func TestGameObjectUse_Door(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1002, Type: 0, Name: "Door", Size: 1.0,
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 2, Entry: 1002},
	}

	g := world.NewGameObject(tpl, data)

	assert.Equal(t, world.GOStateReady, g.GOState)
	g.Use(nil)
	assert.Equal(t, world.GOStateActive, g.GOState)
	g.Use(nil)
	assert.Equal(t, world.GOStateReady, g.GOState)
}

func TestGameObjectUse_Button(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1003, Type: 1, Name: "Button", Size: 1.0,
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 3, Entry: 1003},
	}

	g := world.NewGameObject(tpl, data)

	assert.Equal(t, world.GOStateReady, g.GOState)
	g.Use(nil)
	assert.Equal(t, world.GOStateActive, g.GOState)
	g.Use(nil)
	assert.Equal(t, world.GOStateReady, g.GOState)
}

func TestGameObjectManager_SpawnAndRetrieve(t *testing.T) {
	basedata.SetInstance(testGOStore())
	defer basedata.SetInstance(nil)

	gm := world.NewGameObjectManager()

	// Get by spawn ID
	g := gm.GetObject(1)
	require.NotNil(t, g)
	assert.Equal(t, uint32(1001), g.Entry)

	// Get by GUID
	guid := wow.NewGUID(wow.GameObjectGUID, 2)
	g = gm.GetObjectByGUID(guid)
	require.NotNil(t, g)
	assert.Equal(t, uint32(1002), g.Entry)

	// Get non-existent
	g = gm.GetObject(9999)
	assert.Nil(t, g)
}

func TestGameObjectManager_GetObjectsInMap(t *testing.T) {
	basedata.SetInstance(testGOStore())
	defer basedata.SetInstance(nil)

	gm := world.NewGameObjectManager()

	// Map 0 has 2 objects (GUIDs 1, 2)
	objects := gm.GetObjectsInMap(0)
	assert.Len(t, objects, 2)

	// Map 1 has 1 object (GUID 3)
	objects = gm.GetObjectsInMap(1)
	assert.Len(t, objects, 1)

	// Map 99 has none
	objects = gm.GetObjectsInMap(99)
	assert.Len(t, objects, 0)
}

func TestGameObjectManager_GetTemplate(t *testing.T) {
	basedata.SetInstance(testGOStore())
	defer basedata.SetInstance(nil)

	gm := world.NewGameObjectManager()

	tpl := gm.GetTemplate(1001)
	require.NotNil(t, tpl)
	assert.Equal(t, "Chest", tpl.Name)

	tpl = gm.GetTemplate(9999)
	assert.Nil(t, tpl)
}

func TestGameObjectManager_SpawnObject(t *testing.T) {
	basedata.SetInstance(testGOStore())
	defer basedata.SetInstance(nil)

	gm := world.NewGameObjectManager()

	// Spawn a new object
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{
			GUID:  100,
			Entry: 1001,
			MapID: 5,
		},
	}
	gm.SpawnObject(1001, data)

	g := gm.GetObject(100)
	require.NotNil(t, g)
	assert.Equal(t, uint32(5), g.Map)

	// Try spawning with non-existent template
	data2 := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{
			GUID:  101,
			Entry: 9999,
		},
	}
	gm.SpawnObject(9999, data2)
	g = gm.GetObject(101)
	assert.Nil(t, g) // should not be spawned
}
