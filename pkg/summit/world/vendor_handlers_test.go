package world

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/summit/world/quest"
	"github.com/paalgyula/summit/pkg/summit/world/vendor"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/protocol"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createVendorTestHarness(t *testing.T) (*WorldSession, *Server, chan *wow.Packet, func()) {
	t.Helper()

	c1, c2 := net.Pipe()
	sock := protocol.NewWoWSocket(c2)

	srv := &Server{
		log:       zerolog.Nop(),
		spawns:    NewSpawnManager(),
		vendorMgr: vendor.NewManager(nil),
	}

	bd := &basedata.Store{
		Items: []*basedata.ItemTemplate{
			{
				Entry:         117,
				Name:          "Tough Jerky",
				DisplayID:     1200,
				BuyPrice:      25,
				SellPrice:     5,
				Stackable:     20,
				BuyCount:      5,
				MaxDurability: 0,
			},
			{
				Entry:         159,
				Name:          "Refreshing Spring Water",
				DisplayID:     1201,
				BuyPrice:      25,
				SellPrice:     5,
				Stackable:     20,
				BuyCount:      5,
				MaxDurability: 0,
			},
			{
				Entry:         25,
				Name:          "Worn Shortsword",
				DisplayID:     1500,
				BuyPrice:      35,
				SellPrice:     7,
				Stackable:     1,
				MaxDurability: 20,
			},
			{
				Entry:         4540,
				Name:          "Tough Jerky (Alt)",
				DisplayID:     1200,
				BuyPrice:      25,
				SellPrice:     5,
				Stackable:     20,
				BuyCount:      5,
				MaxDurability: 0,
			},
		},
	}
	basedata.SetInstance(bd)
	srv.baseData = bd

	p := createTestPlayer()
	p.Inventory = player.NewInventory()
	p.Money = 1000

	ws := &WorldSession{
		log:    zerolog.Nop(),
		socket: sock,
		player: p,
		ws:     srv,
	}

	packetChan := make(chan *wow.Packet, 50)
	done := make(chan struct{})

	go func() {
		for {
			header := make([]byte, 4)
			if _, err := io.ReadFull(c1, header); err != nil {
				return
			}
			size := binary.BigEndian.Uint16(header[0:2])
			opcode := binary.LittleEndian.Uint16(header[2:4])
			dataSize := int(size) - 2
			data := make([]byte, dataSize)
			if dataSize > 0 {
				if _, err := io.ReadFull(c1, data); err != nil {
					return
				}
			}
			pkt := wow.NewPacket(wow.OpCode(opcode))
			pkt.WriteBytes(data)

			select {
			case packetChan <- pkt:
			case <-done:
				return
			}
		}
	}()

	cleanup := func() {
		close(done)
		_ = c1.Close()
		_ = c2.Close()
	}

	return ws, srv, packetChan, cleanup
}

func countEmptyBackpackSlots(inv *player.Inventory) int {
	empty := 0
	for i := player.InventorySlotItemStart; i < player.InventorySlotItemEnd; i++ {
		if inv.GetItem(i) == nil {
			empty++
		}
	}
	return empty
}

func readPacketTimeout(t *testing.T, ch chan *wow.Packet, timeout time.Duration) *wow.Packet {
	t.Helper()
	select {
	case pkt := <-ch:
		return pkt
	case <-time.After(timeout):
		t.Fatal("timed out waiting for packet")
		return nil
	}
}

