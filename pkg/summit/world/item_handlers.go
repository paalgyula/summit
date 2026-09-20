package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleAutoEquipItem handles CMSG_AUTOEQUIP_ITEM - right-click equip.
func (gc *WorldSession) HandleAutoEquipItem(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	// Read source bag and slot
	var srcBag int8
	_ = reader.Read(&srcBag)

	var srcSlot uint8
	_ = reader.Read(&srcSlot)

	gc.log.Debug().
		Int8("srcBag", srcBag).
		Uint8("srcSlot", srcSlot).
		Msg("auto equip item")

	// Find the item
	var item *player.Item
	if srcBag == 0 || srcBag == -1 {
		// From inventory/backpack
		item = gc.player.Inventory.GetItem(int(srcSlot))
	} else {
		// From a bag (TODO: implement bag system)
		gc.log.Warn().Msg("bag equip not implemented yet")
		return
	}

	if item == nil {
		gc.log.Warn().Uint8("slot", srcSlot).Msg("no item in slot")
		return
	}

	// Equip the item
	prev := gc.player.EquipItem(item)

	// If there was a previous item, put it in the source slot
	if prev != nil {
		gc.player.Inventory.SetItem(int(srcSlot), prev)
		prev.UpdateFields()
	}

	// Update item fields
	item.UpdateFields()

	// Send inventory update to the player
	gc.sendInventoryUpdate()

	gc.log.Info().
		Uint32("entry", item.ItemEntry).
		Int("slot", item.SlotIndex).
		Msg("item equipped")
}

// HandleAutoEquipItemSlot handles CMSG_AUTOEQUIP_ITEM_SLOT - equip to specific slot.
func (gc *WorldSession) HandleAutoEquipItemSlot(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	// Read item GUID and destination slot
	var itemGUID uint64
	_ = reader.Read(&itemGUID)

	var destSlot uint8
	_ = reader.Read(&destSlot)

	gc.log.Debug().
		Uint64("itemGUID", itemGUID).
		Uint8("destSlot", destSlot).
		Msg("auto equip item slot")

	// Validate destination is an equipment slot
	if int(destSlot) >= player.EquipmentSlotEnd {
		gc.log.Warn().Uint8("slot", destSlot).Msg("invalid equipment slot")
		return
	}

	// Find the item in inventory
	var srcSlot int = -1
	for i := 0; i < player.InventorySlotTotal; i++ {
		item := gc.player.Inventory.GetItem(i)
		if item != nil && uint64(item.GUID()) == itemGUID {
			srcSlot = i
			break
		}
	}

	if srcSlot < 0 {
		gc.log.Warn().Uint64("guid", itemGUID).Msg("item not found")
		return
	}

	item := gc.player.Inventory.GetItem(srcSlot)
	if item == nil {
		return
	}

	// Swap with existing item in destination
	existing := gc.player.Inventory.GetEquipment(int(destSlot))
	gc.player.Inventory.SetItem(srcSlot, existing)
	gc.player.Inventory.SetEquipment(int(destSlot), item)

	// Update item fields
	item.SlotIndex = int(destSlot)
	item.UpdateFields()

	if existing != nil {
		existing.SlotIndex = srcSlot
		existing.UpdateFields()
	}

	// Update player inventory fields
	gc.player.UpdateInventoryFields()

	// Send inventory update
	gc.sendInventoryUpdate()
}

// HandleSwapInvItem handles CMSG_SWAP_INV_ITEM - swap two inventory slots.
func (gc *WorldSession) HandleSwapInvItem(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var srcSlot uint8
	_ = reader.Read(&srcSlot)

	var destSlot uint8
	_ = reader.Read(&destSlot)

	gc.log.Debug().
		Uint8("src", srcSlot).
		Uint8("dest", destSlot).
		Msg("swap inv item")

	srcItem := gc.player.Inventory.GetItem(int(srcSlot))
	destItem := gc.player.Inventory.GetItem(int(destSlot))

	gc.player.Inventory.SetItem(int(srcSlot), destItem)
	gc.player.Inventory.SetItem(int(destSlot), srcItem)

	if srcItem != nil {
		srcItem.SlotIndex = int(destSlot)
		srcItem.UpdateFields()
	}

	if destItem != nil {
		destItem.SlotIndex = int(srcSlot)
		destItem.UpdateFields()
	}

	gc.player.UpdateInventoryFields()
	gc.sendInventoryUpdate()
}

