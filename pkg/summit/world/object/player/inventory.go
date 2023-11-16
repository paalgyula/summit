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

func (i *Inventory) AddEmpty() {
	i.InventorySlots = append(i.InventorySlots, &InventoryItem{
		DisplayInfoID: 0,
		InventoryType: 0,
		EnchantSlot:   0,
	})
}
