package player

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
)

// Equipment slot indices (matching C++ EQUIPMENT_SLOT_* constants).
const (
	EquipmentSlotHead     = 0
	EquipmentSlotNeck     = 1
	EquipmentSlotShoulder = 2
	EquipmentSlotBody     = 3
	EquipmentSlotChest    = 4
	EquipmentSlotWaist    = 5
	EquipmentSlotLegs     = 6
	EquipmentSlotFeet     = 7
	EquipmentSlotWrists   = 8
	EquipmentSlotHands    = 9
	EquipmentSlotFinger1  = 10
	EquipmentSlotFinger2  = 11
	EquipmentSlotTrinket1 = 12
	EquipmentSlotTrinket2 = 13
	EquipmentSlotBack     = 14
	EquipmentSlotMainHand = 15
	EquipmentSlotOffHand  = 16
	EquipmentSlotRanged   = 17
	EquipmentSlotTabard   = 18
	EquipmentSlotEnd      = 19
)

// Bag/backpack slot range.
const (
	// InventorySlotBagStart through InventorySlotBagEnd are the 4 bag slots
	// where players equip containers (backpack, bags). Matches AC INVENTORY_SLOT_BAG_*.
	InventorySlotBagStart = 19
	InventorySlotBagEnd   = 23

	// InventorySlotItemStart through InventorySlotItemEnd are the backpack
	// (main bag) slots. Matches AC INVENTORY_SLOT_ITEM_*.
	InventorySlotItemStart = 23
	InventorySlotItemEnd   = 39

	// InventorySlotTotal is the total number of player inventory slots.
	InventorySlotTotal = 39
)

// Inventory holds all items for a player, indexed by slot.
type Inventory struct {
	// Slots holds items by slot index. nil means empty slot.
	Slots []*Item
}

// NewInventory creates an empty inventory with the standard number of slots.
func NewInventory() *Inventory {
	return &Inventory{
		Slots: make([]*Item, InventorySlotTotal),
	}
}

// GetItem returns the item in the given slot, or nil if empty.
func (inv *Inventory) GetItem(slot int) *Item {
	if slot < 0 || slot >= len(inv.Slots) {
		return nil
	}

	return inv.Slots[slot]
}

// SetItem places an item in the given slot. Returns the previously held item (or nil).
func (inv *Inventory) SetItem(slot int, item *Item) *Item {
	if slot < 0 || slot >= len(inv.Slots) {
		return nil
	}

	prev := inv.Slots[slot]
	inv.Slots[slot] = item

	if item != nil {
		item.SlotIndex = slot
	}

	return prev
}

// RemoveItem removes and returns the item from the given slot.
func (inv *Inventory) RemoveItem(slot int) *Item {
	if slot < 0 || slot >= len(inv.Slots) {
		return nil
	}

	item := inv.Slots[slot]
	inv.Slots[slot] = nil

	if item != nil {
		item.SlotIndex = -1
	}

	return item
}

// GetEquipment returns the item in the given equipment slot (0-18).
func (inv *Inventory) GetEquipment(slot int) *Item {
	if slot < 0 || slot >= EquipmentSlotEnd {
		return nil
	}

	return inv.Slots[slot]
}

// SetEquipment places an item in an equipment slot.
func (inv *Inventory) SetEquipment(slot int, item *Item) *Item {
	return inv.SetItem(slot, item)
}

// GetBackpackItem returns the item in a backpack slot (index 0-11).
func (inv *Inventory) GetBackpackItem(slot int) *Item {
	absSlot := InventorySlotItemStart + slot

	return inv.GetItem(absSlot)
}

// SetBackpackItem places an item in a backpack slot.
func (inv *Inventory) SetBackpackItem(slot int, item *Item) *Item {
	absSlot := InventorySlotItemStart + slot

	return inv.SetItem(absSlot, item)
}

// GetBagItem returns the item in a bag slot (0-3).
func (inv *Inventory) GetBagItem(slot int) *Item {
	absSlot := InventorySlotBagStart + slot

	return inv.GetItem(absSlot)
}

// SetBagItem places an item in a bag slot.
func (inv *Inventory) SetBagItem(slot int, item *Item) *Item {
	absSlot := InventorySlotBagStart + slot

	return inv.SetItem(absSlot, item)
}

// CountItems returns the number of non-nil items in the inventory.
func (inv *Inventory) CountItems() int {
	count := 0

	for _, item := range inv.Slots {
		if item != nil {
			count++
		}
	}

	return count
}

// FindEmptyBackpackSlot returns the index of the first empty backpack slot,
// or -1 if the backpack is full.
func (inv *Inventory) FindEmptyBackpackSlot() int {
	for i := InventorySlotItemStart; i < InventorySlotItemEnd; i++ {
		if inv.Slots[i] == nil {
			return i
		}
	}

	return -1
}

// AddItem places an item in the first available backpack slot.
// For stackable items, it first tries to stack with existing items of the same entry.
// Returns the slot where the item was placed, or -1 if inventory is full.
func (inv *Inventory) AddItem(item *Item) int {
	if item == nil {
		return -1
	}

	tpl := basedata.GetInstance().LookupItem(item.ItemEntry)
	maxStack := uint32(1)
	if tpl != nil {
		maxStack = uint32(tpl.GetMaxStackSize())
	}

	// Try to stack with existing items first
	if maxStack > 1 {
		for i := InventorySlotItemStart; i < InventorySlotItemEnd; i++ {
			existing := inv.Slots[i]
			if existing != nil && existing.ItemEntry == item.ItemEntry && existing.StackCount < maxStack {
				space := maxStack - existing.StackCount
				toAdd := item.StackCount
				if toAdd > space {
					toAdd = space
				}

				existing.StackCount += toAdd
				existing.UpdateFields()
				item.StackCount -= toAdd

				if item.StackCount == 0 {
					return i
				}
			}
		}
	}

	// Find empty slot
	slot := inv.FindEmptyBackpackSlot()
	if slot < 0 {
		return -1 // inventory full
	}

	item.SlotIndex = slot
	inv.Slots[slot] = item

	return slot
}

