package loot

import "math/rand"

// LootItemData holds the data needed to create a LootItem.
type LootItemData struct {
	ItemID       uint32
	Count        uint8
	MaxStackSize uint8 // 0 means 1 (non-stackable)
	NeedsQuest   bool
	GroupID      uint8
}

// LootItem represents a generated loot item ready for a player to pick up.
type LootItem struct {
	ItemID       uint32
	Count        uint8
	NeedsQuest   bool
	FreeForAll   bool
	IsLooted     bool
	IsBlocked    bool
	GroupID      uint8
}

// Loot holds all generated items and gold for a creature/GO/item instance.
type Loot struct {
	Items      []LootItem
	QuestItems []LootItem
	Gold       uint32
}

// NewLoot creates an empty loot instance.
func NewLoot() *Loot {
	return &Loot{}
}

// Empty returns true if there is nothing left to loot.
func (l *Loot) Empty() bool {
	if l.Gold > 0 {
		return false
	}

	for i := range l.Items {
		if !l.Items[i].IsLooted {
			return false
		}
	}

	for i := range l.QuestItems {
		if !l.QuestItems[i].IsLooted {
			return false
		}
	}

	return true
}

// AddItem adds an item to the loot, splitting into stacks if needed.
func (l *Loot) AddItem(data LootItemData) {
	if data.Count == 0 {
		data.Count = 1
	}

	maxStack := data.MaxStackSize
	if maxStack == 0 {
		maxStack = 1
	}

	remaining := data.Count

	for remaining > 0 {
		stackSize := remaining
		if stackSize > maxStack {
			stackSize = maxStack
		}

		item := LootItem{
			ItemID:     data.ItemID,
			Count:      stackSize,
			NeedsQuest: data.NeedsQuest,
			GroupID:    data.GroupID,
		}

		if data.NeedsQuest {
			l.QuestItems = append(l.QuestItems, item)
		} else {
			l.Items = append(l.Items, item)
		}

		remaining -= stackSize
	}
}

// GenerateMoneyLoot sets the gold amount from a min/max range.
func (l *Loot) GenerateMoneyLoot(minAmount, maxAmount uint32) {
	if maxAmount == 0 {
		return
	}

	if maxAmount <= minAmount {
		l.Gold = maxAmount

		return
	}

	l.Gold = minAmount + rand.Uint32()%(maxAmount-minAmount+1)
}

// FillLoot generates loot from the given store and loot ID.
// refStore is the reference_loot_template store (can be nil if no references).
func (l *Loot) FillLoot(lootID uint32, store *LootStore, mode LootMode, refStore ...*LootStore) error {
	tpl := store.GetLootFor(lootID)
	if tpl == nil {
		return nil
	}

	var refs *LootStore
	if len(refStore) > 0 {
		refs = refStore[0]
	}

	tpl.ProcessWithRefs(l, mode, refs)

	return nil
}
