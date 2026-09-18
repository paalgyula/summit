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

	return &Item{
		Object:       obj,
		Owner:        owner,
		Contained:    owner,
		ItemEntry:    entry,
		StackCount:   1,
		SlotIndex:    -1,
		SpellCharges: [5]int32{-1, -1, -1, -1, -1},
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
