package player

const InventorySlotBagEnd = 23

type InventoryItem struct {
	DisplayInfoID uint32
	InventoryType uint8
	EnchantSlot   uint32
}

type Inventory struct {
	InventorySlots []*InventoryItem
}
