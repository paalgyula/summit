package world

import (
	"math"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/summit/world/vendor"
	"github.com/paalgyula/summit/pkg/wow"
)

// getVendorManager returns the vendor manager from the server.
func (gc *WorldSession) getVendorManager() *vendor.Manager {
	server, ok := gc.ws.(*Server)
	if !ok {
		return nil
	}

	return server.vendorMgr
}

// SendListInventory sends SMSG_LIST_INVENTORY to the player.
func (gc *WorldSession) SendListInventory(vendorGUID wow.GUID) {
	if gc.player == nil || gc.player.IsDead() {
		return
	}

	npc := gc.getNPCByGUID(vendorGUID)
	if npc == nil || !vendor.IsVendor(npc.NpcFlags) {
		gc.log.Debug().
			Str("vendor", vendorGUID.String()).
			Msg("SendListInventory: vendor not found or not interactable")

		gc.socket.Send(vendor.BuildSellItem(vendorGUID, 0, vendor.SellErrCantFindVendor))

		return
	}

	// Distance check (within 10 yards)
	dx := gc.player.Location.X - npc.X
	dy := gc.player.Location.Y - npc.Y
	dz := gc.player.Location.Z - npc.Z
	if math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) > 15.0 {
		gc.log.Debug().
			Str("vendor", vendorGUID.String()).
			Msg("SendListInventory: too far from vendor")

		gc.socket.Send(vendor.BuildSellItem(vendorGUID, 0, vendor.SellErrCantFindVendor))

		return
	}

	vMgr := gc.getVendorManager()
	if vMgr == nil {
		return
	}

	items := vMgr.GetVendorItems(npc.EntryID)
	pkt := vendor.BuildListInventory(vendorGUID, items, gc.getItemTemplate, 1.0)
	gc.socket.Send(pkt)
}

// HandleListInventory handles CMSG_LIST_INVENTORY (0x19E).
func (gc *WorldSession) HandleListInventory(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var vendorGUID uint64
	_ = reader.Read(&vendorGUID)

	gc.log.Debug().
		Uint64("vendorGUID", vendorGUID).
		Msg("handling CMSG_LIST_INVENTORY")

	gc.SendListInventory(wow.GUID(vendorGUID))
}

// HandleBuyItem handles CMSG_BUY_ITEM (0x1A2).
func (gc *WorldSession) HandleBuyItem(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var vendorGUID uint64
	var itemID uint32
	var slot uint32
	var count uint32

	_ = reader.Read(&vendorGUID)
	_ = reader.Read(&itemID)
	_ = reader.Read(&slot)
	_ = reader.Read(&count)

	gc.log.Debug().
		Uint64("vendorGUID", vendorGUID).
		Uint32("itemID", itemID).
		Uint32("slot", slot).
		Uint32("count", count).
		Msg("handling CMSG_BUY_ITEM")

	gc.buyItem(vendorGUID, itemID, slot, count)
}

// HandleBuyItemInSlot handles CMSG_BUY_ITEM_IN_SLOT (0x1A3).
func (gc *WorldSession) HandleBuyItemInSlot(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var vendorGUID uint64
	var itemID uint32
	var slot uint32
	var bagGUID uint64
	var bagSlot uint8
	var count uint32

	_ = reader.Read(&vendorGUID)
	_ = reader.Read(&itemID)
	_ = reader.Read(&slot)
	_ = reader.Read(&bagGUID)
	_ = reader.Read(&bagSlot)
	_ = reader.Read(&count)

	gc.log.Debug().
		Uint64("vendorGUID", vendorGUID).
		Uint32("itemID", itemID).
		Uint32("slot", slot).
		Uint64("bagGUID", bagGUID).
		Uint8("bagSlot", bagSlot).
		Uint32("count", count).
		Msg("handling CMSG_BUY_ITEM_IN_SLOT")

	gc.buyItem(vendorGUID, itemID, slot, count)
}

