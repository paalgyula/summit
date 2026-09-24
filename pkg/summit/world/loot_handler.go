package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/loot"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// lootSource tracks what a player is currently looting.
type lootSource struct {
	GUID    wow.GUID
	Loot    *loot.Loot
	LootType loot.LootType
}

// newLoot creates an empty loot with the item-template max-stack resolver wired
// in, so oversized drops are split into valid stacks instead of using the loot
// entry's drop count as if it were the item's stack size.
func newLoot() *loot.Loot {
	l := loot.NewLoot()
	l.MaxStack = func(itemID uint32) uint8 {
		tpl := basedata.GetInstance().LookupItem(itemID)
		if tpl == nil {
			return 0
		}

		if s := tpl.GetMaxStackSize(); s > 1 {
			if s > 255 {
				s = 255
			}

			return uint8(s)
		}

		return 0
	}

	return l
}

// HandleLoot handles CMSG_LOOT — player requests to loot a target.
func (gc *WorldSession) HandleLoot(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var targetGUID uint64
	if err := reader.Read(&targetGUID); err != nil {
		return
	}

	guid := wow.GUID(targetGUID)

	gc.log.Debug().
		Uint64("target", targetGUID).
		Msg("CMSG_LOOT")

	server, ok := gc.ws.(*Server)
	if !ok || server.lootMgr == nil {
		gc.sendLootError(guid, loot.ErrorDidntKill)
		return
	}

	// Determine what we're looting
	var lt loot.LootType

	l := newLoot()

	switch guid.High() {
	case wow.UnitGUID:
		// Creature corpse loot — check the lootable flag and roll the creature's
		// lootid from the loot manager.
		// Source: AzerothCore Player.cpp:8248-8255 (SendLoot checks DYNFLAG_LOOTABLE)
		npc := gc.getNPCByGUID(guid)
		if npc == nil || npc.DynamicFlags&UnitDynFlagLootable == 0 {
			gc.sendLootError(guid, loot.ErrorDidntKill)
			return
		}

		lt = loot.LootCorpse

		if !server.lootMgr.FillLoot(l, loot.StoreCreature, npc.LootID, loot.LootModeDefault) {
			gc.log.Warn().Uint32("entry", npc.EntryID).Uint32("lootId", npc.LootID).
				Msg("creature has no loot template")
			gc.sendLootError(guid, loot.ErrorDidntKill)

			return
		}

		l.GenerateMoneyLoot(npc.MinGold, npc.MaxGold)
	case wow.GameObjectGUID:
		lt = loot.LootCorpse

		lootID := gc.getGameObjectLootID(guid)
		if lootID == 0 || !server.lootMgr.FillLoot(l, loot.StoreGameObject, lootID, loot.LootModeDefault) {
			gc.sendLootError(guid, loot.ErrorDidntKill)

			return
		}
	default:
		gc.sendLootError(guid, loot.ErrorPlayerNotFound)

		return
	}

	// Store loot on the player
	gc.player.LootGUID = targetGUID
	gc.player.ActiveLoot = &lootSource{
		GUID:     guid,
		Loot:     l,
		LootType: lt,
	}

	// Send SMSG_LOOT_RESPONSE
	gc.sendLootResponse(guid, lt, l)
}

