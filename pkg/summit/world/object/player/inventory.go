package player

import "github.com/paalgyula/summit/pkg/wow"

const InventorySlotBagEnd = 23

type InventoryItem struct {
	DisplayInfoID uint32
	InventoryType wow.InventoryType
	EnchantSlot   uint32
}

type Inventory struct {
	InventorySlots []*InventoryItem
}