// buyItem processes purchasing an item from a vendor.
func (gc *WorldSession) buyItem(vendorGUID uint64, itemID uint32, slot uint32, count uint32) {
	if gc.player == nil || gc.player.IsDead() {
		return
	}

	// Client sends slot starting at 1
	if slot == 0 {
		return
	}
	vendorslot := slot - 1

	if count < 1 {
		count = 1
	}

	npc := gc.getNPCByGUID(wow.GUID(vendorGUID))
	if npc == nil || !vendor.IsVendor(npc.NpcFlags) {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrCantFindItem))
		return
	}

	// Distance check (within 15 yards)
	dx := gc.player.Location.X - npc.X
	dy := gc.player.Location.Y - npc.Y
	dz := gc.player.Location.Z - npc.Z
	if math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) > 15.0 {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrDistanceTooFar))
		return
	}

	vMgr := gc.getVendorManager()
	if vMgr == nil {
		return
	}

	vItem := vMgr.GetVendorItem(npc.EntryID, vendorslot)
	if vItem == nil || vItem.Item.Item != itemID {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrCantFindItem))
		return
	}

	tpl := gc.getItemTemplate(itemID)
	if tpl == nil {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrCantFindItem))
		return
	}

	buyCountPerStack := uint32(1)
	if tpl.BuyCount > 1 {
		buyCountPerStack = uint32(tpl.BuyCount)
	}

	totalItemsToReceive := buyCountPerStack * count

	// Check limited stock
	if vItem.Item.MaxCount > 0 {
		if vItem.CurrentCount < totalItemsToReceive {
			gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrItemAlreadySold))
			return
		}
	}

	// Check money
	totalCost := uint32(tpl.BuyPrice) * count
	if gc.player.Money < totalCost {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrNotEnoughMoney))
		return
	}

	// Check bag space before deducting money
	emptySlot := gc.player.Inventory.FindEmptyBackpackSlot()
	maxStack := uint32(tpl.GetMaxStackSize())
	canFit := false

	if maxStack > 1 {
		// Check if it fits in existing stacks
		for i := player.InventorySlotItemStart; i < player.InventorySlotItemEnd; i++ {
			existing := gc.player.Inventory.GetItem(i)
			if existing != nil && existing.ItemEntry == itemID && existing.StackCount < maxStack {
				canFit = true
				break
			}
		}
	}
	if emptySlot >= 0 {
		canFit = true
	}

	if !canFit {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrCantCarryMore))
		gc.sendInventoryChangeFailure(nil, nil, player.EquipResultInventoryFull)
		return
	}

	// Deduct money
	gc.player.ModifyMoney(-int32(totalCost))

	// Update stock
	newStock, ok := vMgr.UpdateStock(npc.EntryID, vendorslot, totalItemsToReceive)
	if !ok {
		// Refund if race condition
		gc.player.ModifyMoney(int32(totalCost))
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrItemAlreadySold))
		return
	}

	// Create and add item
	newItem := player.NewItem(itemID, gc.player.GUID())
	newItem.StackCount = totalItemsToReceive
	newItem.EnsureGUID()

	res := gc.player.Inventory.AddItem(newItem)
	if !res.Placed && len(res.StackedInto) == 0 {
		// Fallback refund
		gc.player.ModifyMoney(int32(totalCost))
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), itemID, vendor.BuyErrCantCarryMore))
		return
	}

	leftInStock := int32(-1)
	if vItem.Item.MaxCount > 0 {
		leftInStock = int32(newStock)
	}

	// Send SMSG_BUY_ITEM
	gc.socket.Send(vendor.BuildBuyItem(wow.GUID(vendorGUID), slot, leftInStock, count))

	// Count total of this item in player inventory for item push result
	totalInInventory := uint32(0)
	for i := 0; i < player.InventorySlotTotal; i++ {
		if it := gc.player.Inventory.GetItem(i); it != nil && it.ItemEntry == itemID {
			totalInInventory += it.StackCount
		}
	}

	// Send SMSG_ITEM_PUSH_RESULT
	gc.socket.Send(vendor.BuildItemPushResult(
		gc.player.GUID(),
		1, // received from NPC
		0, // created
		1, // sendChatMessage
		0, // bagSlot
		uint32(res.Slot),
		itemID,
		0, // suffixFactor
		0, // randomPropertyID
		totalItemsToReceive,
		totalInInventory,
	))

	// Send item updates
	upd := &Updater{}
	for _, stacked := range res.StackedInto {
		if pkt := upd.BuildItemValuesUpdate(stacked, gc.player); pkt != nil {
			gc.socket.Send(pkt)
		}
	}
	if res.Placed {
		if pkt := upd.BuildItemCreateObject(newItem, gc.player); pkt != nil {
			gc.socket.Send(pkt)
		}
	}

	gc.player.UpdateInventoryFields()
	gc.sendInventoryUpdate()

	gc.log.Info().
		Uint32("itemID", itemID).
		Uint32("count", count).
		Uint32("totalCost", totalCost).
		Msg("player bought item from vendor")
}

