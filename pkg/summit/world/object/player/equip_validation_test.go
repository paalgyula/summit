package player_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBaseData creates a base data store with test item templates.
func testBaseData() *basedata.Store {
	return &basedata.Store{
		Items: []*basedata.ItemTemplate{
			// Sword (main hand weapon)
			{Entry: 1001, Class: 2, SubClass: 7, DisplayID: 5000, Name: "Test Sword",
				Quality: 2, InventoryType: 21, RequiredLevel: 10, AllowableClass: -1, AllowableRace: -1,
				Damage: [2]basedata.ItemDamage{{Min: 10, Max: 20, Type: 0}}, Delay: 2000},
			// Shield (off hand)
			{Entry: 1002, Class: 4, SubClass: 6, DisplayID: 5001, Name: "Test Shield",
				Quality: 2, InventoryType: 14, RequiredLevel: 10, AllowableClass: -1, AllowableRace: -1,
				Armor: 100},
			// Helmet
			{Entry: 1003, Class: 4, SubClass: 4, DisplayID: 5002, Name: "Test Plate Helm",
				Quality: 3, InventoryType: 1, RequiredLevel: 20, AllowableClass: -1, AllowableRace: -1,
				Armor: 500, StatsCount: 2,
				Stats: [10]basedata.ItemStat{{Type: 4, Value: 10}, {Type: 7, Value: 20}}},
			// Boots (level 60 requirement)
			{Entry: 1004, Class: 4, SubClass: 4, DisplayID: 5003, Name: "Test Plate Boots",
				Quality: 2, InventoryType: 8, RequiredLevel: 60, AllowableClass: -1, AllowableRace: -1,
				Armor: 300},
			// Paladin-only libram
			{Entry: 1005, Class: 4, SubClass: 7, DisplayID: 5004, Name: "Test Libram",
				Quality: 2, InventoryType: 28, RequiredLevel: 1, AllowableClass: 2, AllowableRace: -1},
			// Horde-only item (orc only)
			{Entry: 1006, Class: 4, SubClass: 4, DisplayID: 5005, Name: "Orc Only Helm",
				Quality: 2, InventoryType: 1, RequiredLevel: 1, AllowableClass: -1, AllowableRace: 2},
			// Two-handed weapon
			{Entry: 1007, Class: 2, SubClass: 8, DisplayID: 5006, Name: "Test 2H Sword",
				Quality: 2, InventoryType: 17, RequiredLevel: 1, AllowableClass: -1, AllowableRace: -1,
				Damage: [2]basedata.ItemDamage{{Min: 30, Max: 50, Type: 0}}, Delay: 3000},
			// Consumable (not equippable)
			{Entry: 1008, Class: 0, SubClass: 0, DisplayID: 5007, Name: "Test Bread",
				Quality: 1, InventoryType: 0, RequiredLevel: 1, AllowableClass: -1, AllowableRace: -1},
			// Ring
			{Entry: 1009, Class: 4, SubClass: 0, DisplayID: 5008, Name: "Test Ring",
				Quality: 3, InventoryType: 11, RequiredLevel: 1, AllowableClass: -1, AllowableRace: -1},
			// Trinket
			{Entry: 1010, Class: 4, SubClass: 0, DisplayID: 5009, Name: "Test Trinket",
				Quality: 3, InventoryType: 12, RequiredLevel: 1, AllowableClass: -1, AllowableRace: -1},
			// Shoulder (armor with stats)
			{Entry: 1011, Class: 4, SubClass: 4, DisplayID: 5010, Name: "Test Plate Shoulders",
				Quality: 2, InventoryType: 3, RequiredLevel: 1, AllowableClass: -1, AllowableRace: -1,
				Armor: 200, StatsCount: 1,
				Stats: [10]basedata.ItemStat{{Type: 7, Value: 15}}},
			// Cloak with resistance
			{Entry: 1012, Class: 4, SubClass: 1, DisplayID: 5011, Name: "Test Cloak",
				Quality: 2, InventoryType: 16, RequiredLevel: 1, AllowableClass: -1, AllowableRace: -1,
				Armor: 50, FireRes: 10, FrostRes: 5},
		},
	}
}

// testPlayer creates a test player at the given level with a class.
func testPlayer(level uint8, class wow.PlayerClass, race wow.PlayerRace) *player.Player {
	p := player.NewPlayer()
	p.Level = level
	p.Class = class
	p.Race = race
	p.Inventory = player.NewInventory()
	p.Object = object.NewObject()
	p.Object.InitValues(int(object.UnitEnd))
	return p
}

