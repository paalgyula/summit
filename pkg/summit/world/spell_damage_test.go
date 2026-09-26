package world

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDealSpellDamage_NormalDamage_NoKill(t *testing.T) {
	p := createTestPlayer()
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 100, 0, 0, 0, 0, 1, 0)
	npc.MinGold = 10
	npc.MaxGold = 20

	dmg := DealSpellDamage(p, npc, 30)

	assert.Equal(t, uint32(30), dmg)
	assert.Equal(t, uint32(70), npc.GetHealth())
	assert.True(t, npc.IsAlive())
	assert.Equal(t, uint32(0), npc.DynamicFlags&UnitDynFlagLootable, "corpse should not be lootable while alive")
}

func TestDealSpellDamage_FatalDamage_KillsCreature_SetsLootable(t *testing.T) {
	p := createTestPlayer()
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 100, 0, 0, 0, 0, 1, 0)
	npc.MinGold = 10
	npc.MaxGold = 20

	dmg := DealSpellDamage(p, npc, 120)

	assert.Equal(t, uint32(120), dmg)
	assert.Equal(t, uint32(0), npc.GetHealth())
	assert.False(t, npc.IsAlive())
	assert.NotEqual(t, uint32(0), npc.DynamicFlags&UnitDynFlagLootable, "corpse must be lootable when killed by spell")
}

func TestSpellEffectSchoolDamage_DirectCast_KillsCreature(t *testing.T) {
	p := createTestPlayer()
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 50, 0, 0, 0, 0, 1, 0)
	npc.MinGold = 10
	npc.MaxGold = 20

	spellInfo := &SpellInfo{
		Id: 133, // Fireball
		Effects: [3]SpellEffectInfo{
			{
				Effect:     uint32(SpellEffectSchoolDamage),
				BasePoints: 50,
			},
		},
	}

	spell := NewSpell(p, spellInfo, TriggeredNone)
	require.NotNil(t, spell)
	spell.Targets.UnitTarget = npc

	spell.effectSchoolDamage(0, npc)

	assert.Equal(t, uint32(0), npc.GetHealth())
	assert.False(t, npc.IsAlive())
	assert.NotEqual(t, uint32(0), npc.DynamicFlags&UnitDynFlagLootable, "direct spell kill must set lootable flag")
}

func TestHandlePeriodicDamage_Tick_KillsCreature(t *testing.T) {
	p := createTestPlayer()
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 25, 0, 0, 0, 0, 1, 0)
	npc.MinGold = 10
	npc.MaxGold = 20

	eff := &AuraEffect{
		Amount: 30,
	}

	handlePeriodicDamage(npc, p, eff)

	assert.Equal(t, uint32(0), npc.GetHealth())
	assert.False(t, npc.IsAlive())
	assert.NotEqual(t, uint32(0), npc.DynamicFlags&UnitDynFlagLootable, "DoT tick kill must set lootable flag")
}

func TestSpellEffectInstantKill_KillsCreature(t *testing.T) {
	p := createTestPlayer()
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 500, 0, 0, 0, 0, 1, 0)
	npc.MinGold = 10
	npc.MaxGold = 20

	spellInfo := &SpellInfo{
		Id: 20484,
		Effects: [3]SpellEffectInfo{
			{
				Effect: uint32(SpellEffectInstantKill),
			},
		},
	}

	spell := NewSpell(p, spellInfo, TriggeredNone)
	require.NotNil(t, spell)

	spell.effectInstantKill(0, npc)

	assert.Equal(t, uint32(0), npc.GetHealth())
	assert.False(t, npc.IsAlive())
	assert.NotEqual(t, uint32(0), npc.DynamicFlags&UnitDynFlagLootable, "instant kill must set lootable flag")
}