func TestHandleListInventory(t *testing.T) {
	ws, srv, ch, cleanup := createVendorTestHarness(t)
	defer cleanup()

	// Spawn vendor NPC
	vendorSpawnID := uint64(5001)
	npc := NewNPC(54, "Test Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	// Send CMSG_LIST_INVENTORY
	pkt := wow.NewPacket(wow.ClientListInventory)
	_ = pkt.Write(uint64(vendorGUID))

	ws.HandleListInventory(pkt.Bytes())

	resp := readPacketTimeout(t, ch, time.Second)
	require.NotNil(t, resp)
	assert.Equal(t, wow.ServerListInventory, resp.Opcode())

	reader := wow.NewPacketReader(resp.Bytes())
	var guid uint64
	var count uint8
	require.NoError(t, reader.Read(&guid))
	require.NoError(t, reader.Read(&count))

	assert.Equal(t, uint64(vendorGUID), guid)
	assert.Greater(t, count, uint8(0))
}

func TestHandleBuyItem_Success(t *testing.T) {
	ws, srv, ch, cleanup := createVendorTestHarness(t)
	defer cleanup()

	vendorSpawnID := uint64(5002)
	npc := NewNPC(54, "Test Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	// Vendor item at slot 1 is item 117 (Tough Jerky, cost 25)
	ws.player.Money = 100

	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	// Send CMSG_BUY_ITEM: vendorGUID (u64), itemID (u32), slot (u32, 1-indexed), count (u32)
	pkt := wow.NewPacket(wow.ClientBuyItem)
	_ = pkt.Write(uint64(vendorGUID))
	_ = pkt.WriteUint32(117)
	_ = pkt.WriteUint32(1)
	_ = pkt.WriteUint32(1)

	ws.HandleBuyItem(pkt.Bytes())

	// Player money should be deducted: 100 - 25 = 75
	assert.Equal(t, uint32(75), ws.player.Money)

	// Packets received: SMSG_BUY_ITEM, SMSG_ITEM_PUSH_RESULT, SMSG_UPDATE_OBJECT...
	foundBuyItem := false
	foundPushResult := false

	for i := 0; i < 5; i++ {
		select {
		case p := <-ch:
			if p.Opcode() == wow.ServerBuyItem {
				foundBuyItem = true
			}
			if p.Opcode() == wow.ServerItemPushResult {
				foundPushResult = true
			}
		case <-time.After(200 * time.Millisecond):
		}
	}

	assert.True(t, foundBuyItem, "ServerBuyItem should have been sent")
	assert.True(t, foundPushResult, "ServerItemPushResult should have been sent")

	// Verify item was added to player's inventory
	emptyCount := countEmptyBackpackSlots(ws.player.Inventory)
	assert.Equal(t, 15, emptyCount) // 16 - 1 = 15
}

func TestHandleBuyItem_NotEnoughMoney(t *testing.T) {
	ws, srv, ch, cleanup := createVendorTestHarness(t)
	defer cleanup()

	vendorSpawnID := uint64(5003)
	npc := NewNPC(54, "Test Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	// Item 117 costs 25 copper; player has only 10 copper
	ws.player.Money = 10

	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	pkt := wow.NewPacket(wow.ClientBuyItem)
	_ = pkt.Write(uint64(vendorGUID))
	_ = pkt.WriteUint32(117)
	_ = pkt.WriteUint32(1)
	_ = pkt.WriteUint32(1)

	ws.HandleBuyItem(pkt.Bytes())

	// Money remains unchanged
	assert.Equal(t, uint32(10), ws.player.Money)

	resp := readPacketTimeout(t, ch, time.Second)
	require.NotNil(t, resp)
	assert.Equal(t, wow.ServerBuyFailed, resp.Opcode())

	reader := wow.NewPacketReader(resp.Bytes())
	var guid uint64
	var itemID uint32
	var reason uint8
	require.NoError(t, reader.Read(&guid))
	require.NoError(t, reader.Read(&itemID))
	require.NoError(t, reader.Read(&reason))

	assert.Equal(t, uint8(vendor.BuyErrNotEnoughMoney), reason)
}

func TestHandleBuyItem_LimitedStock(t *testing.T) {
	ws, srv, ch, cleanup := createVendorTestHarness(t)
	defer cleanup()

	vendorSpawnID := uint64(5004)
	npc := NewNPC(9001, "Limited Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	// Add limited stock item (1 in stock)
	srv.vendorMgr.AddVendorItem(9001, store.VendorItem{
		Entry:        9001,
		Slot:         1,
		Item:         25,
		MaxCount:     1,
		IncrTime:     3600,
		ExtendedCost: 0,
	})

	ws.player.Money = 1000
	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	// Buy the only 1 in stock
	pkt := wow.NewPacket(wow.ClientBuyItem)
	_ = pkt.Write(uint64(vendorGUID))
	_ = pkt.WriteUint32(25)
	_ = pkt.WriteUint32(1)
	_ = pkt.WriteUint32(1)
	ws.HandleBuyItem(pkt.Bytes())

	// Drain packets
	for i := 0; i < 5; i++ {
		select {
		case <-ch:
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Try to buy 1 more -> should fail with BuyErrItemAlreadySold
	pkt2 := wow.NewPacket(wow.ClientBuyItem)
	_ = pkt2.Write(uint64(vendorGUID))
	_ = pkt2.WriteUint32(25)
	_ = pkt2.WriteUint32(1)
	_ = pkt2.WriteUint32(1)
	ws.HandleBuyItem(pkt2.Bytes())

	resp := readPacketTimeout(t, ch, time.Second)
	require.NotNil(t, resp)
	assert.Equal(t, wow.ServerBuyFailed, resp.Opcode())

	reader := wow.NewPacketReader(resp.Bytes())
	var guid uint64
	var itemID uint32
	var reason uint8
	require.NoError(t, reader.Read(&guid))
	require.NoError(t, reader.Read(&itemID))
	require.NoError(t, reader.Read(&reason))

	assert.Equal(t, uint8(vendor.BuyErrItemAlreadySold), reason)
}

func TestHandleSellItem_FullStack(t *testing.T) {
	ws, srv, ch, cleanup := createVendorTestHarness(t)
	defer cleanup()

	vendorSpawnID := uint64(5005)
	npc := NewNPC(54, "Test Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	// Give player an item in backpack: item 25 (sell price 7 copper)
	item := player.NewItem(25, ws.player.GUID())
	item.StackCount = 1
	item.Durability = 20
	item.EnsureGUID()
	ws.player.Inventory.AddItem(item)

	ws.player.Money = 100
	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	// Send CMSG_SELL_ITEM: vendorGUID (u64), itemGUID (u64), count (u32)
	pkt := wow.NewPacket(wow.ClientSellItem)
	_ = pkt.Write(uint64(vendorGUID))
	_ = pkt.Write(uint64(item.GUID()))
	_ = pkt.WriteUint32(1)

	ws.HandleSellItem(pkt.Bytes())

	// Money should increase by 7: 100 + 7 = 107
	assert.Equal(t, uint32(107), ws.player.Money)

	// Item should be removed from inventory
	assert.Equal(t, 16, countEmptyBackpackSlots(ws.player.Inventory))

	// Item should be in buyback slot 0
	buybackItem := ws.player.GetItemFromBuyBackSlot(vendor.BuybackSlotStart)
	require.NotNil(t, buybackItem)
	assert.Equal(t, uint32(25), buybackItem.ItemEntry)
	assert.Equal(t, uint32(7), ws.player.GetBuybackPrice(vendor.BuybackSlotStart))

	// Drain update packets (no sell error packet sent on success)
	hasError := false
	for i := 0; i < 5; i++ {
		select {
		case p := <-ch:
			if p.Opcode() == wow.ServerSellItem {
				hasError = true
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	assert.False(t, hasError, "no sell error should be sent on successful sale")
}

func TestHandleSellItem_SplitStack(t *testing.T) {
	ws, srv, _, cleanup := createVendorTestHarness(t)
	defer cleanup()

	vendorSpawnID := uint64(5006)
	npc := NewNPC(54, "Test Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	// Give player a stack of 10 Tough Jerky (item 4540, sell price 5 copper each)
	item := player.NewItem(4540, ws.player.GUID())
	item.StackCount = 10
	item.EnsureGUID()
	ws.player.Inventory.AddItem(item)

	ws.player.Money = 0
	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	// Sell 4 from the stack
	pkt := wow.NewPacket(wow.ClientSellItem)
	_ = pkt.Write(uint64(vendorGUID))
	_ = pkt.Write(uint64(item.GUID()))
	_ = pkt.WriteUint32(4)

	ws.HandleSellItem(pkt.Bytes())

	// Money: 4 * 5 = 20 copper
	assert.Equal(t, uint32(20), ws.player.Money)

	// Original item stack should now be 6
	invItem := ws.player.Inventory.GetItem(player.InventorySlotItemStart)
	require.NotNil(t, invItem)
	assert.Equal(t, uint32(6), invItem.StackCount)

	// Buyback slot should hold 4
	bbItem := ws.player.GetItemFromBuyBackSlot(vendor.BuybackSlotStart)
	require.NotNil(t, bbItem)
	assert.Equal(t, uint32(4), bbItem.StackCount)
	assert.Equal(t, uint32(20), ws.player.GetBuybackPrice(vendor.BuybackSlotStart))
}

func TestHandleBuybackItem_Success(t *testing.T) {
	ws, srv, _, cleanup := createVendorTestHarness(t)
	defer cleanup()

	vendorSpawnID := uint64(5007)
	npc := NewNPC(54, "Test Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	// Put item directly into buyback slot
	item := player.NewItem(25, ws.player.GUID())
	item.StackCount = 1
	item.EnsureGUID()
	ws.player.AddItemToBuyBackSlot(item, 7)

	ws.player.Money = 50
	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	// Send CMSG_BUYBACK_ITEM: vendorGUID (u64), slot (u32 = 74)
	pkt := wow.NewPacket(wow.ClientBuybackItem)
	_ = pkt.Write(uint64(vendorGUID))
	_ = pkt.WriteUint32(vendor.BuybackSlotStart)

	ws.HandleBuybackItem(pkt.Bytes())

	// Money should decrease by 7: 50 - 7 = 43
	assert.Equal(t, uint32(43), ws.player.Money)

	// Buyback slot should now be empty
	assert.Nil(t, ws.player.GetItemFromBuyBackSlot(vendor.BuybackSlotStart))

	// Item should be in player's backpack
	assert.Equal(t, 15, countEmptyBackpackSlots(ws.player.Inventory))
}

func TestVendor_GossipInteraction(t *testing.T) {
	ws, srv, ch, cleanup := createVendorTestHarness(t)
	defer cleanup()

	vendorSpawnID := uint64(5008)
	npc := NewNPC(54, "Pure Vendor", 49, 1, 10, 500, 0, 0, 0, 0, 0, vendor.NpcFlagVendor)
	npc.SpawnID = vendorSpawnID
	srv.spawns.SpawnNPC(npc)

	vendorGUID := wow.NewGUID(wow.UnitGUID, uint32(vendorSpawnID))

	// 1. Right-click pure vendor: CMSG_QUESTGIVER_HELLO
	helloPkt := wow.NewPacket(wow.ClientQuestgiverHello)
	_ = helloPkt.Write(uint64(vendorGUID))

	ws.HandleQuestgiverHello(helloPkt.Bytes())

	// Pure vendor opens inventory directly: SMSG_LIST_INVENTORY
	resp := readPacketTimeout(t, ch, time.Second)
	require.NotNil(t, resp)
	assert.Equal(t, wow.ServerListInventory, resp.Opcode())

	// 2. Vendor + Questgiver NPC: right-click should send SMSG_GOSSIP_MESSAGE
	srv.questMgr = quest.NewManager(nil)
	srv.questMgr.AddQuestTemplate(&quest.Quest{
		ID:    101,
		Title: "Test Quest",
		Level: 5,
	})
	srv.questMgr.AddQuestStarter(54, 101)

	helloPkt2 := wow.NewPacket(wow.ClientQuestgiverHello)
	_ = helloPkt2.Write(uint64(vendorGUID))

	ws.HandleQuestgiverHello(helloPkt2.Bytes())

	gossipResp := readPacketTimeout(t, ch, time.Second)
	require.NotNil(t, gossipResp)
	assert.Equal(t, wow.ServerGossipMessage, gossipResp.Opcode())

	// 3. Selecting vendor gossip option sends SMSG_LIST_INVENTORY
	selPkt := wow.NewPacket(wow.ClientGossipSelectOption)
	_ = selPkt.Write(uint64(vendorGUID))
	_ = selPkt.WriteUint32(0)                          // menuID
	_ = selPkt.WriteUint32(int(vendor.GossipOptionVendor)) // optionID: vendor

	ws.HandleGossipSelectOption(selPkt.Bytes())

	listResp := readPacketTimeout(t, ch, time.Second)
	require.NotNil(t, listResp)
	assert.Equal(t, wow.ServerListInventory, listResp.Opcode())
}
