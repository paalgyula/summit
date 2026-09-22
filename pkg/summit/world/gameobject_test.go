package world_test

import (
	"math"
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/loot"
	"github.com/paalgyula/summit/pkg/summit/world/object"
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
			{GUID: 1, Entry: 1001, MapID: 0, PhaseMask: 1, PosX: -8940, PosY: -140, PosZ: 83, State: uint8(basedata.GOStateReady)},
			{GUID: 2, Entry: 1002, MapID: 0, PhaseMask: 1, PosX: -8935, PosY: -135, PosZ: 83, O: 1.57, State: uint8(basedata.GOStateReady)},
			{GUID: 3, Entry: 1003, MapID: 1, PhaseMask: 1, PosX: 100, PosY: 200, PosZ: 300, State: uint8(basedata.GOStateReady)},
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
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 1, Entry: 1001, State: uint8(basedata.GOStateReady)},
	}

	g := world.NewGameObject(tpl, data)

	assert.Equal(t, world.GO_READY, g.GetLootState())
	require.NoError(t, g.Use(world.GameObjectUseContext{}))
	assert.Equal(t, world.GO_ACTIVATED, g.GetLootState())
}

func TestGameObjectUse_Door(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1002, Type: 0, Name: "Door", Size: 1.0,
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 2, Entry: 1002, State: uint8(basedata.GOStateReady)},
	}

	g := world.NewGameObject(tpl, data)
	ctx := world.GameObjectUseContext{}

	assert.Equal(t, world.GOStateReady, g.GOState)
	require.NoError(t, g.Use(ctx))
	assert.Equal(t, world.GOStateActive, g.GOState)
	require.NoError(t, g.Use(ctx))
	assert.Equal(t, world.GOStateReady, g.GOState)
}

func TestGameObjectUse_Button(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1003, Type: 1, Name: "Button", Size: 1.0,
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 3, Entry: 1003, State: uint8(basedata.GOStateReady)},
	}

	g := world.NewGameObject(tpl, data)
	ctx := world.GameObjectUseContext{}

	assert.Equal(t, world.GOStateReady, g.GOState)
	require.NoError(t, g.Use(ctx))
	assert.Equal(t, world.GOStateActive, g.GOState)
	require.NoError(t, g.Use(ctx))
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

func TestQuaternionFromSpawn_Orientation(t *testing.T) {
	q := world.QuaternionFromSpawn([4]float32{}, float32(math.Pi/2))

	assert.InDelta(t, 0, q[0], 1e-6)
	assert.InDelta(t, 0, q[1], 1e-6)
	assert.InDelta(t, math.Sqrt2/2, q[2], 1e-6)
	assert.InDelta(t, math.Sqrt2/2, q[3], 1e-6)
}

func TestQuaternionFromSpawn_ExplicitIsNormalized(t *testing.T) {
	q := world.QuaternionFromSpawn([4]float32{1, 0, 0, 1}, 0)

	length := math.Sqrt(float64(q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3]))
	assert.InDelta(t, 1.0, length, 1e-6)
	assert.InDelta(t, math.Sqrt2/2, q[0], 1e-6)
	assert.InDelta(t, math.Sqrt2/2, q[3], 1e-6)
}

func TestPackQuaternion_Identity(t *testing.T) {
	assert.Equal(t, int64(0), world.PackQuaternion([4]float32{0, 0, 0, 1}))
}

func TestPackQuaternion_YawHalfPi(t *testing.T) {
	q := world.QuaternionFromSpawn([4]float32{}, float32(math.Pi/2))

	// z = round(0.70710678 * 2^20) = 741455, x = y = 0
	assert.Equal(t, int64(741455), world.PackQuaternion(q))
}

func TestGameObject_RotationFieldsInitialised(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1002, Type: 0, Name: "Door", Size: 1.0,
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 2, Entry: 1002, O: 1.57},
	}

	g := world.NewGameObject(tpl, data)

	// Parent rotation defaults to identity.
	assert.Equal(t, float32(0), g.Object.GetFloatValue(object.GameobjectParentrotation))
	assert.Equal(t, float32(0), g.Object.GetFloatValue(object.GameobjectParentrotation+1))
	assert.Equal(t, float32(0), g.Object.GetFloatValue(object.GameobjectParentrotation+2))
	assert.Equal(t, float32(1), g.Object.GetFloatValue(object.GameobjectParentrotation+3))

	// World rotation is derived from the orientation and packs non-zero.
	assert.NotEqual(t, int64(0), g.PackedRotation)
	assert.Equal(t, g.PackedRotation, g.GetPackedRotation())
}

