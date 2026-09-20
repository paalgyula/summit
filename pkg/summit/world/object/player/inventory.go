package player

import "github.com/paalgyula/summit/pkg/wow"

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
	// InventorySlotBuybackStart through InventorySlotBuybackEnd are buyback slots.
	InventorySlotBuybackStart = 19
	InventorySlotBuybackEnd   = 23

	// InventorySlotBagStart through InventorySlotBagEnd are the 4 bag slots.
	InventorySlotBagStart = 23
	InventorySlotBagEnd   = 27

	// InventorySlotItemStart through InventorySlotItemEnd are the backpack (main bag) slots.
	InventorySlotItemStart = 27
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

// ToCharacterEnum writes inventory item display data for the character enum packet.
//
//nolint:errcheck
func (inv *Inventory) ToCharacterEnum(w *wow.Packet) {
	for i := 0; i < EquipmentSlotEnd; i++ {
		var item *Item
		if i < len(inv.Slots) {
			item = inv.Slots[i]
		}
		if item == nil {
			w.Write(uint32(0)) // DisplayInfoID
			w.Write(wow.InventoryType(0))
			w.Write(uint32(0)) // EnchantSlot
		} else {
			w.Write(item.ItemEntry)
			w.Write(SlotToInventoryType(i))
			w.Write(item.GetEnchant(0))
		}
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
	case wow.InventoryTypeChest:
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
	case wow.InventoryTypeWeaponMainHand:
		return EquipmentSlotMainHand
	case wow.InventoryTypeWeaponOffHand:
		return EquipmentSlotOffHand
	case wow.InventoryTypeShield, wow.InventoryTypeHoldable:
		return EquipmentSlotOffHand
	case wow.InventoryTypeRanged, wow.InventoryTypeRelic:
		return EquipmentSlotRanged
	case wow.InventoryTypeTabard:
		return EquipmentSlotTabard
	default:
		return -1
	}
}