func TestFindEquipSlotForItem(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	tests := []struct {
		name     string
		entry    uint32
		expected int
	}{
		{"sword goes to mainhand", 1001, player.EquipmentSlotMainHand},
		{"shield goes to offhand", 1002, player.EquipmentSlotOffHand},
		{"helmet goes to head", 1003, player.EquipmentSlotHead},
		{"boots go to feet", 1004, player.EquipmentSlotFeet},
		{"libram goes to ranged", 1005, player.EquipmentSlotRanged},
		{"2h weapon goes to mainhand", 1007, player.EquipmentSlotMainHand},
		{"ring goes to finger1", 1009, player.EquipmentSlotFinger1},
		{"trinket goes to trinket1", 1010, player.EquipmentSlotTrinket1},
		{"shoulders go to shoulders", 1011, player.EquipmentSlotShoulder},
		{"cloak goes to back", 1012, player.EquipmentSlotBack},
		{"consumable returns -1", 1008, -1},
		{"unknown item returns -1", 9999, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
			tpl := basedata.GetInstance().LookupItem(tt.entry)
			slot := player.FindEquipSlotForItem(p, tpl, -1)
			assert.Equal(t, tt.expected, slot)
		})
	}
}

func TestFindEquipSlotForItemPrefersEmpty(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	tpl := basedata.GetInstance().LookupItem(1009) // ring

	// First ring should go to finger1
	slot := player.FindEquipSlotForItem(p, tpl, -1)
	assert.Equal(t, player.EquipmentSlotFinger1, slot)

	// Put a ring in finger1
	ring1 := player.NewItem(1009, p.GUID())
	p.Inventory.SetEquipment(player.EquipmentSlotFinger1, ring1)

	// Now should prefer finger2
	slot = player.FindEquipSlotForItem(p, tpl, -1)
	assert.Equal(t, player.EquipmentSlotFinger2, slot)

	// Fill finger2 too
	ring2 := player.NewItem(1009, p.GUID())
	p.Inventory.SetEquipment(player.EquipmentSlotFinger2, ring2)

	// Should return finger1 for swap
	slot = player.FindEquipSlotForItem(p, tpl, -1)
	assert.Equal(t, player.EquipmentSlotFinger1, slot)
}

func TestFindEquipSlotForItemWithRequestedSlot(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	tpl := basedata.GetInstance().LookupItem(1009) // ring

	// Request finger2 slot specifically - should work since ring can go in finger1 or finger2
	slot := player.FindEquipSlotForItem(p, tpl, player.EquipmentSlotFinger2)
	assert.Equal(t, player.EquipmentSlotFinger2, slot)

	// Request invalid slot (head for a ring) should return -1
	slot = player.FindEquipSlotForItem(p, tpl, player.EquipmentSlotHead)
	assert.Equal(t, -1, slot)
}

func TestCanEquipItem_LevelCheck(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(5, wow.ClassWarior, wow.RaceHuman) // level 5
	item := player.NewItem(1003, p.GUID())              // requires level 20
	tpl := basedata.GetInstance().LookupItem(1003)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultCantEquipLevel, result)
}

func TestCanEquipItem_LevelOK(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	item := player.NewItem(1003, p.GUID()) // requires level 20
	tpl := basedata.GetInstance().LookupItem(1003)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultOK, result)
}

func TestCanEquipItem_ClassCheck(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	// Paladin-only item (class mask = 2 = paladin)
	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman) // warrior
	item := player.NewItem(1005, p.GUID())
	tpl := basedata.GetInstance().LookupItem(1005)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultCantEquipClass, result)
}

func TestCanEquipItem_ClassOK(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	// Paladin-only item, paladin player
	p := testPlayer(80, wow.ClassPaladin, wow.RaceHuman)
	item := player.NewItem(1005, p.GUID())
	tpl := basedata.GetInstance().LookupItem(1005)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultOK, result)
}

func TestCanEquipItem_RaceCheck(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	// Orc-only item (race mask = 2 = orc)
	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman) // human
	item := player.NewItem(1006, p.GUID())
	tpl := basedata.GetInstance().LookupItem(1006)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultCantEquipRace, result)
}

func TestCanEquipItem_RaceOK(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	// Orc-only item, orc player
	p := testPlayer(80, wow.ClassWarior, wow.RaceOrc)
	item := player.NewItem(1006, p.GUID())
	tpl := basedata.GetInstance().LookupItem(1006)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultOK, result)
}

func TestCanEquipItem_CombatRestriction(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	p.InCombat = true // in combat

	item := player.NewItem(1001, p.GUID()) // weapon - can equip in combat
	tpl := basedata.GetInstance().LookupItem(1001)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultOK, result) // weapons can be swapped in combat
}

func TestCanEquipItem_CombatRestrictionArmor(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	p.InCombat = true // in combat

	item := player.NewItem(1003, p.GUID()) // helmet - cannot equip in combat
	tpl := basedata.GetInstance().LookupItem(1003)

	result := player.CanEquipItem(p, item, tpl, -1)
	assert.Equal(t, player.EquipResultCantEquipSlot, result)
}

func TestCanEquipItem_NilChecks(t *testing.T) {
	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	result := player.CanEquipItem(p, nil, nil, -1)
	assert.Equal(t, player.EquipResultWrongSlot, result)

	item := player.NewItem(1001, p.GUID())
	result = player.CanEquipItem(p, item, nil, -1)
	assert.Equal(t, player.EquipResultWrongSlot, result)
}

