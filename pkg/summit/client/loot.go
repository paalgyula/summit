package client

import "github.com/paalgyula/summit/pkg/wow"

// LootItem is one lootable stack on a corpse or container.
type LootItem struct {
	Slot  uint8
	Entry uint32
	Count uint32
}

// LootInfo is the contents of an opened loot window (SMSG_LOOT_RESPONSE).
type LootInfo struct {
	GUID  wow.GUID
	Type  uint8
	Gold  uint32
	Items []LootItem
}

// LootOpened returns a channel of opened loot windows. The bot drains it to
// take the contents.
func (wc *WorldClient) LootOpened() <-chan LootInfo {
	return wc.lootCh
}

// CombatErrors returns a channel of "can't attack" style server messages.
func (wc *WorldClient) CombatErrors() <-chan string {
	return wc.combatErrors
}

func (wc *WorldClient) pushCombatError(msg string) {
	select {
	case wc.combatErrors <- msg:
	default:
	}
}

// handleLootResponse decodes SMSG_LOOT_RESPONSE into LootInfo.
func (wc *WorldClient) handleLootResponse(msg *ServerMessage) {
	r := msg.Reader()

	var guid uint64
	if err := r.Read(&guid); err != nil {
		return
	}

	var lootType uint8
	_ = r.Read(&lootType)

	var gold uint32
	_ = r.Read(&gold)

	info := LootInfo{GUID: wow.GUID(guid), Type: lootType, Gold: gold}

	var itemCount uint8
	_ = r.Read(&itemCount)

	for i := 0; i < int(itemCount); i++ {
		var (
			slot       uint8
			entry      uint32
			count      uint32
			displayID  uint32
			suffix     uint32
			randomProp int32
			slotType   uint8
		)
		_ = r.Read(&slot)
		_ = r.Read(&entry)
		_ = r.Read(&count)
		_ = r.Read(&displayID)
		_ = r.Read(&suffix)
		_ = r.Read(&randomProp)
		_ = r.Read(&slotType)

		info.Items = append(info.Items, LootItem{Slot: slot, Entry: entry, Count: count})
	}

	var questCount uint8
	_ = r.Read(&questCount)

	for i := 0; i < int(questCount); i++ {
		var (
			slot       uint8
			entry      uint32
			count      uint32
			displayID  uint32
			suffix     uint32
			randomProp int32
			slotType   uint8
		)
		_ = r.Read(&slot)
		_ = r.Read(&entry)
		_ = r.Read(&count)
		_ = r.Read(&displayID)
		_ = r.Read(&suffix)
		_ = r.Read(&randomProp)
		_ = r.Read(&slotType)

		info.Items = append(info.Items, LootItem{Slot: slot, Entry: entry, Count: count})
	}

	wc.lootMu.Lock()
	wc.lastLoot = &info
	wc.lootMu.Unlock()

	wc.log.Info().
		Uint64("loot", guid).
		Uint8("type", lootType).
		Uint32("gold", gold).
		Int("items", len(info.Items)).
		Msg("loot opened")

	select {
	case wc.lootCh <- info:
	default:
	}
}

// handleLootRemoved decodes SMSG_LOOT_REMOVED (u8 slot).
func (wc *WorldClient) handleLootRemoved(msg *ServerMessage) {
	r := msg.Reader()

	var slot uint8
	if err := r.Read(&slot); err != nil {
		return
	}

	wc.lootMu.Lock()
	if wc.lastLoot != nil {
		items := wc.lastLoot.Items[:0]
		for _, it := range wc.lastLoot.Items {
			if it.Slot != slot {
				items = append(items, it)
			}
		}
		wc.lastLoot.Items = items
	}
	wc.lootMu.Unlock()

	wc.log.Debug().Uint8("slot", slot).Msg("loot slot removed")
}

// handleLootClearMoney clears the gold from the current loot window.
func (wc *WorldClient) handleLootClearMoney(_ *ServerMessage) {
	wc.lootMu.Lock()
	if wc.lastLoot != nil {
		wc.lastLoot.Gold = 0
	}
	wc.lootMu.Unlock()
}

// handleLootReleaseResponse clears the tracked loot window.
func (wc *WorldClient) handleLootReleaseResponse(_ *ServerMessage) {
	wc.lootMu.Lock()
	wc.lastLoot = nil
	wc.lootMu.Unlock()
}

// handleAttackSwingError turns the SMSG_ATTACKSWING_* errors into combat
// messages for the bot.
func (wc *WorldClient) handleAttackSwingError(opcode wow.OpCode) {
	messages := map[wow.OpCode]string{
		wow.ServerAttackswingNotinrange: "out of range",
		wow.ServerAttackswingBadfacing:  "bad facing",
		wow.ServerAttackswingDeadtarget: "target is dead",
		wow.ServerAttackswingCantAttack: "cannot attack that target",
	}

	msg, ok := messages[opcode]
	if !ok {
		return
	}

	wc.log.Debug().Str("reason", msg).Msg("attack error")
	wc.pushCombatError(msg)
}