// HandleAutostoreLootItem handles CMSG_AUTOSTORE_LOOT_ITEM — player takes an item.
// This is called both by manual clicking and by the client's auto-loot feature
// (when the player has "Auto Loot" enabled in Interface settings).
//
// Source: AzerothCore LootHandler.cpp:33 (HandleAutostoreLootItemOpcode).
func (gc *WorldSession) HandleAutostoreLootItem(data wow.PacketData) {
	if gc.player == nil || gc.player.ActiveLoot == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var slot uint8
	if err := reader.Read(&slot); err != nil {
		return
	}

	src, ok := gc.player.ActiveLoot.(*lootSource)
	if !ok {
		return
	}

	l := src.Loot

	gc.log.Debug().
		Uint8("slot", slot).
		Msg("CMSG_AUTOSTORE_LOOT_ITEM")

	// Determine if this is a normal item or quest item slot
	normalCount := len(l.Items)
	questCount := len(l.QuestItems)

	var item *loot.LootItem
	var isQuestItem bool

	if int(slot) < normalCount {
		item = &l.Items[slot]
	} else if questSlot := int(slot) - normalCount; questSlot < questCount {
		item = &l.QuestItems[questSlot]
		isQuestItem = true
	} else {
		gc.log.Warn().Uint8("slot", slot).Msg("invalid loot slot")
		return
	}

	if item.IsLooted {
		return
	}

	// Create a new inventory item from the loot item
	newItem := player.NewItem(item.ItemID, gc.player.GUID())
	newItem.StackCount = uint32(item.Count)
	newItem.EnsureGUID()

	// Try to add to inventory (may merge into existing stacks and/or place a
	// new stack for the remainder)
	res := gc.player.Inventory.AddItem(newItem)
	if !res.Placed && len(res.StackedInto) == 0 {
		// Inventory full
		gc.log.Warn().Msg("inventory full, cannot loot item")

		return
	}

	// Mark as looted
	item.IsLooted = true

	// Send SMSG_LOOT_REMOVED to notify other looters
	gc.sendLootRemoved(slot)

	upd := &Updater{}

	// Existing stacks that grew need a values update so the client sees the new
	// stack count (a create block would be wrong - the object already exists).
	for _, stacked := range res.StackedInto {
		if pkt := upd.BuildItemValuesUpdate(stacked, gc.player); pkt != nil {
			gc.Send(pkt)
		}
	}

	// Only a newly placed stack gets a CreateObject block. A fully merged loot
	// item has no object of its own (StackCount 0, no slot), so creating one
	// would leave a phantom item on the client.
	if res.Placed {
		if pkt := upd.BuildItemCreateObject(newItem, gc.player); pkt != nil {
			gc.Send(pkt)
		}
	}

	// Update player's inventory fields
	gc.player.UpdateInventoryFields()

	_ = isQuestItem // TODO: quest item notification logic

	// If all items looted and no gold, auto-release
	if l.Empty() {
		gc.HandleLootRelease(nil)
	}
}

// HandleLootMoney handles CMSG_LOOT_MONEY — player takes gold.
func (gc *WorldSession) HandleLootMoney(data wow.PacketData) {
	if gc.player == nil || gc.player.ActiveLoot == nil {
		return
	}

	src, ok := gc.player.ActiveLoot.(*lootSource)
	if !ok {
		return
	}

	l := src.Loot

	if l.Gold == 0 {
		return
	}

	gold := l.Gold
	l.Gold = 0

	// TODO: split gold among group members if in a group
	gc.player.ModifyMoney(int32(gold))

	// Notify all looters
	gc.sendLootClearMoney()

	gc.log.Debug().Uint32("gold", gold).Msg("loot gold taken")
}

// HandleLootRelease handles CMSG_LOOT_RELEASE — player closes loot window.
func (gc *WorldSession) HandleLootRelease(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	if gc.player.ActiveLoot == nil {
		return
	}

	src, ok := gc.player.ActiveLoot.(*lootSource)
	if !ok {
		return
	}

	// Send SMSG_LOOT_RELEASE_RESPONSE
	pkt := wow.NewPacket(wow.ServerLootReleaseResponse)
	_ = pkt.Write(src.GUID)
	_ = pkt.WriteOne(1) // always 1
	gc.Send(pkt)

	// Clear lootable flag on the creature corpse
	// Source: AzerothCore LootHandler.cpp:447-454 (DoLootRelease)
	if src.Loot != nil && src.Loot.Empty() {
		gc.clearLootableFlag(src.GUID)
	}

	// Clear loot state
	gc.player.LootGUID = 0
	gc.player.ActiveLoot = nil

	gc.log.Debug().Msg("loot released")
}

