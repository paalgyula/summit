package player

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

// Item represents a game item in a player's inventory.
type Item struct {
	*object.Object

	// Owner is the GUID of the player who owns this item.
	Owner wow.GUID

	// Contained is the GUID of the container (bag) holding this item.
	Contained wow.GUID

	// ItemEntry is the item template ID from item_template (DBC/DB).
	ItemEntry uint32

	// StackCount is how many of this item are stacked.
	StackCount uint32

	// Durability current durability.
	Durability uint32
	// MaxDurability maximum durability.
	MaxDurability uint32

	// Enchantments stores enchantment IDs per slot (12 enchant slots).
	Enchantments [12]uint32

	// ItemFlags ( ITEM_FLAG_UNKNOWN_0, ITEM_FLAG_CONJURED, etc).
	ItemFlags uint32

	// PropertySeed for random properties.
	PropertySeed uint32
	// RandomPropertiesID for random enchantments.
	RandomPropertiesID uint32

	// SpellCharges remaining charges for consumables.
	SpellCharges [5]int32

	// Duration remaining duration in seconds (0 = permanent).
	Duration uint32

	// SlotIndex is the inventory slot this item occupies.
	SlotIndex int

	// PlayTime is the total time the item has existed (for statistics).
	PlayTime uint32
}

// NewItem creates a new item with the given entry and owner GUID.
func NewItem(entry uint32, owner wow.GUID) *Item {
	obj := object.NewObject()
	obj.SetObjectType(obj.ObjectType() | wow.TypeMaskItem)

	item := &Item{
		Object:       obj,
		Owner:        owner,
		Contained:    owner,
		ItemEntry:    entry,
		StackCount:   1,
		SlotIndex:    -1,
		SpellCharges: [5]int32{-1, -1, -1, -1, -1},
	}

	item.initUpdateFields()

	return item
}

// initUpdateFields sets up the item's update field values.
func (i *Item) initUpdateFields() {
	i.Object.InitValues(int(object.ItemEnd))

	// Object fields
	guid := i.Object.GUID()
	i.Object.SetUInt32Value(object.ObjectFieldGuid, uint32(guid))
	i.Object.SetUInt32Value(object.ObjectFieldGuid+1, uint32(uint64(guid)>>32))
	i.Object.SetUInt32Value(object.ObjectFieldType, uint32(wow.TypeIDItem))
	i.Object.SetUInt32Value(object.ObjectFieldEntry, i.ItemEntry)
	i.Object.SetFloatValue(object.ObjectFieldScaleX, 1.0)

	// Item fields
	i.Object.SetUInt32Value(object.ItemFieldOwner, uint32(i.Owner))
	i.Object.SetUInt32Value(object.ItemFieldOwner+1, uint32(uint64(i.Owner)>>32))
	i.Object.SetUInt32Value(object.ItemFieldContained, uint32(i.Contained))
	i.Object.SetUInt32Value(object.ItemFieldContained+1, uint32(uint64(i.Contained)>>32))
	i.Object.SetUInt32Value(object.ItemFieldStackCount, i.StackCount)
	i.Object.SetUInt32Value(object.ItemFieldFlags, i.ItemFlags)
	i.Object.SetUInt32Value(object.ItemFieldDurability, i.Durability)
	i.Object.SetUInt32Value(object.ItemFieldMaxdurability, i.MaxDurability)
	i.Object.SetUInt32Value(object.ItemFieldPropertySeed, i.PropertySeed)
	i.Object.SetUInt32Value(object.ItemFieldRandomPropertiesId, i.RandomPropertiesID)
	i.Object.SetUInt32Value(object.ItemFieldCreatePlayedTime, i.PlayTime)

	// Set spell charges
	for j := 0; j < 5; j++ {
		i.Object.SetInt32Value(object.UpdateField(int(object.ItemFieldSpellCharges)+j), i.SpellCharges[j])
	}

	// Set enchantments (each enchant has 3 fields: spell_id + duration + charges)
	for j := 0; j < 12; j++ {
		enchantField := object.UpdateField(int(object.ItemFieldEnchantment1_1) + j*3)
		i.Object.SetUInt32Value(enchantField, i.Enchantments[j])
	}
}

// UpdateFields updates the item's update field values from its current state.
func (i *Item) UpdateFields() {
	i.Object.SetUInt32Value(object.ItemFieldStackCount, i.StackCount)
	i.Object.SetUInt32Value(object.ItemFieldFlags, i.ItemFlags)
	i.Object.SetUInt32Value(object.ItemFieldDurability, i.Durability)
	i.Object.SetUInt32Value(object.ItemFieldMaxdurability, i.MaxDurability)

	for j := 0; j < 5; j++ {
		i.Object.SetInt32Value(object.UpdateField(int(object.ItemFieldSpellCharges)+j), i.SpellCharges[j])
	}

	for j := 0; j < 12; j++ {
		enchantField := object.UpdateField(int(object.ItemFieldEnchantment1_1) + j*3)
		i.Object.SetUInt32Value(enchantField, i.Enchantments[j])
	}
}

// GUID returns the item's GUID (set externally via Object.guid).
func (i *Item) GUID() wow.GUID {
	return i.Object.GUID()
}

// GetEnchant returns the enchantment ID for the given enchant slot.
func (i *Item) GetEnchant(slot int) uint32 {
	if slot < 0 || slot >= len(i.Enchantments) {
		return 0
	}

	return i.Enchantments[slot]
}

// SetEnchant sets the enchantment ID for the given enchant slot.
func (i *Item) SetEnchant(slot int, enchantID uint32) {
	if slot < 0 || slot >= len(i.Enchantments) {
		return
	}

	i.Enchantments[slot] = enchantID
}

// IsStackable returns true if the item can stack with others.
func (i *Item) IsStackable() bool {
	return i.StackCount > 1
}

// HasEnchant returns true if any enchantment slot is non-zero.
func (i *Item) HasEnchant() bool {
	for _, e := range i.Enchantments {
		if e != 0 {
			return true
		}
	}

	return false
}