func TestGameObject_InteractionDistance(t *testing.T) {
	tests := []struct {
		name     string
		typeID   world.GameObjectType
		expected float32
	}{
		{"questgiver", world.GameObjectTypeQuestGiver, 5.5555553},
		{"door", world.GameObjectTypeDoor, 5.0},
		{"chair", world.GameObjectTypeChair, 3.0},
		{"fishinghole", world.GameObjectTypeFishingHole, 20.5},
		{"mailbox", world.GameObjectTypeMailbox, 10.0},
		{"chest", world.GameObjectTypeChest, 5.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &world.GameObject{Type: tt.typeID}
			assert.InDelta(t, tt.expected, g.InteractionDistance(), 1e-6)
		})
	}
}

func TestGameObject_IsWithinInteractionDistance(t *testing.T) {
	g := &world.GameObject{Type: world.GameObjectTypeChest, X: 10, Y: 20, Z: 30}

	assert.True(t, g.IsWithinInteractionDistance(10, 20, 30))
	assert.True(t, g.IsWithinInteractionDistance(14, 20, 30))  // 4 yd away
	assert.False(t, g.IsWithinInteractionDistance(16, 20, 30)) // 6 yd away
}

func TestGameObject_CreateUpdateType(t *testing.T) {
	assert.Equal(t, wow.ObjectUpdateType(wow.UpdateTypeCreateObject2),
		(&world.GameObject{Type: world.GameObjectTypeTrap}).CreateUpdateType())
	assert.Equal(t, wow.ObjectUpdateType(wow.UpdateTypeCreateObject2),
		(&world.GameObject{Type: world.GameObjectTypeFlagStand}).CreateUpdateType())
	assert.Equal(t, wow.ObjectUpdateType(wow.UpdateTypeCreateObject),
		(&world.GameObject{Type: world.GameObjectTypeDoor}).CreateUpdateType())
	assert.Equal(t, wow.ObjectUpdateType(wow.UpdateTypeCreateObject),
		(&world.GameObject{Type: world.GameObjectTypeChest}).CreateUpdateType())
}

func TestGameObjectTemplate_AddonAccessors(t *testing.T) {
	trap := &basedata.GameObjectTemplate{
		Type: 6, // Trap
		// Data[3]=spellId, Data[4]=type, Data[5]=cooldown(secs), Data[6]=autoClose
		Data: [24]int32{0, 60, 10, 8001, 1, 30, 4000},
	}
	assert.Equal(t, uint32(8001), trap.GetSpellID())
	assert.Equal(t, uint32(30000), trap.GetCooldown())
	assert.Equal(t, uint32(4000), trap.GetAutoCloseTime())

	spellcaster := &basedata.GameObjectTemplate{
		Type: 22, // SpellCaster
		// Data[0]=spellId, Data[1]=charges, Data[3]=allowMounted
		Data: [24]int32{9001, 5, 0, 1},
	}
	assert.Equal(t, uint32(9001), spellcaster.GetSpellID())
	assert.Equal(t, uint32(5), spellcaster.GetCharges())
	assert.True(t, spellcaster.IsUsableMounted())

	chest := &basedata.GameObjectTemplate{
		Type: 3, // Chest
		// Data[1]=lootId, Data[3]=consumable, Data[7]=linkedTrapId, Data[8]=questId
		Data: [24]int32{0, 5001, 0, 1, 0, 0, 0, 7001, 1234},
	}
	assert.True(t, chest.IsDespawnAtAction())
	assert.Equal(t, uint32(7001), chest.GetLinkedGameObjectEntry())
	assert.Equal(t, int32(1234), chest.GetQuestID())

	goober := &basedata.GameObjectTemplate{
		Type: 10, // Goober
		// Data[3]=autoClose, Data[6]=cooldown, Data[10]=spellId, Data[17]=allowMounted
		Data: [24]int32{0, 0, 0, 3000, 0, 0, 60, 0, 0, 0, 8002, 0, 0, 0, 0, 0, 0, 1},
	}
	assert.True(t, goober.IsUsableMounted())
	assert.Equal(t, uint32(60000), goober.GetCooldown())
	assert.Equal(t, uint32(8002), goober.GetSpellID())
}

