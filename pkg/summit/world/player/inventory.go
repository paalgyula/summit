package player

const InventorySlotBagEnd = 23

type InventoryItem struct {
	DisplayInfoID uint32
	InventoryType uint8
}

type Inventory struct {
	InventorySlots [InventorySlotBagEnd]InventoryItem
}
