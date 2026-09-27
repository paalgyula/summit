package vendor

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackets_ListInventory(t *testing.T) {
	vendorGUID := wow.NewGUID(wow.UnitGUID, 12345)

	items := []*VendorItemInstance{
		{
			Item: store.VendorItem{
				Entry:        12345,
				Slot:         1,
				Item:         25, // Worn Shortsword
				MaxCount:     0,
				IncrTime:     0,
				ExtendedCost: 0,
			},
			CurrentCount: 0,
		},
		{
			Item: store.VendorItem{
				Entry:        12345,
				Slot:         2,
				Item:         4540, // Tough Jerky
				MaxCount:     5,
				IncrTime:     300,
				ExtendedCost: 0,
			},
			CurrentCount: 5,
		},
	}

	getItemTemplate := func(id uint32) *basedata.ItemTemplate {
		if id == 25 {
			return &basedata.ItemTemplate{
				Entry:         25,
				DisplayID:     1500,
				BuyPrice:      35,
				MaxCount:      0,
				Stackable:     1,
				MaxDurability: 20,
			}
		}
		if id == 4540 {
			return &basedata.ItemTemplate{
				Entry:         4540,
				DisplayID:     1200,
				BuyPrice:      25,
				MaxCount:      0,
				Stackable:     20,
				BuyCount:      5,
				MaxDurability: 0,
			}
		}
		return nil
	}

	pkt := BuildListInventory(vendorGUID, items, getItemTemplate, 1.0)
	require.NotNil(t, pkt)
	assert.Equal(t, wow.ServerListInventory, pkt.Opcode())

	reader := wow.NewPacketReader(pkt.Bytes())
	var guid uint64
	var count uint8
	require.NoError(t, reader.Read(&guid))
	require.NoError(t, reader.Read(&count))

	assert.Equal(t, uint64(vendorGUID), guid)
	assert.Equal(t, uint8(2), count)

	// Item 1
	var slot1, itemID1, displayID1 uint32
	var maxCount1 int32
	var price1, durability1, buyCount1, extCost1 uint32
	require.NoError(t, reader.Read(&slot1))
	require.NoError(t, reader.Read(&itemID1))
	require.NoError(t, reader.Read(&displayID1))
	require.NoError(t, reader.Read(&maxCount1))
	require.NoError(t, reader.Read(&price1))
	require.NoError(t, reader.Read(&durability1))
	require.NoError(t, reader.Read(&buyCount1))
	require.NoError(t, reader.Read(&extCost1))

	assert.Equal(t, uint32(1), slot1)
	assert.Equal(t, uint32(25), itemID1)
	assert.Equal(t, uint32(1500), displayID1)
	assert.Equal(t, int32(-1), maxCount1) // unlimited stock encoded as -1 (0xFFFFFFFF)
	assert.Equal(t, uint32(35), price1)
	assert.Equal(t, uint32(20), durability1)
	assert.Equal(t, uint32(1), buyCount1)
	assert.Equal(t, uint32(0), extCost1)

	// Item 2
	var slot2, itemID2, displayID2 uint32
	var maxCount2 int32
	var price2, durability2, buyCount2, extCost2 uint32
	require.NoError(t, reader.Read(&slot2))
	require.NoError(t, reader.Read(&itemID2))
	require.NoError(t, reader.Read(&displayID2))
	require.NoError(t, reader.Read(&maxCount2))
	require.NoError(t, reader.Read(&price2))
	require.NoError(t, reader.Read(&durability2))
	require.NoError(t, reader.Read(&buyCount2))
	require.NoError(t, reader.Read(&extCost2))

	assert.Equal(t, uint32(2), slot2)
	assert.Equal(t, uint32(4540), itemID2)
	assert.Equal(t, uint32(1200), displayID2)
	assert.Equal(t, int32(5), maxCount2)
	assert.Equal(t, uint32(25), price2)
	assert.Equal(t, uint32(0), durability2)
	assert.Equal(t, uint32(5), buyCount2)
	assert.Equal(t, uint32(0), extCost2)
}

func TestPackets_BuyItem(t *testing.T) {
	vendorGUID := wow.NewGUID(wow.UnitGUID, 999)
	pkt := BuildBuyItem(vendorGUID, 2, -1, 1)
	require.NotNil(t, pkt)
	assert.Equal(t, wow.ServerBuyItem, pkt.Opcode())

	reader := wow.NewPacketReader(pkt.Bytes())
	var guid uint64
	var slot uint32
	var leftInStock int32
	var boughtCount uint32
	require.NoError(t, reader.Read(&guid))
	require.NoError(t, reader.Read(&slot))
	require.NoError(t, reader.Read(&leftInStock))
	require.NoError(t, reader.Read(&boughtCount))

	assert.Equal(t, uint64(vendorGUID), guid)
	assert.Equal(t, uint32(2), slot)
	assert.Equal(t, int32(-1), leftInStock)
	assert.Equal(t, uint32(1), boughtCount)
}