func TestGameObject_Flags(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1001, Type: 3, Name: "Chest", Size: 1.0,
			Flags:   uint32(basedata.GOFlagLocked),
			Faction: 35,
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 1, Entry: 1001, State: uint8(basedata.GOStateReady)},
	}

	g := world.NewGameObject(tpl, data)

	assert.Equal(t, uint32(35), g.Faction)
	assert.True(t, g.HasGameObjectFlag(basedata.GOFlagLocked))

	// A locked chest cannot be opened.
	err := g.Use(world.GameObjectUseContext{})
	assert.ErrorIs(t, err, world.ErrGameObjectLocked)

	g.RemoveGameObjectFlag(basedata.GOFlagLocked)
	assert.False(t, g.HasGameObjectFlag(basedata.GOFlagLocked))

	g.SetGameObjectFlag(basedata.GOFlagNoDespawn)
	assert.True(t, g.HasGameObjectFlag(basedata.GOFlagNoDespawn))
}

func TestGameObject_DoorAutoClose(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1002, Type: 0, Name: "Door", Size: 1.0,
			// Data[2]=autoCloseTime(ms)
			Data: [24]int32{0, 0, 5000},
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 2, Entry: 1002, State: uint8(basedata.GOStateReady)},
	}

	g := world.NewGameObject(tpl, data)
	ctx := world.GameObjectUseContext{}

	require.NoError(t, g.Use(ctx))
	assert.Equal(t, world.GOStateActive, g.GOState)
	assert.Equal(t, world.GO_ACTIVATED, g.GetLootState())

	// Force the auto-close timer to expire and tick.
	g.CooldownAt = time.Now().Add(-time.Second)
	g.Update(50, ctx)

	assert.Equal(t, world.GOStateReady, g.GOState)
	assert.Equal(t, world.GO_READY, g.GetLootState())
}

func TestGameObject_TrapArming(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1005, Type: 6, Name: "Trap", Size: 1.0,
			Data: [24]int32{0, 60, 10, 8001, 0, 30},
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 5, Entry: 1005, State: uint8(basedata.GOStateReady)},
	}

	g := world.NewGameObject(tpl, data)
	g.SetLootState(world.GO_NOT_READY)

	g.Update(50, world.GameObjectUseContext{})
	assert.Equal(t, world.GO_READY, g.GetLootState())
}

func TestGameObject_ChestRestock(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry: 1001, Type: 3, Name: "Chest", Size: 1.0,
			// Data[2]=restockTime(secs), Data[3]=consumable(0)
			Data: [24]int32{0, 5001, 60, 0},
		},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 1, Entry: 1001, State: uint8(basedata.GOStateReady)},
	}

	g := world.NewGameObject(tpl, data)
	g.SetLootState(world.GO_NOT_READY)
	g.RestockAt = time.Now().Add(-time.Second)

	g.Update(50, world.GameObjectUseContext{})
	assert.Equal(t, world.GO_READY, g.GetLootState())
}

func TestGameObject_SummonedDespawn(t *testing.T) {
	tpl := &world.GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{Entry: 1, Type: 3, Name: "Summoned", Size: 1.0},
	}
	data := &world.GameObjectData{
		GameObjectSpawn: &basedata.GameObjectSpawn{GUID: 1, Entry: 1},
	}

	g := world.NewGameObject(tpl, data)
	g.SetOwnerGUID(wow.NewGUID(wow.PlayerGUID, 42))
	g.Loot = &loot.Loot{Gold: 5}
	g.SetLootState(world.GO_JUST_DEACTIVATED)

	g.Update(50, world.GameObjectUseContext{})

	assert.Nil(t, g.Loot)
	assert.False(t, g.SpawnedByDefault)
}
