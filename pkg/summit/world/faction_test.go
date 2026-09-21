package world_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFactionManager creates a FactionManager with test data matching AC's
// typical faction layout:
// - Faction 1 = Alliance (template IDs 1, 2)
// - Faction 2 = Horde (template IDs 3, 4)
// - Faction 3 = Monster (template ID 5)
// - Faction 14 = Stormwind (template ID 11)
// - Faction 67 = Orgrimmar (template ID 12)
func testFactionManager() *world.FactionManager {
	fm := world.NewFactionManager()

	templates := []*wotlk.FactionTemplateEntry{
		// Alliance base (template 1)
		{
			ID:           1,
			Faction:      1,
			FactionFlags: 0,
			OurMask:      world.FactionMaskAlliance,
			FriendlyMask: world.FactionMaskAlliance | world.FactionMaskPlayer,
			HostileMask:  world.FactionMaskHorde | world.FactionMaskMonster,
		},
		// Alliance player (template 2)
		{
			ID:           2,
			Faction:      1,
			FactionFlags: 0,
			OurMask:      world.FactionMaskAlliance,
			FriendlyMask: world.FactionMaskAlliance,
			HostileMask:  world.FactionMaskHorde,
		},
		// Horde base (template 3)
		{
			ID:           3,
			Faction:      2,
			FactionFlags: 0,
			OurMask:      world.FactionMaskHorde,
			FriendlyMask: world.FactionMaskHorde | world.FactionMaskPlayer,
			HostileMask:  world.FactionMaskAlliance | world.FactionMaskMonster,
		},
		// Horde player (template 4)
		{
			ID:           4,
			Faction:      2,
			FactionFlags: 0,
			OurMask:      world.FactionMaskHorde,
			FriendlyMask: world.FactionMaskHorde,
			HostileMask:  world.FactionMaskAlliance,
		},
		// Monster (template 5)
		{
			ID:           5,
			Faction:      3,
			FactionFlags: 0,
			OurMask:      world.FactionMaskMonster,
			FriendlyMask: world.FactionMaskMonster,
			HostileMask:  world.FactionMaskAlliance | world.FactionMaskHorde | world.FactionMaskPlayer,
		},
		// Stormwind (template 11) - enemy of Orgrimmar
		{
			ID:           11,
			Faction:      14,
			FactionFlags: 0,
			OurMask:      world.FactionMaskAlliance,
			FriendlyMask: world.FactionMaskAlliance,
			HostileMask:  world.FactionMaskHorde,
			Field6:       12, // enemy: Orgrimmar
		},
		// Orgrimmar (template 12) - enemy of Stormwind
		{
			ID:           12,
			Faction:      67,
			FactionFlags: 0,
			OurMask:      world.FactionMaskHorde,
			FriendlyMask: world.FactionMaskHorde,
			HostileMask:  world.FactionMaskAlliance,
			Field6:       11, // enemy: Stormwind
		},
		// Friendly NPC (template 20) - friendly to Alliance
		{
			ID:           20,
			Faction:      100,
			FactionFlags: 0,
			OurMask:      0,
			FriendlyMask: 0,
			HostileMask:  0,
			Field10:      1, // friend: Alliance
		},
		// Hostile NPC (template 21) - enemy of Horde
		{
			ID:           21,
			Faction:      101,
			FactionFlags: 0,
			OurMask:      0,
			FriendlyMask: 0,
			HostileMask:  0,
			Field6:       2, // enemy: Horde faction ID
		},
	}

	fm.LoadFromDBC(templates)
	return fm
}

func TestFactionManager_LoadFromDBC(t *testing.T) {
	fm := world.NewFactionManager()
	assert.False(t, fm.IsLoaded())

	fm.LoadFromDBC([]*wotlk.FactionTemplateEntry{
		{ID: 1, Faction: 1},
		{ID: 2, Faction: 2},
	})

	assert.True(t, fm.IsLoaded())
	assert.NotNil(t, fm.GetTemplate(1))
	assert.NotNil(t, fm.GetTemplate(2))
	assert.Nil(t, fm.GetTemplate(999))
}

func TestFactionManager_SameFactionIsFriendly(t *testing.T) {
	fm := testFactionManager()

	// Same template ID
	rank := fm.GetReactionTo(1, 1)
	assert.Equal(t, world.RepFriendly, rank)

	// Same faction base (different templates)
	rank = fm.GetReactionTo(1, 2) // Alliance base vs Alliance player
	assert.Equal(t, world.RepFriendly, rank)
}

