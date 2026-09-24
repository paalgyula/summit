package loot

import (
	"math/rand"
)

// LootStore holds all loot templates for a specific table (creature, GO, item, etc.).
type LootStore struct {
	name      string
	templates map[uint32]*LootTemplate
}

// NewLootStore creates a new empty loot store.
func NewLootStore(name string) *LootStore {
	return &LootStore{
		name:      name,
		templates: make(map[uint32]*LootTemplate),
	}
}

// HasLootFor returns true if the store has a template for the given entry.
func (s *LootStore) HasLootFor(entry uint32) bool {
	_, ok := s.templates[entry]
	return ok
}

// GetLootFor returns the template for the given entry, or nil.
func (s *LootStore) GetLootFor(entry uint32) *LootTemplate {
	return s.templates[entry]
}

// AddEntry adds a loot entry to the store, creating the template if needed.
func (s *LootStore) AddEntry(entry LootEntry) {
	tpl, ok := s.templates[entry.Entry]
	if !ok {
		tpl = &LootTemplate{}
		s.templates[entry.Entry] = tpl
	}

	tpl.AddEntry(entry)
}

// LootTemplate processes loot entries for a single loot ID.
type LootTemplate struct {
	entries []LootEntry     // ungrouped entries
	groups  [128]*LootGroup // grouped entries, indexed by groupid-1
}

// AddEntry adds a loot entry to the template.
func (t *LootTemplate) AddEntry(entry LootEntry) {
	if entry.GroupID > 0 {
		idx := entry.GroupID - 1
		if idx >= uint8(len(t.groups)) {
			return
		}

		if t.groups[idx] == nil {
			t.groups[idx] = &LootGroup{}
		}

		t.groups[idx].AddEntry(entry)
	} else {
		t.entries = append(t.entries, entry)
	}
}

// Process rolls all entries and adds resulting items to the loot.
func (t *LootTemplate) Process(loot *Loot, mode LootMode) {
	t.ProcessWithRefs(loot, mode, nil)
}

// ProcessWithRefs rolls all entries, resolving references against refStore.
func (t *LootTemplate) ProcessWithRefs(loot *Loot, mode LootMode, refStore *LootStore) {
	// Ungrouped entries: each rolls independently
	for _, entry := range t.entries {
		if entry.LootMode != 0 && !entry.LootMode.Has(mode) {
			continue
		}

		if !rollChance(entry.Chance) {
			continue
		}

		if entry.Ref != 0 {
			// Reference to another loot template
			refID := uint32(-entry.Ref)
			if refStore != nil {
				refTpl := refStore.GetLootFor(refID)
				if refTpl != nil {
					multiplier := entry.MaxCount
					if multiplier == 0 {
						multiplier = 1
					}

					for i := uint8(0); i < multiplier; i++ {
						refTpl.Process(loot, mode)
					}
				}
			}

			continue
		}

		count := rollCount(entry.MinCount, entry.MaxCount)
		loot.AddItem(LootItemData{
			ItemID:     entry.Item,
			Count:      count,
			NeedsQuest: entry.NeedQuest,
			GroupID:    entry.GroupID,
		})
	}

	// Grouped entries: each group drops exactly one item
	for _, group := range t.groups {
		if group == nil {
			continue
		}

		group.Process(loot, mode, refStore)
	}
}

// LootGroup represents a group of loot entries where only one item drops.
type LootGroup struct {
	explicitlyChanced []*LootEntry // entries with chance > 0
	equalChanced      []*LootEntry // entries with chance == 0 (equal probability)
}

// AddEntry adds an entry to the group.
func (g *LootGroup) AddEntry(entry LootEntry) {
	e := entry // copy

	if entry.Chance > 0 {
		g.explicitlyChanced = append(g.explicitlyChanced, &e)
	} else {
		g.equalChanced = append(g.equalChanced, &e)
	}
}

// Process rolls one item from this group.
func (g *LootGroup) Process(loot *Loot, mode LootMode, refStore *LootStore) {
	// Try explicitly chanced items first
	if len(g.explicitlyChanced) > 0 {
		roll := rand.Float32() * 100.0

		for _, entry := range g.explicitlyChanced {
			if entry.LootMode != 0 && !entry.LootMode.Has(mode) {
				continue
			}

			if roll < entry.Chance {
				if entry.Ref != 0 {
					refID := uint32(-entry.Ref)
					if refStore != nil {
						refTpl := refStore.GetLootFor(refID)
						if refTpl != nil {
							refTpl.Process(loot, mode)
						}
					}
				} else {
					count := rollCount(entry.MinCount, entry.MaxCount)
					loot.AddItem(LootItemData{
						ItemID:     entry.Item,
						Count:      count,
						NeedsQuest: entry.NeedQuest,
						GroupID:    entry.GroupID,
					})
				}

				return
			}

			roll -= entry.Chance
		}
	}

	// Fall back to equal-chanced items
	if len(g.equalChanced) > 0 {
		// Filter valid entries
		var valid []*LootEntry

		for _, entry := range g.equalChanced {
			if entry.LootMode == 0 || entry.LootMode.Has(mode) {
				valid = append(valid, entry)
			}
		}

		if len(valid) > 0 {
			entry := valid[rand.Intn(len(valid))]

			if entry.Ref != 0 {
				refID := uint32(-entry.Ref)
				if refStore != nil {
					refTpl := refStore.GetLootFor(refID)
					if refTpl != nil {
						refTpl.Process(loot, mode)
					}
				}
			} else {
				count := rollCount(entry.MinCount, entry.MaxCount)
				loot.AddItem(LootItemData{
					ItemID:     entry.Item,
					Count:      count,
					NeedsQuest: entry.NeedQuest,
					GroupID:    entry.GroupID,
				})
			}
		}
	}
}

// rollChance returns true based on the given percentage.
// 100% always succeeds, 0% always fails.
func rollChance(chance float32) bool {
	if chance >= 100.0 {
		return true
	}

	if chance <= 0.0 {
		return true // equal-chanced items always "pass" at top level
	}

	return rand.Float32()*100.0 < chance
}

// rollCount returns a random count between min and max (inclusive).
func rollCount(min, max uint8) uint8 {
	if min == 0 {
		min = 1
	}

	if max <= min {
		return min
	}

	return min + uint8(rand.Intn(int(max-min+1)))
}
