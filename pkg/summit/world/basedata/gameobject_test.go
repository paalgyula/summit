package basedata_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/stretchr/testify/assert"
)

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
		{"text", 9, [24]int32{}, 0},
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
		{"door", 0, [24]int32{}, 0},
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
		{"chest", 3, [24]int32{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := &basedata.GameObjectTemplate{Type: tt.typeID, Data: tt.data}
			assert.Equal(t, tt.expected, tpl.GetAutoCloseTime())
		})
	}
}

func TestGameObjectTemplate_QuestGiverData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 2,
		Data: [24]int32{1234, 0, 0, 5678},
	}

	assert.Equal(t, uint32(1234), tpl.GetQuestGiverLockID())
	assert.Equal(t, uint32(5678), tpl.GetQuestGiverGossipID())
}

func TestGameObjectTemplate_SpellCasterData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 22,
		Data: [24]int32{8001, 1},
	}

	assert.Equal(t, uint32(8001), tpl.GetSpellCasterSpellID())
	assert.True(t, tpl.IsSpellCasterPartyOnly())
}

func TestGameObjectTemplate_FishingHoleData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 17,
		Data: [24]int32{0, 20, 6001},
	}

	assert.Equal(t, uint32(0), tpl.GetFishingHoleLockID())
	assert.Equal(t, uint32(6001), tpl.GetFishingHoleLootID())
}

func TestGameObjectTemplate_GenericData(t *testing.T) {
	tpl := &basedata.GameObjectTemplate{
		Type: 5,
		Data: [24]int32{0, 0, 1, 0, 0, 100},
	}

	assert.True(t, tpl.IsGenericServerOnly())
	assert.Equal(t, int32(100), tpl.GetGenericQuestID())
}

func TestGameObjectSpawn_Fields(t *testing.T) {
	spawn := &basedata.GameObjectSpawn{
		GUID:      12345,
		Entry:     1001,
		MapID:     0,
		PhaseMask: 1,
		PosX:      -8940.0,
		PosY:      -140.0,
		PosZ:      83.0,
		O:         1.57,
		State:     0,
	}

	assert.Equal(t, uint32(12345), spawn.GUID)
	assert.Equal(t, uint32(1001), spawn.Entry)
	assert.Equal(t, float32(-8940.0), spawn.PosX)
}

func TestStore_LookupGameObjectTemplate(t *testing.T) {
	s := &basedata.Store{
		GOTemplates: []*basedata.GameObjectTemplate{
			{Entry: 1001, Type: 3, Name: "Chest"},
			{Entry: 1002, Type: 0, Name: "Door"},
		},
	}
	s.Index()

	tpl := s.LookupGameObjectTemplate(1001)
	assert.NotNil(t, tpl)
	assert.Equal(t, "Chest", tpl.Name)

	tpl = s.LookupGameObjectTemplate(9999)
	assert.Nil(t, tpl)
}

func TestStore_LookupGameObjectSpawn(t *testing.T) {
	s := &basedata.Store{
		GOSpawns: []*basedata.GameObjectSpawn{
			{GUID: 1, Entry: 1001, MapID: 0},
			{GUID: 2, Entry: 1002, MapID: 1},
		},
	}
	s.Index()

	spawn := s.LookupGameObjectSpawn(1)
	assert.NotNil(t, spawn)
	assert.Equal(t, uint32(1001), spawn.Entry)

	spawn = s.LookupGameObjectSpawn(9999)
	assert.Nil(t, spawn)
}

func TestStore_LookupGameObjectLoot(t *testing.T) {
	s := &basedata.Store{
		GOLoots: []*basedata.GameObjectLootEntry{
			{Entry: 5001, Item: 2589, Chance: 50.0},
			{Entry: 5001, Item: 2590, Chance: 30.0},
			{Entry: 6001, Item: 3000, Chance: 100.0},
		},
	}
	s.Index()

	loot := s.LookupGameObjectLoot(5001)
	assert.Len(t, loot, 2)
	assert.Equal(t, uint32(2589), loot[0].Item)
	assert.Equal(t, uint32(2590), loot[1].Item)

	loot = s.LookupGameObjectLoot(6001)
	assert.Len(t, loot, 1)

	loot = s.LookupGameObjectLoot(9999)
	assert.Nil(t, loot)
}