func TestPackets_BuyFailed(t *testing.T) {
	vendorGUID := wow.NewGUID(wow.UnitGUID, 999)
	pkt := BuildBuyFailed(vendorGUID, 1234, BuyErrNotEnoughMoney)
	require.NotNil(t, pkt)
	assert.Equal(t, wow.ServerBuyFailed, pkt.Opcode())

	reader := wow.NewPacketReader(pkt.Bytes())
	var guid uint64
	var itemID uint32
	var reason uint8
	require.NoError(t, reader.Read(&guid))
	require.NoError(t, reader.Read(&itemID))
	require.NoError(t, reader.Read(&reason))

	assert.Equal(t, uint64(vendorGUID), guid)
	assert.Equal(t, uint32(1234), itemID)
	assert.Equal(t, uint8(BuyErrNotEnoughMoney), reason)
}

func TestPackets_SellItem(t *testing.T) {
	vendorGUID := wow.NewGUID(wow.UnitGUID, 999)
	itemGUID := wow.NewGUID(wow.ItemGUID, 555)
	pkt := BuildSellItem(vendorGUID, itemGUID, SellErrCantSellItem)
	require.NotNil(t, pkt)
	assert.Equal(t, wow.ServerSellItem, pkt.Opcode())

	reader := wow.NewPacketReader(pkt.Bytes())
	var vGUID, iGUID uint64
	var reason uint8
	require.NoError(t, reader.Read(&vGUID))
	require.NoError(t, reader.Read(&iGUID))
	require.NoError(t, reader.Read(&reason))

	assert.Equal(t, uint64(vendorGUID), vGUID)
	assert.Equal(t, uint64(itemGUID), iGUID)
	assert.Equal(t, uint8(SellErrCantSellItem), reason)
}

func TestPackets_ItemPushResult(t *testing.T) {
	playerGUID := wow.NewGUID(wow.PlayerGUID, 100)
	pkt := BuildItemPushResult(playerGUID, 1, 0, 1, 0xFF, 0xFFFFFFFF, 25, 0, 0, 1, 1)
	require.NotNil(t, pkt)
	assert.Equal(t, wow.ServerItemPushResult, pkt.Opcode())

	reader := wow.NewPacketReader(pkt.Bytes())
	var guid uint64
	var received, created, showInChat uint32
	var bagSlot uint8
	var itemSlot, itemID, suffix, randProp, count, stackCount uint32

	require.NoError(t, reader.Read(&guid))
	require.NoError(t, reader.Read(&received))
	require.NoError(t, reader.Read(&created))
	require.NoError(t, reader.Read(&showInChat))
	require.NoError(t, reader.Read(&bagSlot))
	require.NoError(t, reader.Read(&itemSlot))
	require.NoError(t, reader.Read(&itemID))
	require.NoError(t, reader.Read(&suffix))
	require.NoError(t, reader.Read(&randProp))
	require.NoError(t, reader.Read(&count))
	require.NoError(t, reader.Read(&stackCount))

	assert.Equal(t, uint64(playerGUID), guid)
	assert.Equal(t, uint32(1), received)
	assert.Equal(t, uint32(0), created)
	assert.Equal(t, uint32(1), showInChat)
	assert.Equal(t, uint8(0xFF), bagSlot)
	assert.Equal(t, uint32(0xFFFFFFFF), itemSlot)
	assert.Equal(t, uint32(25), itemID)
	assert.Equal(t, uint32(1), count)
	assert.Equal(t, uint32(1), stackCount)
}

func TestManager_FallbackAndStock(t *testing.T) {
	mgr := NewManager(nil)
	require.NotNil(t, mgr)

	// Fallback vendor 54 should exist
	assert.True(t, mgr.HasVendorItems(54))
	items := mgr.GetVendorItems(54)
	assert.NotEmpty(t, items)

	// Add a test vendor with limited stock
	testEntry := uint32(9999)
	mgr.AddVendorItem(testEntry, store.VendorItem{
		Entry:        testEntry,
		Slot:         1,
		Item:         1111,
		MaxCount:     3,
		IncrTime:     1, // 1 second restock
		ExtendedCost: 0,
	})

	assert.True(t, mgr.HasVendorItems(testEntry))
	inst := mgr.GetVendorItem(testEntry, 0)
	require.NotNil(t, inst)
	assert.Equal(t, uint32(3), inst.CurrentCount)

	// Buy 2 items -> stock should become 1
	rem, ok := mgr.UpdateStock(testEntry, 0, 2)
	assert.True(t, ok)
	assert.Equal(t, uint32(1), rem)

	// Buy 1 item -> stock becomes 0
	rem, ok = mgr.UpdateStock(testEntry, 0, 1)
	assert.True(t, ok)
	assert.Equal(t, uint32(0), rem)

	// Try to buy 1 item when 0 remaining -> should fail
	_, ok = mgr.UpdateStock(testEntry, 0, 1)
	assert.False(t, ok)

	// Wait 1.1s for restock timer
	time.Sleep(1100 * time.Millisecond)

	// Calling GetVendorItems should replenish stock by 1
	items = mgr.GetVendorItems(testEntry)
	require.NotEmpty(t, items)
	assert.GreaterOrEqual(t, items[0].CurrentCount, uint32(1))
}