// CharEnumSlots is the number of item slots SMSG_CHAR_ENUM describes for a
// character: the 19 equipment slots + 4 bag slots = 23 total.
const CharEnumSlots = InventorySlotBagEnd // 19 equipment + 4 bag = 23

// ToCharacterEnum writes inventory item display data for the character enum
// packet: the display id and inventory type of the item template in every
// equipment and bag slot, and its enchant's visual.
//
//nolint:errcheck
func (inv *Inventory) ToCharacterEnum(w *wow.Packet) {
	for i := 0; i < CharEnumSlots; i++ {
		slot := i
		if i >= EquipmentSlotEnd {
			slot = InventorySlotBagStart + (i - EquipmentSlotEnd)
		}

		var item *Item
		if slot < len(inv.Slots) {
			item = inv.Slots[slot]
		}

		tpl := (*basedata.ItemTemplate)(nil)
		if item != nil {
			tpl = basedata.GetInstance().LookupItem(item.ItemEntry)
		}

		if item == nil || tpl == nil {
			w.Write(uint32(0)) // DisplayInfoID
			w.Write(wow.InventoryType(0))
			w.Write(uint32(0)) // EnchantSlot

			continue
		}

		w.Write(tpl.DisplayID)
		w.Write(tpl.InventoryType)
		w.Write(item.GetEnchant(0))
	}
}

// SlotToInventoryType maps an equipment slot index to the corresponding InventoryType.
func SlotToInventoryType(slot int) wow.InventoryType {
	switch slot {
	case EquipmentSlotHead:
		return wow.InventoryTypeHead
	case EquipmentSlotNeck:
		return wow.InventoryTypeNeck
	case EquipmentSlotShoulder:
		return wow.InventoryTypeShoulders
	case EquipmentSlotBody:
		return wow.InventoryTypeBody
	case EquipmentSlotChest:
		return wow.InventoryTypeChest
	case EquipmentSlotWaist:
		return wow.InventoryTypeWaist
	case EquipmentSlotLegs:
		return wow.InventoryTypeLegs
	case EquipmentSlotFeet:
		return wow.InventoryTypeFeet
	case EquipmentSlotWrists:
		return wow.InventoryTypeWrists
	case EquipmentSlotHands:
		return wow.InventoryTypeHands
	case EquipmentSlotFinger1, EquipmentSlotFinger2:
		return wow.InventoryTypeFinger
	case EquipmentSlotTrinket1, EquipmentSlotTrinket2:
		return wow.InventoryTypeTrinket
	case EquipmentSlotBack:
		return wow.InventoryTypeCloak
	case EquipmentSlotMainHand:
		return wow.InventoryTypeWeaponMainHand
	case EquipmentSlotOffHand:
		return wow.InventoryTypeWeaponOffHand
	case EquipmentSlotRanged:
		return wow.InventoryTypeRanged
	case EquipmentSlotTabard:
		return wow.InventoryTypeTabard
	default:
		return wow.InventoryType(0)
	}
}

// FindEquipSlot determines the correct equipment slot for an item based on its inventory type.
// Returns -1 if the item cannot be equipped.
func FindEquipSlot(inventoryType wow.InventoryType) int {
	switch inventoryType {
	case wow.InventoryTypeHead:
		return EquipmentSlotHead
	case wow.InventoryTypeNeck:
		return EquipmentSlotNeck
	case wow.InventoryTypeShoulders:
		return EquipmentSlotShoulder
	case wow.InventoryTypeBody:
		return EquipmentSlotBody
	case wow.InventoryTypeChest, wow.InventoryTypeRobe:
		return EquipmentSlotChest
	case wow.InventoryTypeWaist:
		return EquipmentSlotWaist
	case wow.InventoryTypeLegs:
		return EquipmentSlotLegs
	case wow.InventoryTypeFeet:
		return EquipmentSlotFeet
	case wow.InventoryTypeWrists:
		return EquipmentSlotWrists
	case wow.InventoryTypeHands:
		return EquipmentSlotHands
	case wow.InventoryTypeFinger:
		return EquipmentSlotFinger1 // TODO: check if finger2 is empty
	case wow.InventoryTypeTrinket:
		return EquipmentSlotTrinket1 // TODO: check if trinket2 is empty
	case wow.InventoryTypeCloak:
		return EquipmentSlotBack
	case wow.InventoryTypeWeaponMainHand, wow.InventoryTypeWeapon, wow.InventoryType2hweapon:
		return EquipmentSlotMainHand
	case wow.InventoryTypeWeaponOffHand:
		return EquipmentSlotOffHand
	case wow.InventoryTypeShield, wow.InventoryTypeHoldable:
		return EquipmentSlotOffHand
	case wow.InventoryTypeRanged, wow.InventoryTypeRangedRight, wow.InventoryTypeThrown, wow.InventoryTypeRelic:
		return EquipmentSlotRanged
	case wow.InventoryTypeTabard:
		return EquipmentSlotTabard
	default:
		return -1
	}
}