// sendLootResponse builds and sends SMSG_LOOT_RESPONSE.
func (gc *WorldSession) sendLootResponse(guid wow.GUID, lt loot.LootType, l *loot.Loot) {
	pkt := wow.NewPacket(wow.ServerLootResponse)

	_ = pkt.Write(guid)
	_ = pkt.WriteOne(int(lt))

	// Gold
	_ = pkt.Write(l.Gold)

	// Items
	_ = pkt.WriteOne(len(l.Items))

	for i := range l.Items {
		item := &l.Items[i]
		_ = pkt.WriteOne(i)                // slot index
		_ = pkt.Write(item.ItemID)         // item entry
		_ = pkt.Write(uint32(item.Count))  // count
		_ = pkt.Write(uint32(0))           // display info ID (TODO: look up)
		_ = pkt.Write(uint32(0))           // random suffix
		_ = pkt.Write(int32(0))            // random property ID
		_ = pkt.WriteOne(int(loot.SlotAllowLoot)) // slot type
	}

	// Quest items
	_ = pkt.WriteOne(len(l.QuestItems))

	for i := range l.QuestItems {
		item := &l.QuestItems[i]
		slotIdx := len(l.Items) + i
		_ = pkt.WriteOne(slotIdx)           // slot index (after normal items)
		_ = pkt.Write(item.ItemID)
		_ = pkt.Write(uint32(item.Count))
		_ = pkt.Write(uint32(0))            // display info ID
		_ = pkt.Write(uint32(0))            // random suffix
		_ = pkt.Write(int32(0))             // random property ID
		_ = pkt.WriteOne(int(loot.SlotAllowLoot))
	}

	gc.Send(pkt)
}

// sendLootError sends a loot error response.
func (gc *WorldSession) sendLootError(guid wow.GUID, err loot.LootError) {
	pkt := wow.NewPacket(wow.ServerLootResponse)
	_ = pkt.Write(guid)
	_ = pkt.WriteOne(int(loot.LootNone))
	_ = pkt.WriteOne(int(err))
	gc.Send(pkt)
}

// sendLootRemoved sends SMSG_LOOT_REMOVED for a slot.
func (gc *WorldSession) sendLootRemoved(slot uint8) {
	pkt := wow.NewPacket(wow.ServerLootRemoved)
	_ = pkt.WriteOne(int(slot))
	gc.Send(pkt)
}

// sendLootClearMoney sends SMSG_LOOT_CLEAR_MONEY.
func (gc *WorldSession) sendLootClearMoney() {
	pkt := wow.NewPacket(wow.ServerLootClearMoney)
	gc.Send(pkt)
}

// getGameObjectLootID returns the loot ID for a game object GUID.
func (gc *WorldSession) getGameObjectLootID(guid wow.GUID) uint32 {
	bd := basedata.GetInstance()
	if bd == nil {
		return 0
	}

	spawn := bd.LookupGameObjectSpawn(guid.Counter())
	if spawn == nil {
		return 0
	}

	tpl := bd.LookupGameObjectTemplate(spawn.Entry)
	if tpl == nil {
		return 0
	}

	return tpl.GetLootID()
}

// clearLootableFlag clears UNIT_DYNFLAG_LOOTABLE on a creature corpse.
// Called when loot is fully exhausted or the player releases the loot window.
// Source: AzerothCore LootHandler.cpp:447-454, Creature.cpp:1334.
func (gc *WorldSession) clearLootableFlag(guid wow.GUID) {
	server, ok := gc.ws.(*Server)
	if !ok || server.spawns == nil {
		return
	}

	npc := server.spawns.GetNPC(uint64(guid.Counter()))
	if npc == nil {
		return
	}

	npc.DynamicFlags &^= UnitDynFlagLootable
	npc.Object.SetUInt32Value(object.UnitDynamicFlags, npc.DynamicFlags)
}