func TestFactionManager_AllianceVsHorde(t *testing.T) {
	fm := testFactionManager()

	// Alliance vs Horde (mask-based)
	rank := fm.GetReactionTo(1, 3) // Alliance base vs Horde base
	assert.Equal(t, world.RepHostile, rank)

	// Horde vs Alliance
	rank = fm.GetReactionTo(3, 1)
	assert.Equal(t, world.RepHostile, rank)
}

func TestFactionManager_AllianceVsMonster(t *testing.T) {
	fm := testFactionManager()

	// Alliance vs Monster
	rank := fm.GetReactionTo(1, 5)
	assert.Equal(t, world.RepHostile, rank)

	// Monster vs Alliance
	rank = fm.GetReactionTo(5, 1)
	assert.Equal(t, world.RepHostile, rank)
}

func TestFactionManager_HordeVsMonster(t *testing.T) {
	fm := testFactionManager()

	// Horde vs Monster
	rank := fm.GetReactionTo(3, 5)
	assert.Equal(t, world.RepHostile, rank)

	// Monster vs Horde
	rank = fm.GetReactionTo(5, 3)
	assert.Equal(t, world.RepHostile, rank)
}

func TestFactionManager_StormwindVsOrgrimmar(t *testing.T) {
	fm := testFactionManager()

	// Stormwind vs Orgrimmar (explicit enemy list)
	rank := fm.GetReactionTo(11, 12)
	assert.Equal(t, world.RepHostile, rank)

	// Orgrimmar vs Stormwind
	rank = fm.GetReactionTo(12, 11)
	assert.Equal(t, world.RepHostile, rank)
}

func TestFactionManager_FriendlyNPC(t *testing.T) {
	fm := testFactionManager()

	// Friendly NPC vs Alliance (friend list)
	rank := fm.GetReactionTo(20, 1)
	assert.Equal(t, world.RepFriendly, rank)

	// Alliance vs Friendly NPC (reverse)
	rank = fm.GetReactionTo(1, 20)
	// Alliance's HostileMask doesn't include NPC's OurMask (0), so neutral or friendly
	assert.True(t, rank >= world.RepNeutral)
}

func TestFactionManager_NeutralFactions(t *testing.T) {
	fm := testFactionManager()

	// Two unknown factions should be neutral
	rank := fm.GetReactionTo(999, 888)
	assert.Equal(t, world.RepNeutral, rank)

	// Known vs unknown
	rank = fm.GetReactionTo(1, 999)
	assert.Equal(t, world.RepNeutral, rank)
}

func TestFactionManager_IsHostileTo(t *testing.T) {
	fm := testFactionManager()

	assert.True(t, fm.IsHostileTo(1, 3))   // Alliance vs Horde
	assert.True(t, fm.IsHostileTo(3, 1))   // Horde vs Alliance
	assert.True(t, fm.IsHostileTo(1, 5))   // Alliance vs Monster
	assert.False(t, fm.IsHostileTo(1, 1))  // Alliance vs Alliance
	assert.False(t, fm.IsHostileTo(1, 2))  // Alliance vs Alliance player
}

func TestFactionManager_IsFriendlyTo(t *testing.T) {
	fm := testFactionManager()

	assert.True(t, fm.IsFriendlyTo(1, 1))  // Same faction
	assert.True(t, fm.IsFriendlyTo(1, 2))  // Alliance vs Alliance player
	assert.True(t, fm.IsFriendlyTo(3, 4))  // Horde vs Horde player
	assert.False(t, fm.IsFriendlyTo(1, 3)) // Alliance vs Horde
	assert.False(t, fm.IsFriendlyTo(1, 5)) // Alliance vs Monster
}

