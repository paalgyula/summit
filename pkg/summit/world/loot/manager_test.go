package loot

import "testing"

func TestManagerFillLootResolvesReference(t *testing.T) {
	m := NewManager()

	// creature loot 1: guaranteed item 100 plus a reference to 9000.
	m.Add(StoreCreature, LootEntry{Entry: 1, Item: 100, Chance: 100, MinCount: 1, MaxCount: 1})
	m.Add(StoreCreature, LootEntry{Entry: 1, Item: 9000, Ref: -9000, Chance: 100, MaxCount: 1})
	// reference 9000: guaranteed item 200.
	m.Add(StoreReference, LootEntry{Entry: 9000, Item: 200, Chance: 100, MinCount: 1, MaxCount: 1})

	if m.Count(StoreCreature) != 1 {
		t.Fatalf("Count(StoreCreature) = %d, want 1", m.Count(StoreCreature))
	}

	l := NewLoot()
	if !m.FillLoot(l, StoreCreature, 1, LootModeDefault) {
		t.Fatal("FillLoot returned false for a known loot id")
	}

	got := map[uint32]bool{}
	for _, it := range l.Items {
		got[it.ItemID] = true
	}

	if !got[100] || !got[200] {
		t.Fatalf("expected items 100 and 200, got %v", got)
	}
}

func TestManagerQuestRequired(t *testing.T) {
	m := NewManager()
	m.Add(StoreCreature, LootEntry{Entry: 7, Item: 42, Chance: 100, NeedQuest: true, MinCount: 1, MaxCount: 1})

	l := NewLoot()
	if !m.FillLoot(l, StoreCreature, 7, LootModeDefault) {
		t.Fatal("FillLoot returned false")
	}

	if len(l.QuestItems) != 1 || l.QuestItems[0].ItemID != 42 {
		t.Fatalf("expected quest item 42, got %+v", l.QuestItems)
	}
}

func TestManagerMissingLoot(t *testing.T) {
	m := NewManager()

	if m.FillLoot(NewLoot(), StoreCreature, 12345, LootModeDefault) {
		t.Fatal("FillLoot should return false for an unknown loot id")
	}

	if m.HasLoot(StoreCreature, 12345) {
		t.Fatal("HasLoot should be false for an unknown loot id")
	}

	if m.GetLootFor(StoreCreature, 12345) != nil {
		t.Fatal("GetLootFor should be nil for an unknown loot id")
	}
}

func TestStoreTypeString(t *testing.T) {
	cases := map[StoreType]string{
		StoreCreature:   "creature",
		StoreGameObject: "gameobject",
		StoreItem:       "item",
		StoreReference:  "reference",
		StoreType(99):   "unknown",
	}

	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("StoreType(%d).String() = %q, want %q", in, got, want)
		}
	}
}