// HandleSwapItem handles CMSG_SWAP_ITEM - swap items between bags.
func (gc *WorldSession) HandleSwapItem(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var srcBag int8
	_ = reader.Read(&srcBag)

	var srcSlot uint8
	_ = reader.Read(&srcSlot)

	var destBag int8
	_ = reader.Read(&destBag)

	var destSlot uint8
	_ = reader.Read(&destSlot)

	gc.log.Debug().
		Int8("srcBag", srcBag).
		Uint8("srcSlot", srcSlot).
		Int8("destBag", destBag).
		Uint8("destSlot", destSlot).
		Msg("swap item")

	// TODO: Implement bag-to-bag swapping
	// For now, only handle same-bag swaps
	if srcBag != destBag {
		gc.log.Warn().Msg("cross-bag swap not implemented")
		return
	}

	srcItem := gc.player.Inventory.GetItem(int(srcSlot))
	destItem := gc.player.Inventory.GetItem(int(destSlot))

	gc.player.Inventory.SetItem(int(srcSlot), destItem)
	gc.player.Inventory.SetItem(int(destSlot), srcItem)

	if srcItem != nil {
		srcItem.SlotIndex = int(destSlot)
		srcItem.UpdateFields()
	}

	if destItem != nil {
		destItem.SlotIndex = int(srcSlot)
		destItem.UpdateFields()
	}

	gc.player.UpdateInventoryFields()
	gc.sendInventoryUpdate()
}

// HandleDestroyItem handles CMSG_DESTROYITEM - destroy an item.
func (gc *WorldSession) HandleDestroyItem(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var bag int8
	_ = reader.Read(&bag)

	var slot uint8
	_ = reader.Read(&slot)

	var count uint8
	_ = reader.Read(&count)

	gc.log.Debug().
		Int8("bag", bag).
		Uint8("slot", slot).
		Uint8("count", count).
		Msg("destroy item")

	item := gc.player.Inventory.GetItem(int(slot))
	if item == nil {
		return
	}

	// Remove item
	gc.player.Inventory.RemoveItem(int(slot))
	gc.player.UpdateInventoryFields()

	// Send destroy object to client
	gc.sendDestroyObject(item.GUID())

	gc.log.Info().
		Uint32("entry", item.ItemEntry).
		Msg("item destroyed")
}

// HandleItemQuerySingle handles CMSG_ITEM_QUERY_SINGLE - request item template data.
func (gc *WorldSession) HandleItemQuerySingle(data wow.PacketData) {
	reader := wow.NewPacketReader(data)

	var entry uint32
	_ = reader.Read(&entry)

	gc.log.Debug().Uint32("entry", entry).Msg("item query")

	// Send item query response
	gc.sendItemQueryResponse(entry)
}

