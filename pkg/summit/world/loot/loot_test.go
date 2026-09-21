package loot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollChance_AlwaysDrop(t *testing.T) {
	assert.True(t, rollChance(100.0))
	assert.True(t, rollChance(200.0))
}

func TestRollChance_NeverDrop(t *testing.T) {
	// 0 chance means equal-chanced (always passes at top level)
	assert.True(t, rollChance(0.0))
}

func TestRollChance_Probabilistic(t *testing.T) {
	// 50% chance — run many times, should get roughly half
	passed := 0
	n := 10000

	for i := 0; i < n; i++ {
		if rollChance(50.0) {
			passed++
		}
	}

	ratio := float64(passed) / float64(n)
	assert.InDelta(t, 0.5, ratio, 0.05, "50%% chance should pass ~50%% of the time")
}

func TestRollCount_MinEqualsMax(t *testing.T) {
	assert.Equal(t, uint8(5), rollCount(5, 5))
}

func TestRollCount_Range(t *testing.T) {
	for i := 0; i < 100; i++ {
		c := rollCount(1, 10)
		assert.True(t, c >= 1 && c <= 10)
	}
}

func TestRollCount_MinZero(t *testing.T) {
	// MinCount=0 should be treated as 1
	c := rollCount(0, 5)
	assert.True(t, c >= 1 && c <= 5)
}

func TestLootStore_IndexMultipleEntries(t *testing.T) {
	store := NewLootStore("test")
	store.AddEntry(LootEntry{Entry: 1, Item: 100, Chance: 100, MinCount: 1, MaxCount: 1})
	store.AddEntry(LootEntry{Entry: 1, Item: 101, Chance: 50, MinCount: 1, MaxCount: 1})
	store.AddEntry(LootEntry{Entry: 2, Item: 200, Chance: 100, MinCount: 1, MaxCount: 1})

	tpl := store.GetLootFor(1)
	require.NotNil(t, tpl)
	assert.Len(t, tpl.entries, 2)

	tpl = store.GetLootFor(2)
	require.NotNil(t, tpl)
	assert.Len(t, tpl.entries, 1)
}

func TestLootTemplate_GroupedDropsExactlyOne(t *testing.T) {
	tpl := &LootTemplate{}
	tpl.AddEntry(LootEntry{Entry: 1, Item: 100, Chance: 100, GroupID: 1, MinCount: 1, MaxCount: 1})
	tpl.AddEntry(LootEntry{Entry: 1, Item: 101, Chance: 100, GroupID: 1, MinCount: 1, MaxCount: 1})
	tpl.AddEntry(LootEntry{Entry: 1, Item: 102, Chance: 100, GroupID: 1, MinCount: 1, MaxCount: 1})

	// Run 100 times — should always get exactly 1 item from the group
	for i := 0; i < 100; i++ {
		loot := NewLoot()
		tpl.Process(loot, LootModeDefault)
		require.Len(t, loot.Items, 1, "group should drop exactly one item")
	}
}

func TestLootTemplate_MixedGroupedAndUngrouped(t *testing.T) {
	tpl := &LootTemplate{}
	// Ungrouped guaranteed
	tpl.AddEntry(LootEntry{Entry: 1, Item: 100, Chance: 100, MinCount: 1, MaxCount: 1})
	// Group 1: two items, one drops
	tpl.AddEntry(LootEntry{Entry: 1, Item: 200, Chance: 0, GroupID: 1, MinCount: 1, MaxCount: 1})
	tpl.AddEntry(LootEntry{Entry: 1, Item: 201, Chance: 0, GroupID: 1, MinCount: 1, MaxCount: 1})

	loot := NewLoot()
	tpl.Process(loot, LootModeDefault)

	// 1 ungrouped + 1 from group = 2 items
	assert.Len(t, loot.Items, 2)
	assert.Equal(t, uint32(100), loot.Items[0].ItemID)
}

func TestLootTemplate_ExplicitlyChancedGroup(t *testing.T) {
	tpl := &LootTemplate{}
	tpl.AddEntry(LootEntry{Entry: 1, Item: 100, Chance: 100, GroupID: 1, MinCount: 1, MaxCount: 1})
	tpl.AddEntry(LootEntry{Entry: 1, Item: 101, Chance: 0, GroupID: 1, MinCount: 1, MaxCount: 1})

	// Item 100 has 100% chance — should always drop
	for i := 0; i < 50; i++ {
		loot := NewLoot()
		tpl.Process(loot, LootModeDefault)
		require.Len(t, loot.Items, 1)
		assert.Equal(t, uint32(100), loot.Items[0].ItemID)
	}
}

func TestLoot_FillLootWithReference(t *testing.T) {
	// Creature loot drops a reference to a "common goods" table
	creatureStore := NewLootStore("creature")
	creatureStore.AddEntry(LootEntry{Entry: 100, Item: 0, Ref: -5000, Chance: 100, MinCount: 1, MaxCount: 2})

	// Reference table has some items
	refStore := NewLootStore("reference")
	refStore.AddEntry(LootEntry{Entry: 5000, Item: 2589, Chance: 100, MinCount: 1, MaxCount: 1})

	loot := NewLoot()
	err := loot.FillLoot(100, creatureStore, LootModeDefault, refStore)
	require.NoError(t, err)

	// MaxCount=2 means reference is processed 2 times
	assert.Len(t, loot.Items, 2)
}
