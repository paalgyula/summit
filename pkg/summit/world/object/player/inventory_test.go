package player_test

import (
	"encoding/binary"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The human warrior's CharStartOutfit row, with Item.dbc templates for it.
func humanWarriorBaseData() *basedata.Store {
	return &basedata.Store{
		Items: []*basedata.ItemTemplate{
			{Entry: 38, DisplayID: 9891, InventoryType: wow.InventoryTypeBody},
			{Entry: 39, DisplayID: 9892, InventoryType: wow.InventoryTypeLegs},
			{Entry: 40, DisplayID: 10141, InventoryType: wow.InventoryTypeFeet},
			{Entry: 25, DisplayID: 1542, InventoryType: wow.InventoryTypeWeaponMainHand, SheatheType: 3},
			{Entry: 2362, DisplayID: 18730, InventoryType: wow.InventoryTypeShield, SheatheType: 4},
			{Entry: 117, DisplayID: 2473, InventoryType: wow.InventoryTypeNonEquip},
		},
	}
}

func TestInitInventoryPlacesOutfitBySlot(t *testing.T) {
	basedata.SetInstance(humanWarriorBaseData())
	defer basedata.SetInstance(nil)

	outfit := []*basedata.InventorySlot{
		{ItemID: 38, DisplayItemID: 9891, InventoryType: wow.InventoryTypeBody},
		{ItemID: 39, DisplayItemID: 9892, InventoryType: wow.InventoryTypeLegs},
		{ItemID: 40, DisplayItemID: 10141, InventoryType: wow.InventoryTypeFeet},
		{ItemID: 25, DisplayItemID: 1542, InventoryType: wow.InventoryTypeWeaponMainHand},
		{ItemID: 2362, DisplayItemID: 18730, InventoryType: wow.InventoryTypeShield},
		{ItemID: 117, DisplayItemID: 2473, InventoryType: wow.InventoryTypeNonEquip},
		{ItemID: 0},
	}

	p := &player.Player{ID: 7}
	p.InitInventory(outfit)

	assert.EqualValues(t, 38, p.Inventory.GetEquipment(player.EquipmentSlotBody).ItemEntry)
	assert.EqualValues(t, 39, p.Inventory.GetEquipment(player.EquipmentSlotLegs).ItemEntry)
	assert.EqualValues(t, 40, p.Inventory.GetEquipment(player.EquipmentSlotFeet).ItemEntry)
	assert.EqualValues(t, 25, p.Inventory.GetEquipment(player.EquipmentSlotMainHand).ItemEntry)
	assert.EqualValues(t, 2362, p.Inventory.GetEquipment(player.EquipmentSlotOffHand).ItemEntry)
	assert.Nil(t, p.Inventory.GetEquipment(player.EquipmentSlotHead))
	// The bread goes to the first backpack slot
	assert.EqualValues(t, 117, p.Inventory.GetBackpackItem(0).ItemEntry)
	assert.Equal(t, 6, p.Inventory.CountItems())
}

func TestCharacterEnumWritesDisplayIDs(t *testing.T) {
	basedata.SetInstance(humanWarriorBaseData())
	defer basedata.SetInstance(nil)

	p := &player.Player{ID: 7}
	p.InitInventory([]*basedata.InventorySlot{
		{ItemID: 25, InventoryType: wow.InventoryTypeWeaponMainHand},
		{ItemID: 40, InventoryType: wow.InventoryTypeFeet},
	})

	w := wow.NewPacket(0)
	p.Inventory.ToCharacterEnum(w)
	data := w.Bytes()

	// 19 equipment slots + 4 bags, 9 bytes each: display id, inventory type, enchant
	assert.Len(t, data, player.CharEnumSlots*9)
	assert.Equal(t, 23, player.CharEnumSlots)

	slot := func(i int) (uint32, uint8, uint32) {
		o := i * 9
		return binary.LittleEndian.Uint32(data[o:]), data[o+4], binary.LittleEndian.Uint32(data[o+5:])
	}
	display, invType, enchant := slot(player.EquipmentSlotMainHand)
	assert.EqualValues(t, 1542, display)
	assert.EqualValues(t, wow.InventoryTypeWeaponMainHand, invType)
	assert.EqualValues(t, 0, enchant)

	display, invType, _ = slot(player.EquipmentSlotFeet)
	assert.EqualValues(t, 10141, display)
	assert.EqualValues(t, wow.InventoryTypeFeet, invType)

	display, invType, _ = slot(player.EquipmentSlotHead)
	assert.EqualValues(t, 0, display)
	assert.EqualValues(t, 0, invType)
}

func TestAddItemStacksIntoExisting(t *testing.T) {
	basedata.SetInstance(&basedata.Store{
		Items: []*basedata.ItemTemplate{
			{Entry: 2589, Stackable: 20},
		},
	})
	defer basedata.SetInstance(nil)

	p := &player.Player{ID: 7}
	p.Inventory = player.NewInventory()

	first := player.NewItem(2589, p.GUID())
	first.StackCount = 5
	res := p.Inventory.AddItem(first)
	require.True(t, res.Placed)
	require.Empty(t, res.StackedInto)

	// A second drop merges into the first stack without opening a new slot.
	second := player.NewItem(2589, p.GUID())
	second.StackCount = 10
	res = p.Inventory.AddItem(second)
	assert.False(t, res.Placed)
	require.Len(t, res.StackedInto, 1)
	assert.EqualValues(t, 15, res.StackedInto[0].StackCount)

	// 15 more: 5 fill the first stack to its max, the remaining 10 open a slot.
	third := player.NewItem(2589, p.GUID())
	third.StackCount = 15
	res = p.Inventory.AddItem(third)
	require.Len(t, res.StackedInto, 1)
	assert.EqualValues(t, 20, res.StackedInto[0].StackCount)
	assert.True(t, res.Placed)
	assert.EqualValues(t, 10, p.Inventory.GetItem(res.Slot).StackCount)
}

func TestInspectSummitDat(t *testing.T) {
	bd, err := basedata.LoadFromFile("../../../../../summit.dat")
	if err != nil {
		t.Skip("summit.dat not found")
	}
	t.Logf("Items loaded: %d, CreateInfo: %d", len(bd.Items), len(bd.PlayerCreateInfo))
	for _, pci := range bd.PlayerCreateInfo {
		if pci.Race == wow.RaceTauren {
			t.Logf("Tauren Class %d Gender %d:", pci.Class, pci.Gender)
			for i, slot := range pci.Inventory {
				if slot != nil && slot.ItemID > 0 {
					t.Logf("  Slot %d: ItemID=%d, DisplayID=%d, InvType=%d",
						i, slot.ItemID, slot.DisplayItemID, slot.InventoryType)
				}
			}
		}
	}
}
