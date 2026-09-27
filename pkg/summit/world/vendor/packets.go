package vendor

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
)

// BuildListInventory creates SMSG_LIST_INVENTORY (0x19F) matching AzerothCore.
func BuildListInventory(
	vendorGUID wow.GUID,
	items []*VendorItemInstance,
	getItemTemplate func(entry uint32) *basedata.ItemTemplate,
	discountMod float32,
) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerListInventory)
	_ = pkt.Write(vendorGUID)

	// Filter valid items
	type itemData struct {
		slot         uint32
		itemID       uint32
		displayID    uint32
		leftInStock  int32
		price        uint32
		durability   uint32
		buyCount     uint32
		extendedCost uint32
	}

	var validItems []itemData

	for i, it := range items {
		if len(validItems) >= MaxVendorItems {
			break
		}

		tpl := getItemTemplate(it.Item.Item)
		if tpl == nil {
			continue
		}

		leftInStock := int32(-1)
		if it.Item.MaxCount > 0 {
			leftInStock = int32(it.CurrentCount)
			if leftInStock <= 0 {
				continue // sold out items not displayed
			}
		}

		price := uint32(float32(tpl.BuyPrice) * discountMod)

		buyCount := uint32(1)
		if tpl.BuyCount > 1 {
			buyCount = uint32(tpl.BuyCount)
		}

		validItems = append(validItems, itemData{
			slot:         uint32(i + 1), // 1-indexed slot
			itemID:       it.Item.Item,
			displayID:    tpl.DisplayID,
			leftInStock:  leftInStock,
			price:        price,
			durability:   tpl.MaxDurability,
			buyCount:     buyCount,
			extendedCost: it.Item.ExtendedCost,
		})
	}

	_ = pkt.WriteOne(len(validItems))

	if len(validItems) == 0 {
		_ = pkt.WriteOne(0) // Error byte: vendor has no inventory
		return pkt
	}

	for _, it := range validItems {
		_ = pkt.WriteUint32(int(it.slot))
		_ = pkt.WriteUint32(int(it.itemID))
		_ = pkt.WriteUint32(int(it.displayID))
		_ = pkt.Write(it.leftInStock)
		_ = pkt.WriteUint32(int(it.price))
		_ = pkt.WriteUint32(int(it.durability))
		_ = pkt.WriteUint32(int(it.buyCount))
		_ = pkt.WriteUint32(int(it.extendedCost))
	}

	return pkt
}

// BuildBuyItem creates SMSG_BUY_ITEM (0x1A4) matching AzerothCore.
func BuildBuyItem(vendorGUID wow.GUID, slot uint32, leftInStock int32, boughtCount uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerBuyItem)
	_ = pkt.Write(vendorGUID)
	_ = pkt.WriteUint32(int(slot))
	_ = pkt.Write(leftInStock)
	_ = pkt.WriteUint32(int(boughtCount))

	return pkt
}

// BuildBuyFailed creates SMSG_BUY_FAILED (0x1A5) matching AzerothCore.
func BuildBuyFailed(vendorGUID wow.GUID, itemID uint32, err BuyResult) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerBuyFailed)
	_ = pkt.Write(vendorGUID)
	_ = pkt.WriteUint32(int(itemID))
	_ = pkt.WriteOne(int(err))

	return pkt
}

// BuildSellItem creates SMSG_SELL_ITEM (0x1A1) matching AzerothCore.
func BuildSellItem(vendorGUID, itemGUID wow.GUID, err SellResult) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerSellItem)
	_ = pkt.Write(vendorGUID)
	_ = pkt.Write(itemGUID)
	_ = pkt.WriteOne(int(err))

	return pkt
}

// BuildItemPushResult creates SMSG_ITEM_PUSH_RESULT (0x166) matching AzerothCore.
func BuildItemPushResult(
	playerGUID wow.GUID,
	received, created, sendChatMessage uint32,
	bagSlot uint8,
	itemSlot uint32,
	itemID, suffixFactor uint32,
	randomPropertyID int32,
	count, totalCount uint32,
) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerItemPushResult)
	_ = pkt.Write(playerGUID)
	_ = pkt.WriteUint32(int(received))
	_ = pkt.WriteUint32(int(created))
	_ = pkt.WriteUint32(int(sendChatMessage))
	_ = pkt.WriteOne(int(bagSlot))
	_ = pkt.WriteUint32(int(itemSlot))
	_ = pkt.WriteUint32(int(itemID))
	_ = pkt.WriteUint32(int(suffixFactor))
	_ = pkt.Write(randomPropertyID)
	_ = pkt.WriteUint32(int(count))
	_ = pkt.WriteUint32(int(totalCount))

	return pkt
}