// HandleSellItem handles CMSG_SELL_ITEM (0x1A0).
func (gc *WorldSession) HandleSellItem(data wow.PacketData) {
	if gc.player == nil || gc.player.IsDead() {
		return
	}

	reader := wow.NewPacketReader(data)

	var vendorGUID uint64
	var itemGUID uint64
	var count uint32

	_ = reader.Read(&vendorGUID)
	_ = reader.Read(&itemGUID)
	_ = reader.Read(&count)

	gc.log.Debug().
		Uint64("vendorGUID", vendorGUID).
		Uint64("itemGUID", itemGUID).
		Uint32("count", count).
		Msg("handling CMSG_SELL_ITEM")

	npc := gc.getNPCByGUID(wow.GUID(vendorGUID))
	if npc == nil || !vendor.IsVendor(npc.NpcFlags) {
		gc.socket.Send(vendor.BuildSellItem(wow.GUID(vendorGUID), wow.GUID(itemGUID), vendor.SellErrCantFindVendor))
		return
	}

	// Distance check (within 15 yards)
	dx := gc.player.Location.X - npc.X
	dy := gc.player.Location.Y - npc.Y
	dz := gc.player.Location.Z - npc.Z
	if math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) > 15.0 {
		gc.socket.Send(vendor.BuildSellItem(wow.GUID(vendorGUID), wow.GUID(itemGUID), vendor.SellErrCantFindVendor))
		return
	}

	// Find the item in player inventory or equipment
	var foundItem *player.Item
	var itemSlot int = -1

	for i := 0; i < player.InventorySlotTotal; i++ {
		it := gc.player.Inventory.GetItem(i)
		if it != nil && uint64(it.GUID()) == itemGUID {
			foundItem = it
			itemSlot = i
			break
		}
	}

	if foundItem == nil {
		gc.socket.Send(vendor.BuildSellItem(wow.GUID(vendorGUID), wow.GUID(itemGUID), vendor.SellErrCantFindItem))
		return
	}

	tpl := gc.getItemTemplate(foundItem.ItemEntry)
	if tpl == nil {
		gc.socket.Send(vendor.BuildSellItem(wow.GUID(vendorGUID), wow.GUID(itemGUID), vendor.SellErrCantFindItem))
		return
	}

	if tpl.SellPrice == 0 {
		gc.socket.Send(vendor.BuildSellItem(wow.GUID(vendorGUID), wow.GUID(itemGUID), vendor.SellErrCantSellItem))
		return
	}

	sellCount := foundItem.StackCount
	if count > 0 && count < foundItem.StackCount {
		sellCount = count
	}

	money := tpl.SellPrice * sellCount

	// Calculate durability repair deduction if item is damaged
	if foundItem.MaxDurability > 0 && foundItem.Durability < foundItem.MaxDurability {
		durabilityLost := foundItem.MaxDurability - foundItem.Durability
		// Basic durability cost estimation
		durabilityCost := uint32(float32(durabilityLost) * float32(tpl.ItemLevel) * 0.5)
		if durabilityCost >= money {
			money = 1 // minimum 1 copper
		} else {
			money -= durabilityCost
		}
	}

	upd := &Updater{}

	if sellCount < foundItem.StackCount {
		// Split stack: create a clone for the sold portion and put it into buyback
		clonedItem := player.NewItem(foundItem.ItemEntry, gc.player.GUID())
		clonedItem.StackCount = sellCount
		clonedItem.Durability = foundItem.Durability
		clonedItem.MaxDurability = foundItem.MaxDurability
		clonedItem.EnsureGUID()

		foundItem.StackCount -= sellCount
		foundItem.UpdateFields()

		// Update original item stack on client
		if pkt := upd.BuildItemValuesUpdate(foundItem, gc.player); pkt != nil {
			gc.socket.Send(pkt)
		}

		// Add cloned item to buyback and send create block to client
		gc.player.AddItemToBuyBackSlot(clonedItem, money)
		if pkt := upd.BuildItemCreateObject(clonedItem, gc.player); pkt != nil {
			gc.socket.Send(pkt)
		}
	} else {
		// Full stack sold: remove from inventory and move to buyback
		gc.player.Inventory.RemoveItem(itemSlot)
		gc.player.AddItemToBuyBackSlot(foundItem, money)
	}

	gc.player.ModifyMoney(int32(money))
	gc.player.UpdateInventoryFields()
	gc.sendInventoryUpdate()

	gc.log.Info().
		Uint32("itemEntry", foundItem.ItemEntry).
		Uint32("count", sellCount).
		Uint32("moneyEarned", money).
		Msg("player sold item to vendor")
}

