package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/loot"
	"github.com/paalgyula/summit/pkg/wow"
)

// lootSource tracks what a player is currently looting.
type lootSource struct {
	GUID    wow.GUID
	Loot    *loot.Loot
	LootType loot.LootType
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

	// Determine what we're looting
	var lt loot.LootType
	var lootID uint32

	switch guid.High() {
	case wow.UnitGUID:
		// Creature corpse loot — TODO: look up creature's lootId from creature_template
		lt = loot.LootCorpse
		lootID = 0 // placeholder until creature loot integration
	case wow.GameObjectGUID:
		lt = loot.LootCorpse
		lootID = gc.getGameObjectLootID(guid)
	default:
		gc.sendLootError(guid, loot.ErrorPlayerNotFound)
		return
	}

	if lootID == 0 {
		gc.sendLootError(guid, loot.ErrorDidntKill)
		return
	}

	// Build loot from template
	l := loot.NewLoot()

	bd := basedata.GetInstance()
	if bd == nil {
		gc.sendLootError(guid, loot.ErrorDidntKill)
		return
	}

	var refStore *loot.LootStore
	if len(bd.ReferenceLootEntries) > 0 {
		refStore = loot.NewLootStore("reference")
		for _, e := range bd.ReferenceLootEntries {
			refStore.AddEntry(*e)
		}
	}

	_ = l.FillLoot(lootID, gc.getLootStore(lt), loot.LootModeDefault, refStore)

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

	if int(slot) >= len(l.Items) {
		gc.log.Warn().Uint8("slot", slot).Msg("invalid loot slot")
		return
	}

	item := &l.Items[slot]
	if item.IsLooted {
		return
	}

	// TODO: validate player can loot this slot (permission check)
	// TODO: check inventory space, call player.AddItem
	// For now, just mark as looted
	item.IsLooted = true

	// Notify all looters that this slot was removed
	gc.sendLootRemoved(slot)

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

// getLootStore returns the appropriate LootStore for the given loot type.
func (gc *WorldSession) getLootStore(lt loot.LootType) *loot.LootStore {
	bd := basedata.GetInstance()
	if bd == nil {
		return loot.NewLootStore("empty")
	}

	store := loot.NewLootStore("runtime")

	switch lt {
	case loot.LootCorpse:
		// Load creature loot entries into the store
		for _, entries := range bd.CreatureLoots {
			for _, e := range entries {
				store.AddEntry(e)
			}
		}
		// Also load gameobject loot (for GO loot)
		for _, entries := range bd.GameObjectLoots {
			for _, e := range entries {
				store.AddEntry(e)
			}
		}
	}

	return store
}