func TestEquipItemWithValidation_Success(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	item := player.NewItem(1001, p.GUID()) // sword
	tpl := basedata.GetInstance().LookupItem(1001)

	prev, result := p.EquipItemWithValidation(item, tpl, -1)
	assert.Equal(t, player.EquipResultOK, result)
	assert.Nil(t, prev)
	assert.Equal(t, player.EquipmentSlotMainHand, item.SlotIndex)
}

func TestEquipItemWithValidation_Swap(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Equip first sword
	sword1 := player.NewItem(1001, p.GUID())
	tpl := basedata.GetInstance().LookupItem(1001)
	_, result := p.EquipItemWithValidation(sword1, tpl, -1)
	require.Equal(t, player.EquipResultOK, result)

	// Equip second sword - should swap
	sword2 := player.NewItem(1001, p.GUID())
	prev, result := p.EquipItemWithValidation(sword2, tpl, -1)
	assert.Equal(t, player.EquipResultOK, result)
	assert.NotNil(t, prev)
	assert.Equal(t, sword1, prev)
	assert.Equal(t, player.EquipmentSlotMainHand, sword2.SlotIndex)
	// sword1 should be in the old slot of sword2 (which was -1, so removed)
}

func TestEquipItemWithValidation_WrongSlot(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	item := player.NewItem(1003, p.GUID()) // helmet
	tpl := basedata.GetInstance().LookupItem(1003)

	// Try to equip helmet in the ranged slot (wrong slot)
	prev, result := p.EquipItemWithValidation(item, tpl, player.EquipmentSlotRanged)
	assert.Equal(t, player.EquipResultWrongSlot, result)
	assert.Nil(t, prev)
}

func TestEquipItemWithValidation_LevelFail(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(5, wow.ClassWarior, wow.RaceHuman) // level 5
	item := player.NewItem(1003, p.GUID())              // requires level 20
	tpl := basedata.GetInstance().LookupItem(1003)

	prev, result := p.EquipItemWithValidation(item, tpl, -1)
	assert.Equal(t, player.EquipResultCantEquipLevel, result)
	assert.Nil(t, prev)
}

func TestUnequipItem(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	item := player.NewItem(1001, p.GUID())
	tpl := basedata.GetInstance().LookupItem(1001)

	// Equip
	_, result := p.EquipItemWithValidation(item, tpl, -1)
	require.Equal(t, player.EquipResultOK, result)
	assert.Equal(t, player.EquipmentSlotMainHand, item.SlotIndex)
	assert.NotNil(t, p.Inventory.GetEquipment(player.EquipmentSlotMainHand))

	// Unequip using the simple UnequipItem(slot)
	removed := p.UnequipItem(player.EquipmentSlotMainHand)
	assert.NotNil(t, removed)
	assert.Equal(t, item, removed)
	assert.Nil(t, p.Inventory.GetEquipment(player.EquipmentSlotMainHand))
	// Item is removed from slot (SlotIndex set to -1 by RemoveItem)
	assert.Equal(t, -1, removed.SlotIndex)
}

func TestGetFreeInventorySlot(t *testing.T) {
	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Should return first backpack slot
	slot := p.GetFreeInventorySlot()
	assert.Equal(t, player.InventorySlotItemStart, slot)

	// Fill all backpack slots
	for i := player.InventorySlotItemStart; i < player.InventorySlotItemEnd; i++ {
		item := player.NewItem(uint32(1000+i), p.GUID())
		p.Inventory.SetItem(i, item)
	}

	// Should return -1 when full
	slot = p.GetFreeInventorySlot()
	assert.Equal(t, -1, slot)
}

func TestHasItemCount(t *testing.T) {
	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Empty inventory
	count := p.HasItemCount(1001)
	assert.Equal(t, 0, count)

	// Add items
	item1 := player.NewItem(1001, p.GUID())
	item1.StackCount = 5
	p.Inventory.SetItem(player.InventorySlotItemStart, item1)

	item2 := player.NewItem(1001, p.GUID())
	item2.StackCount = 3
	p.Inventory.SetItem(player.InventorySlotItemStart+1, item2)

	count = p.HasItemCount(1001)
	assert.Equal(t, 8, count) // 5 + 3

	// Different item
	count = p.HasItemCount(1002)
	assert.Equal(t, 0, count)
}

func TestEquipResultString(t *testing.T) {
	tests := []struct {
		result   player.EquipResult
		expected string
	}{
		{player.EquipResultOK, "OK"},
		{player.EquipResultCantEquipLevel, "You must reach a higher level to use this item."},
		{player.EquipResultCantEquipClass, "Your class cannot use this item."},
		{player.EquipResultCantEquipRace, "Your race cannot use this item."},
		{player.EquipResultWrongSlot, "You can't equip that in that slot."},
		{player.EquipResultCantEquipSlot, "There is no equipment slot open for that item."},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.String())
		})
	}
}