// HandleBuybackItem handles CMSG_BUYBACK_ITEM (0x290).
func (gc *WorldSession) HandleBuybackItem(data wow.PacketData) {
	if gc.player == nil || gc.player.IsDead() {
		return
	}

	reader := wow.NewPacketReader(data)

	var vendorGUID uint64
	var slot uint32

	_ = reader.Read(&vendorGUID)
	_ = reader.Read(&slot)

	gc.log.Debug().
		Uint64("vendorGUID", vendorGUID).
		Uint32("slot", slot).
		Msg("handling CMSG_BUYBACK_ITEM")

	npc := gc.getNPCByGUID(wow.GUID(vendorGUID))
	if npc == nil || !vendor.IsVendor(npc.NpcFlags) {
		gc.socket.Send(vendor.BuildSellItem(wow.GUID(vendorGUID), 0, vendor.SellErrCantFindVendor))
		return
	}

	// Distance check (within 15 yards)
	dx := gc.player.Location.X - npc.X
	dy := gc.player.Location.Y - npc.Y
	dz := gc.player.Location.Z - npc.Z
	if math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) > 15.0 {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), 0, vendor.BuyErrDistanceTooFar))
		return
	}

	buybackItem := gc.player.GetItemFromBuyBackSlot(slot)
	if buybackItem == nil {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), 0, vendor.BuyErrCantFindItem))
		return
	}

	price := gc.player.GetBuybackPrice(slot)
	if gc.player.Money < price {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), buybackItem.ItemEntry, vendor.BuyErrNotEnoughMoney))
		return
	}

	// Check if backpack has space
	if gc.player.Inventory.FindEmptyBackpackSlot() < 0 {
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), buybackItem.ItemEntry, vendor.BuyErrCantCarryMore))
		gc.sendInventoryChangeFailure(buybackItem, nil, player.EquipResultInventoryFull)
		return
	}

	// Deduct money
	gc.player.ModifyMoney(-int32(price))

	// Remove item from buyback slot
	removedItem := gc.player.RemoveItemFromBuyBackSlot(slot, false)
	if removedItem == nil {
		return
	}

	// Add item back into player inventory
	res := gc.player.Inventory.AddItem(removedItem)
	if !res.Placed && len(res.StackedInto) == 0 {
		// Restore to buyback if add failed
		gc.player.AddItemToBuyBackSlot(removedItem, price)
		gc.player.ModifyMoney(int32(price))
		gc.socket.Send(vendor.BuildBuyFailed(wow.GUID(vendorGUID), buybackItem.ItemEntry, vendor.BuyErrCantCarryMore))
		return
	}

	upd := &Updater{}
	for _, stacked := range res.StackedInto {
		if pkt := upd.BuildItemValuesUpdate(stacked, gc.player); pkt != nil {
			gc.socket.Send(pkt)
		}
	}
	if res.Placed {
		if pkt := upd.BuildItemCreateObject(removedItem, gc.player); pkt != nil {
			gc.socket.Send(pkt)
		}
	}

	gc.player.UpdateInventoryFields()
	gc.sendInventoryUpdate()

	gc.log.Info().
		Uint32("itemEntry", removedItem.ItemEntry).
		Uint32("price", price).
		Msg("player bought back item from vendor")
}