// sendItemQueryResponse sends SMSG_ITEM_QUERY_SINGLE_RESPONSE.
func (gc *WorldSession) sendItemQueryResponse(entry uint32) {
	pkt := wow.NewPacket(wow.ServerItemQuerySingleResponse)

	// Write item entry (with error bit cleared)
	_ = pkt.Write(entry)

	// Write item class (2 = ITEM_CLASS_WEAPON, 4 = ITEM_CLASS_ARMOR)
	// TODO: Look up from item_template database
	itemClass := uint32(2) // Weapon
	_ = pkt.Write(itemClass)

	// Write subclass
	_ = pkt.Write(uint32(0))

	// Write name (null-terminated string)
	pkt.WriteString(fmt.Sprintf("Item %d", entry))

	// Write remaining fields (simplified - would need full item_template data)
	for i := 0; i < 32; i++ {
		_ = pkt.Write(uint32(0)) // padding
	}

	// Write displayid
	_ = pkt.Write(uint32(0))

	// Write quality
	_ = pkt.Write(uint32(0)) // Common

	// Write flags
	_ = pkt.Write(uint32(0))

	// Write buy price
	_ = pkt.Write(uint32(0))

	// Write sell price
	_ = pkt.Write(uint32(0))

	// Write inventory type
	_ = pkt.Write(uint32(0))

	// Write allowable class
	_ = pkt.Write(uint32(0))

	// Write allowable race
	_ = pkt.Write(uint32(0))

	// Write item level
	_ = pkt.Write(uint32(0))

	// Write required level
	_ = pkt.Write(uint32(0))

	// Write required skill
	_ = pkt.Write(uint32(0))

	// Write required skill rank
	_ = pkt.Write(uint32(0))

	// Write required spell
	_ = pkt.Write(uint32(0))

	// Write required reputation faction
	_ = pkt.Write(uint32(0))

	// Write required reputation rank
	_ = pkt.Write(uint32(0))

	// Write max count
	_ = pkt.Write(uint32(0))

	// Write max stack
	_ = pkt.Write(uint32(1))

	// Write container slots
	_ = pkt.Write(uint32(0))

	// Write stats count
	_ = pkt.Write(uint32(0))

	// Write 10 stat pairs (type + value)
	for i := 0; i < 10; i++ {
		_ = pkt.Write(uint32(0)) // stat type
		_ = pkt.Write(int32(0))  // stat value
	}

	// Write 5 damage pairs (min + max + type)
	for i := 0; i < 5; i++ {
		_ = pkt.Write(float32(0)) // min damage
		_ = pkt.Write(float32(0)) // max damage
		_ = pkt.Write(uint32(0))  // damage type
	}

	// Write armor
	_ = pkt.Write(uint32(0))

	// Write holy/fire/nature/frost/shadow/arcane resistance
	for i := 0; i < 6; i++ {
		_ = pkt.Write(uint32(0))
	}

	// Write delay
	_ = pkt.Write(uint32(0))

	// Write ammo type
	_ = pkt.Write(uint32(0))

	// Write ranged damage
	_ = pkt.Write(float32(0))

	// Write spell info (5 spells)
	for i := 0; i < 5; i++ {
		_ = pkt.Write(uint32(0)) // spell id
		_ = pkt.Write(uint32(0)) // trigger
		_ = pkt.Write(int32(0))  // charges
		_ = pkt.Write(int32(0))  // cooldown
		_ = pkt.Write(uint32(0)) // category
		_ = pkt.Write(int32(0))  // category cooldown
	}

	// Write socket info (3 sockets)
	for i := 0; i < 3; i++ {
		_ = pkt.Write(uint32(0)) // color
		_ = pkt.Write(uint32(0)) // content
	}

	// Write socket bonus
	_ = pkt.Write(uint32(0))

	// Write gem properties
	_ = pkt.Write(uint32(0))

	// Write item set
	_ = pkt.Write(uint32(0))

	// Write max durability
	_ = pkt.Write(uint32(0))

	// Write area
	_ = pkt.Write(uint32(0))

	// Write map
	_ = pkt.Write(uint32(0))

	// Write bag family
	_ = pkt.Write(uint32(0))

	// Write tool category
	_ = pkt.Write(uint32(0))

	// Write item set category
	_ = pkt.Write(uint32(0))

	// Write primary skill line
	_ = pkt.Write(uint32(0))

	// Write required skill rank
	_ = pkt.Write(uint32(0))

	// Write component skill line
	_ = pkt.Write(uint32(0))

	// Write primary stat
	_ = pkt.Write(int32(0))

	// Write secondary stat
	_ = pkt.Write(int32(0))

	// Write spell ICD
	_ = pkt.Write(uint32(0))

	gc.socket.Send(pkt)
}

// sendInventoryUpdate sends the player's inventory update.
func (gc *WorldSession) sendInventoryUpdate() {
	if gc.player == nil {
		return
	}

	// Build a values update with the inventory fields
	upd := &Updater{}
	pkt := upd.BuildInventoryUpdate(gc.player)

	if pkt != nil {
		gc.socket.Send(pkt)
	}
}

// equipStartingItems equips the starting items for a new character.
func (gc *WorldSession) equipStartingItems() {
	if gc.player == nil || gc.player.Inventory == nil {
		return
	}

	// Move items from backpack to equipment slots
	for i := player.EquipmentSlotHead; i < player.EquipmentSlotEnd; i++ {
		if gc.player.Inventory.GetEquipment(i) != nil {
			continue // Already equipped
		}

		// Find a matching item in backpack
		for j := player.InventorySlotItemStart; j < player.InventorySlotItemEnd; j++ {
			item := gc.player.Inventory.GetItem(j)
			if item == nil {
				continue
			}

			inventoryType := player.GetItemInventoryType(item.ItemEntry)
			slot := player.FindEquipSlot(inventoryType)
			if slot == i {
				// Move to equipment slot
				gc.player.Inventory.SetItem(j, nil)
				gc.player.Inventory.SetEquipment(i, item)
				item.SlotIndex = i
				item.UpdateFields()
				break
			}
		}
	}

	gc.player.UpdateInventoryFields()
}