func TestGetEnemyFactions(t *testing.T) {
	tests := []struct {
		name     string
		entry    *wotlk.FactionTemplateEntry
		expected []uint32
	}{
		{
			name:     "nil entry",
			entry:    nil,
			expected: nil,
		},
		{
			name: "no enemies",
			entry: &wotlk.FactionTemplateEntry{
				Field6: 0, Field7: 0, Field8: 0, Field9: 0,
			},
			expected: nil,
		},
		{
			name: "one enemy",
			entry: &wotlk.FactionTemplateEntry{
				Field6: 100, Field7: 0, Field8: 0, Field9: 0,
			},
			expected: []uint32{100},
		},
		{
			name: "multiple enemies",
			entry: &wotlk.FactionTemplateEntry{
				Field6: 100, Field7: 200, Field8: 300, Field9: 0,
			},
			expected: []uint32{100, 200, 300},
		},
		{
			name: "all enemies",
			entry: &wotlk.FactionTemplateEntry{
				Field6: 100, Field7: 200, Field8: 300, Field9: 400,
			},
			expected: []uint32{100, 200, 300, 400},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := world.GetEnemyFactions(tt.entry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFriendFactions(t *testing.T) {
	tests := []struct {
		name     string
		entry    *wotlk.FactionTemplateEntry
		expected []uint32
	}{
		{
			name:     "nil entry",
			entry:    nil,
			expected: nil,
		},
		{
			name: "no friends",
			entry: &wotlk.FactionTemplateEntry{
				Field10: 0, Field11: 0, Field12: 0, Field13: 0,
			},
			expected: nil,
		},
		{
			name: "one friend",
			entry: &wotlk.FactionTemplateEntry{
				Field10: 100, Field11: 0, Field12: 0, Field13: 0,
			},
			expected: []uint32{100},
		},
		{
			name: "multiple friends",
			entry: &wotlk.FactionTemplateEntry{
				Field10: 100, Field11: 200, Field12: 300, Field13: 0,
			},
			expected: []uint32{100, 200, 300},
		},
		{
			name: "all friends",
			entry: &wotlk.FactionTemplateEntry{
				Field10: 100, Field11: 200, Field12: 300, Field13: 400,
			},
			expected: []uint32{100, 200, 300, 400},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := world.GetFriendFactions(tt.entry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPlayerFaction(t *testing.T) {
	tests := []struct {
		race     uint32
		expected uint32
	}{
		{1, 1},  // Human → Alliance
		{3, 1},  // Dwarf → Alliance
		{4, 1},  // NightElf → Alliance
		{7, 1},  // Gnome → Alliance
		{11, 1}, // Draenei → Alliance
		{2, 2},  // Orc → Horde
		{5, 2},  // Undead → Horde
		{6, 2},  // Tauren → Horde
		{8, 2},  // Troll → Horde
		{10, 2}, // BloodElf → Horde
		{22, 2}, // Goblin → Horde
		{0, 0},  // Unknown → none
		{99, 0}, // Unknown → none
	}

	for _, tt := range tests {
		t.Run("race_"+string(rune(tt.race+'0')), func(t *testing.T) {
			result := world.GetPlayerFaction(tt.race)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFactionManager_GetTemplate(t *testing.T) {
	fm := testFactionManager()

	// Existing template
	tpl := fm.GetTemplate(1)
	require.NotNil(t, tpl)
	assert.Equal(t, uint32(1), tpl.Faction)

	// Non-existing template
	tpl = fm.GetTemplate(9999)
	assert.Nil(t, tpl)
}

func TestReputationRank_Constants(t *testing.T) {
	// Verify rank ordering
	assert.True(t, world.RepHated < world.RepHostile)
	assert.True(t, world.RepHostile < world.RepUnfriendly)
	assert.True(t, world.RepUnfriendly < world.RepNeutral)
	assert.True(t, world.RepNeutral < world.RepFriendly)
	assert.True(t, world.RepFriendly < world.RepHonored)
	assert.True(t, world.RepHonored < world.RepRevered)
	assert.True(t, world.RepRevered < world.RepExalted)
}

func TestFactionMask_Constants(t *testing.T) {
	// Verify team masks don't overlap each other
	assert.Equal(t, uint32(0), world.FactionMaskAlliance&world.FactionMaskHorde)
	assert.Equal(t, uint32(0), world.FactionMaskAlliance&world.FactionMaskMonster)
	assert.Equal(t, uint32(0), world.FactionMaskHorde&world.FactionMaskMonster)
	// Player mask is separate (bit 0), team masks are bits 1-3
	assert.Equal(t, uint32(1), world.FactionMaskPlayer)
	assert.Equal(t, uint32(2), world.FactionMaskAlliance)
	assert.Equal(t, uint32(4), world.FactionMaskHorde)
	assert.Equal(t, uint32(8), world.FactionMaskMonster)
}
