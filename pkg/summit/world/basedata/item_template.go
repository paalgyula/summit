package basedata

import "github.com/paalgyula/summit/pkg/wow"

// ItemStat represents a stat modifier on an item.
type ItemStat struct {
	Type  uint32 // ITEM_STAT_TYPE_*
	Value int32
}

// ItemDamage represents a damage range for an item.
type ItemDamage struct {
	Type uint32  // from Resistances.dbc
	Min  float32
	Max  float32
}

// ItemSpell represents a spell triggered by an item.
type ItemSpell struct {
	SpellID          int32
	Trigger          uint32 // 0=on_use, 1=on_equip, 2=on_hit, 3=on_store
	Charges          int32
	PPMRate          float32
	Cooldown         int32
	Category         uint32
	CategoryCooldown int32
}

// ItemSocket represents a gem socket on an item.
type ItemSocket struct {
	Color   uint32 // SOCKET_COLOR_*
	Content uint32 // required content type
}

// ItemTemplate is the full item_template data the server needs for
// equip validation, stat application, loot, and vendor operations.
type ItemTemplate struct {
	Entry            uint32
	Class            uint32 // ITEM_CLASS_*
	SubClass         uint32 // ITEM_SUBCLASS_* (depends on Class)
	DisplayID        uint32
	Name             string
	Quality          uint32 // item quality (0=poor..6=legendary)
	InventoryType    wow.InventoryType // INVTYPE_*

	// Requirements
	AllowableClass     int32  // class mask, -1 = all
	AllowableRace      int32  // race mask, -1 = all
	RequiredLevel      uint32
	RequiredSkill      uint32
	RequiredSkillRank  uint32
	RequiredHonorRank  uint32
	RequiredCityRank   uint32
	RequiredRepFaction uint32
	RequiredRepRank    uint32

	// Stats
	ItemLevel   uint32
	StatsCount  uint32
	Stats       [10]ItemStat
	Damage      [2]ItemDamage
	Armor       uint32
	HolyRes     int32
	FireRes     int32
	NatureRes   int32
	FrostRes    int32
	ShadowRes   int32
	ArcaneRes   int32
	Delay       uint32
	AmmoType    uint32
	RangedModRange float32

	// Spells and sockets
	Spells      [5]ItemSpell
	Sockets     [3]ItemSocket
	SocketBonus uint32

	// Pricing
	BuyPrice  int32
	SellPrice uint32
	BuyCount  uint32

	// Durability and limits
	MaxDurability  uint32
	MaxCount       int32 // -1 = unlimited
	Stackable      int32 // 0 = not stackable, -1 = currency
	ContainerSlots uint32

	// Flags and bonding
	Flags       uint32 // ITEM_FLAG_*
	Bonding     uint32 // 0=no_bind, 1=bind_on_pickup, 2=bind_on_equip, 3=bind_on_use, 4=bind_on_quest
	SheatheType uint32
	Material    int32
	BagFamily   uint32

	// Misc
	Block            uint32
	ItemSet          uint32
	LockID           uint32
	RandomProperty   int32
	RandomSuffix     int32
	PageText         uint32
	LanguageID       uint32
	PageMaterial     uint32
	StartQuest       uint32
	DisenchantID     uint32
	FoodType         uint32
	Duration         uint32
	ItemLimitCategory uint32

	// Loot
	MinMoneyLoot uint32
	MaxMoneyLoot uint32
}

// GetMaxStackSize returns the maximum stack count for this item.
func (t *ItemTemplate) GetMaxStackSize() int32 {
	if t.Stackable <= 0 {
		if t.Stackable == 0 {
			return 1
		}
		return -1 // currency
	}
	return t.Stackable
}

// IsWeapon returns true if the item is a weapon class.
func (t *ItemTemplate) IsWeapon() bool {
	return t.Class == 2 // ITEM_CLASS_WEAPON
}

// IsArmor returns true if the item is armor.
func (t *ItemTemplate) IsArmor() bool {
	return t.Class == 4 // ITEM_CLASS_ARMOR
}

// IsEquippable returns true if the item has an inventory type that allows equipping.
func (t *ItemTemplate) IsEquippable() bool {
	return t.InventoryType != 0
}

// CanChangeEquipStateInCombat returns true if the item type can be swapped during combat.
func (t *ItemTemplate) CanChangeEquipStateInCombat() bool {
	switch t.InventoryType {
	case 28, 14, 23: // INVTYPE_RELIC, INVTYPE_SHIELD, INVTYPE_HOLDABLE
		return true
	}

	switch t.Class {
	case 2, 6: // ITEM_CLASS_WEAPON, ITEM_CLASS_PROJECTILE
		return true
	}

	return false
}
