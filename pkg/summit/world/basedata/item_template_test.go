package basedata_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

func testStore() *basedata.Store {
	return &basedata.Store{
		Items: []*basedata.ItemTemplate{
			{Entry: 1001, Class: 2, SubClass: 7, DisplayID: 5000, Name: "Sword",
				Quality: 2, InventoryType: wow.InventoryTypeWeaponMainHand, RequiredLevel: 10, AllowableClass: -1, AllowableRace: -1},
			{Entry: 1002, Class: 4, SubClass: 6, DisplayID: 5001, Name: "Shield",
				Quality: 2, InventoryType: wow.InventoryTypeShield, RequiredLevel: 10, AllowableClass: -1, AllowableRace: -1},
			{Entry: 1003, Class: 4, SubClass: 4, DisplayID: 5002, Name: "Plate Helm",
				Quality: 3, InventoryType: wow.InventoryTypeHead, RequiredLevel: 20, AllowableClass: -1, AllowableRace: -1},
		},
	}
}

func TestLookupItem_Found(t *testing.T) {
	s := testStore()
	s.Index()

	tpl := s.LookupItem(1001)
	assert.NotNil(t, tpl)
	assert.Equal(t, "Sword", tpl.Name)
	assert.EqualValues(t, 21, tpl.InventoryType)
}

func TestLookupItem_NotFound(t *testing.T) {
	s := testStore()
	s.Index()

	tpl := s.LookupItem(9999)
	assert.Nil(t, tpl)
}

func TestLookupItem_NilStore(t *testing.T) {
	var s *basedata.Store
	tpl := s.LookupItem(1001)
	assert.Nil(t, tpl)
}

func TestGetItemInventoryType_Found(t *testing.T) {
	basedata.SetInstance(testStore())
	defer basedata.SetInstance(nil)

	invType := basedata.GetItemInventoryType(1001)
	assert.Equal(t, wow.InventoryTypeWeaponMainHand, invType)
}

func TestGetItemInventoryType_NotFound(t *testing.T) {
	basedata.SetInstance(testStore())
	defer basedata.SetInstance(nil)

	invType := basedata.GetItemInventoryType(9999)
	assert.Equal(t, wow.InventoryTypeNonEquip, invType)
}

func TestGetItemTemplate_Found(t *testing.T) {
	basedata.SetInstance(testStore())
	defer basedata.SetInstance(nil)

	tpl := basedata.GetItemTemplate(1002)
	assert.NotNil(t, tpl)
	assert.Equal(t, "Shield", tpl.Name)
}

func TestGetItemTemplate_NotFound(t *testing.T) {
	basedata.SetInstance(testStore())
	defer basedata.SetInstance(nil)

	tpl := basedata.GetItemTemplate(9999)
	assert.Nil(t, tpl)
}

func TestItemTemplate_GetMaxStackSize(t *testing.T) {
	tests := []struct {
		name     string
		stackable int32
		expected int32
	}{
		{"normal stack", 5, 5},
		{"not stackable", 0, 1},
		{"currency", -1, -1},
		{"unlimited", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := &basedata.ItemTemplate{Stackable: tt.stackable}
			assert.Equal(t, tt.expected, tpl.GetMaxStackSize())
		})
	}
}

func TestItemTemplate_IsWeapon(t *testing.T) {
	weapon := &basedata.ItemTemplate{Class: 2}
	armor := &basedata.ItemTemplate{Class: 4}
	consumable := &basedata.ItemTemplate{Class: 0}

	assert.True(t, weapon.IsWeapon())
	assert.False(t, armor.IsWeapon())
	assert.False(t, consumable.IsWeapon())
}

func TestItemTemplate_IsArmor(t *testing.T) {
	weapon := &basedata.ItemTemplate{Class: 2}
	armor := &basedata.ItemTemplate{Class: 4}
	consumable := &basedata.ItemTemplate{Class: 0}

	assert.False(t, weapon.IsArmor())
	assert.True(t, armor.IsArmor())
	assert.False(t, consumable.IsArmor())
}

func TestItemTemplate_IsEquippable(t *testing.T) {
	weapon := &basedata.ItemTemplate{InventoryType: wow.InventoryTypeWeaponMainHand}
	nonEquip := &basedata.ItemTemplate{InventoryType: wow.InventoryTypeNonEquip}

	assert.True(t, weapon.IsEquippable())
	assert.False(t, nonEquip.IsEquippable())
}

func TestItemTemplate_CanChangeEquipStateInCombat(t *testing.T) {
	tests := []struct {
		name     string
		class    uint32
		invType  wow.InventoryType
		expected bool
	}{
		{"weapon can swap", 2, wow.InventoryTypeWeaponMainHand, true},
		{"projectile can swap", 6, wow.InventoryTypeNonEquip, true},
		{"relic can swap", 4, wow.InventoryTypeRelic, true},
		{"shield can swap", 4, wow.InventoryTypeShield, true},
		{"holdable can swap", 4, wow.InventoryTypeHoldable, true},
		{"helmet cannot swap", 4, wow.InventoryTypeHead, false},
		{"chest cannot swap", 4, wow.InventoryTypeChest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := &basedata.ItemTemplate{Class: tt.class, InventoryType: tt.invType}
			assert.Equal(t, tt.expected, tpl.CanChangeEquipStateInCombat())
		})
	}
}
