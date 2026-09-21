package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
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

	// Validate equip
	tpl := gc.getItemTemplate(uint32(item.ItemEntry))
	result := player.CanEquipItem(gc.player, item, tpl, -1)
	if result != player.EquipResultOK {
		gc.log.Warn().
			Int("result", int(result)).
			Uint32("entry", item.ItemEntry).
			Msg("cannot equip item")

		gc.sendInventoryChangeFailure(item, nil, result)
		return
	}

	// Equip the item with validation
	prev, result := gc.player.EquipItemWithValidation(item, tpl, -1)
	if result != player.EquipResultOK {
		gc.log.Warn().
			Int("result", int(result)).
			Uint32("entry", item.ItemEntry).
			Msg("equip failed")

		gc.sendInventoryChangeFailure(item, prev, result)
		return
	}

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

	// Validate equip
	tpl := gc.getItemTemplate(uint32(item.ItemEntry))
	result := player.CanEquipItemInSlot(gc.player, item, tpl, int(destSlot))
	if result != player.EquipResultOK {
		gc.log.Warn().
			Int("result", int(result)).
			Uint32("entry", item.ItemEntry).
			Msg("cannot equip item in slot")

		gc.sendInventoryChangeFailure(item, nil, result)
		return
	}

	// Equip with validation
	prev, result := gc.player.EquipItemWithValidation(item, tpl, int(destSlot))
	if result != player.EquipResultOK {
		gc.sendInventoryChangeFailure(item, prev, result)
		return
	}

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

	// Try to look up the full template
	tpl := gc.getItemTemplate(entry)
	if tpl == nil {
		// Item not found - send minimal response with error
		_ = pkt.Write(uint32(tpl.Class))
		_ = pkt.Write(uint32(tpl.SubClass))
		pkt.WriteString(tpl.Name)

		// Write remaining fields as zeros
		for i := 0; i < 32; i++ {
			_ = pkt.Write(uint32(0))
		}

		gc.socket.Send(pkt)
		return
	}

	// Write item class
	_ = pkt.Write(tpl.Class)
	// Write subclass
	_ = pkt.Write(tpl.SubClass)
	// Write name
	pkt.WriteString(tpl.Name)

	// Write displayid
	_ = pkt.Write(tpl.DisplayID)
	// Write quality
	_ = pkt.Write(tpl.Quality)
	// Write flags
	_ = pkt.Write(tpl.Flags)
	// Write buy price
	_ = pkt.Write(uint32(tpl.BuyPrice))
	// Write sell price
	_ = pkt.Write(tpl.SellPrice)
	// Write inventory type
	_ = pkt.Write(tpl.InventoryType)
	// Write allowable class
	_ = pkt.Write(uint32(tpl.AllowableClass))
	// Write allowable race
	_ = pkt.Write(uint32(tpl.AllowableRace))
	// Write item level
	_ = pkt.Write(tpl.ItemLevel)
	// Write required level
	_ = pkt.Write(tpl.RequiredLevel)
	// Write required skill
	_ = pkt.Write(tpl.RequiredSkill)
	// Write required skill rank
	_ = pkt.Write(tpl.RequiredSkillRank)
	// Write required spell
	_ = pkt.Write(uint32(0))
	// Write required reputation faction
	_ = pkt.Write(tpl.RequiredRepFaction)
	// Write required reputation rank
	_ = pkt.Write(tpl.RequiredRepRank)
	// Write max count
	_ = pkt.Write(uint32(tpl.MaxCount))
	// Write max stack
	_ = pkt.Write(uint32(tpl.Stackable))
	// Write container slots
	_ = pkt.Write(tpl.ContainerSlots)

	// Write stats count
	_ = pkt.Write(tpl.StatsCount)

	// Write 10 stat pairs (type + value)
	for i := 0; i < 10; i++ {
		if i < len(tpl.Stats) {
			_ = pkt.Write(tpl.Stats[i].Type)
			_ = pkt.Write(tpl.Stats[i].Value)
		} else {
			_ = pkt.Write(uint32(0))
			_ = pkt.Write(int32(0))
		}
	}

	// Write 2 damage pairs (min + max + type)
	for i := 0; i < 2; i++ {
		if i < len(tpl.Damage) {
			_ = pkt.Write(tpl.Damage[i].Min)
			_ = pkt.Write(tpl.Damage[i].Max)
			_ = pkt.Write(tpl.Damage[i].Type)
		} else {
			_ = pkt.Write(float32(0))
			_ = pkt.Write(float32(0))
			_ = pkt.Write(uint32(0))
		}
	}

	// Write armor
	_ = pkt.Write(tpl.Armor)

	// Write holy/fire/nature/frost/shadow/arcane resistance
	_ = pkt.Write(uint32(tpl.HolyRes))
	_ = pkt.Write(uint32(tpl.FireRes))
	_ = pkt.Write(uint32(tpl.NatureRes))
	_ = pkt.Write(uint32(tpl.FrostRes))
	_ = pkt.Write(uint32(tpl.ShadowRes))
	_ = pkt.Write(uint32(tpl.ArcaneRes))

	// Write delay
	_ = pkt.Write(tpl.Delay)
	// Write ammo type
	_ = pkt.Write(tpl.AmmoType)
	// Write ranged damage
	_ = pkt.Write(tpl.RangedModRange)

	// Write spell info (5 spells)
	for i := 0; i < 5; i++ {
		if i < len(tpl.Spells) {
			_ = pkt.Write(uint32(tpl.Spells[i].SpellID))
			_ = pkt.Write(tpl.Spells[i].Trigger)
			_ = pkt.Write(tpl.Spells[i].Charges)
			_ = pkt.Write(tpl.Spells[i].Cooldown)
			_ = pkt.Write(tpl.Spells[i].Category)
			_ = pkt.Write(tpl.Spells[i].CategoryCooldown)
		} else {
			_ = pkt.Write(uint32(0))
			_ = pkt.Write(uint32(0))
			_ = pkt.Write(int32(0))
			_ = pkt.Write(int32(0))
			_ = pkt.Write(uint32(0))
			_ = pkt.Write(int32(0))
		}
	}

	// Write socket info (3 sockets)
	for i := 0; i < 3; i++ {
		if i < len(tpl.Sockets) {
			_ = pkt.Write(tpl.Sockets[i].Color)
			_ = pkt.Write(tpl.Sockets[i].Content)
		} else {
			_ = pkt.Write(uint32(0))
			_ = pkt.Write(uint32(0))
		}
	}

	// Write socket bonus
	_ = pkt.Write(tpl.SocketBonus)

	// Write gem properties
	_ = pkt.Write(uint32(0))

	// Write item set
	_ = pkt.Write(tpl.ItemSet)

	// Write max durability
	_ = pkt.Write(tpl.MaxDurability)

	// Write area
	_ = pkt.Write(uint32(0))
	// Write map
	_ = pkt.Write(uint32(0))

	// Write bag family
	_ = pkt.Write(tpl.BagFamily)

	// Write tool category
	_ = pkt.Write(uint32(0))
	// Write item set category
	_ = pkt.Write(uint32(0))
	// Write primary skill line
	_ = pkt.Write(uint32(0))
	// Write required skill rank
	_ = pkt.Write(tpl.RequiredSkillRank)
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

// getItemTemplate retrieves an item template from basedata.
func (gc *WorldSession) getItemTemplate(entry uint32) *basedata.ItemTemplate {
	return basedata.GetInstance().LookupItem(entry)
}

// sendInventoryChangeFailure sends SMSG_INVENTORY_CHANGE_FAILURE.
func (gc *WorldSession) sendInventoryChangeFailure(item *player.Item, other *player.Item, result player.EquipResult) {
	if gc.player == nil {
		return
	}

	pkt := wow.NewPacket(wow.ServerInventoryChangeFailure)

	_ = pkt.Write(uint8(result))
	_ = pkt.Write(uint64(item.GUID()))

	if other != nil {
		_ = pkt.Write(uint64(other.GUID()))
	} else {
		_ = pkt.Write(uint64(0))
	}

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

// unused import guard
var _ = fmt.Sprintf
