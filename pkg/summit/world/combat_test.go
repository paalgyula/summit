package world

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	basedata.SetInstance(&basedata.Store{
		Items: []*basedata.ItemTemplate{
			{
				Entry:         2361,
				Class:         2,
				SubClass:      5,
				Name:          "Battleworn Hammer",
				DisplayID:     8690,
				Quality:       1,
				InventoryType: wow.InventoryType2hweapon,
				Delay:         2900,
				Damage: [2]basedata.ItemDamage{
					{Min: 3, Max: 5, Type: 0},
				},
			},
		},
	})
}

// setupBattlewornHammerPlayer creates a player with Battleworn Hammer (Item 2361: 3-5 dmg, 2.9s speed)
// equipped in the mainhand slot.
func setupBattlewornHammerPlayer(t *testing.T, race wow.PlayerRace, class wow.PlayerClass) *player.Player {
	t.Helper()

	p := player.NewPlayer()
	p.ID = 100
	p.Name = "TestWarrior"
	p.Race = race
	p.Class = class
	p.Level = 1
	p.Init()

	// Item 2361: Battleworn Hammer (3-5 damage, 2.9 speed)
	hammer := player.NewItem(2361, p.GUID())
	require.NotNil(t, hammer)

	p.EquipItem(hammer)
	return p
}

// TestMeleeDamage_BattlewornHammer_Equipped verifies that equipping the Battleworn Hammer
// (Item 2361: 3-5 damage, 2.9s speed) correctly sets weapon damage and speed on the player,
// and deals more than 1 damage on a level 1 enemy.
func TestMeleeDamage_BattlewornHammer_Equipped(t *testing.T) {
	// Ensure basedata has item 2361
	tpl := basedata.GetInstance().LookupItem(2361)
	require.NotNil(t, tpl, "Battleworn Hammer item template must exist")
	assert.Equal(t, float32(3), tpl.Damage[0].Min)
	assert.Equal(t, float32(5), tpl.Damage[0].Max)
	assert.Equal(t, uint32(2900), tpl.Delay)

	// Create level 1 Tauren Warrior (matches user's character "Boocie")
	p := setupBattlewornHammerPlayer(t, wow.RaceTauren, wow.ClassWarior)

	// Verify weapon damage and speed resolved from inventory item
	minDmg, maxDmg := p.GetWeaponDamage(0)
	assert.Equal(t, uint32(3), minDmg, "min weapon damage should be 3")
	assert.Equal(t, uint32(5), maxDmg, "max weapon damage should be 5")

	speed := p.GetWeaponSpeed(0)
	assert.Equal(t, 2900*time.Millisecond, speed, "weapon speed should be 2.9s (2900ms)")

	// Tauren Warrior starting stats: Str 28. Base AP = 1*3 + 28*2 - 20 = 39.
	assert.Equal(t, uint32(39), p.GetAttackPower(), "attack power should be 39")

	// Target: Level 1 NPC enemy (has 30 armor)
	enemy := NewNPC(1001, "Level 1 Enemy", 49, 14, 1, 100, 0, 0, 0, 0, 0, 0)
	require.NotNil(t, enemy)

	// Run multiple damage calculations to verify range and ensure damage is > 1 on hit
	for i := 0; i < 50; i++ {
		dmgInfo := CalculateMeleeDamage(p, enemy, BaseAttack)
		require.NotNil(t, dmgInfo)

		// On a normal or crit hit, damage MUST be well above 1 (expected ~10-13 normal, ~20-26 crit)
		if dmgInfo.HitOutcome == MeleeHitNormal || dmgInfo.HitOutcome == MeleeHitCrit {
			damage := dmgInfo.Damages[0].Damage
			assert.Greater(t, damage, uint32(1), "damage must be greater than 1 per hit on lvl 1 enemy")
			assert.GreaterOrEqual(t, damage, uint32(8), "damage should be at least 8 with AP bonus")
		}
	}
}

// TestMeleeDamage_HumanPaladin_BattlewornHammer verifies damage for a level 1 Paladin
// wielding the Battleworn Hammer against a level 1 enemy.
func TestMeleeDamage_HumanPaladin_BattlewornHammer(t *testing.T) {
	p := setupBattlewornHammerPlayer(t, wow.RaceHuman, wow.ClassPaladin)

	// Human Paladin starting stats: Str 22. Base AP = 1*3 + 22*2 - 20 = 27.
	assert.Equal(t, uint32(27), p.GetAttackPower(), "Paladin AP should be 27")

	minDmg, maxDmg := p.GetWeaponDamage(0)
	assert.Equal(t, uint32(3), minDmg)
	assert.Equal(t, uint32(5), maxDmg)

	// AP contribution for 2.9s weapon with 27 AP:
	// AP_bonus = (27 / 14.0) * 2.9 = 5.59
	// Damage before armor = min (3 + 5.59 = 8.59), max (5 + 5.59 = 10.59)
	enemy := NewNPC(1001, "Kobold Vermin", 49, 14, 1, 100, 0, 0, 0, 0, 0, 0)

	for i := 0; i < 50; i++ {
		dmgInfo := CalculateMeleeDamage(p, enemy, BaseAttack)
		if dmgInfo.HitOutcome == MeleeHitNormal {
			damage := dmgInfo.Damages[0].Damage
			assert.Greater(t, damage, uint32(1), "Paladin must deal more than 1 damage per hit on lvl 1 enemy")
			assert.GreaterOrEqual(t, damage, uint32(7), "normal damage should be in range [7, 12]")
			assert.LessOrEqual(t, damage, uint32(12), "normal damage should be in range [7, 12]")
		}
	}
}

// TestMeleeDamage_ZeroAP_RawWeaponDamage verifies that even with 0 AP, the 3-5 damage
// mace deals 3 to 5 damage (minus slight armor mitigation), and never truncates to 1.
func TestMeleeDamage_ZeroAP_RawWeaponDamage(t *testing.T) {
	p := setupBattlewornHammerPlayer(t, wow.RaceHuman, wow.ClassPaladin)
	p.AttackPower = 0 // Force 0 AP

	enemy := NewNPC(1001, "Test Mob", 49, 14, 1, 100, 0, 0, 0, 0, 0, 0)

	for i := 0; i < 50; i++ {
		dmgInfo := CalculateMeleeDamage(p, enemy, BaseAttack)
		if dmgInfo.HitOutcome == MeleeHitNormal {
			damage := dmgInfo.Damages[0].Damage
			assert.Greater(t, damage, uint32(1), "raw weapon damage (3-5) on lvl 1 enemy must be > 1")
			assert.GreaterOrEqual(t, damage, uint32(2))
			assert.LessOrEqual(t, damage, uint32(5))
		}
	}
}

// TestMeleeDamage_UnarmedFallback verifies that when unarmed, the player falls back
// to 1-2 damage and 2000ms attack speed.
func TestMeleeDamage_UnarmedFallback(t *testing.T) {
	p := player.NewPlayer()
	p.ID = 101
	p.Name = "UnarmedPlayer"
	p.Race = wow.RaceHuman
	p.Class = wow.ClassWarior
	p.Level = 1
	p.Init()

	minDmg, maxDmg := p.GetWeaponDamage(0)
	assert.Equal(t, uint32(1), minDmg, "unarmed min damage should be 1")
	assert.Equal(t, uint32(2), maxDmg, "unarmed max damage should be 2")

	speed := p.GetWeaponSpeed(0)
	assert.Equal(t, 2000*time.Millisecond, speed, "unarmed speed should be 2.0s")
}

// TestMeleeDamage_EquipAndUnequip verifies that equipping and unequipping weapon
// dynamically switches between weapon damage/speed and unarmed values.
func TestMeleeDamage_EquipAndUnequip(t *testing.T) {
	p := player.NewPlayer()
	p.ID = 102
	p.Name = "Swapper"
	p.Race = wow.RaceHuman
	p.Class = wow.ClassPaladin
	p.Level = 1
	p.Init()

	// Initial unarmed
	assert.Equal(t, 2000*time.Millisecond, p.GetWeaponSpeed(0))
	minDmg, maxDmg := p.GetWeaponDamage(0)
	assert.Equal(t, uint32(1), minDmg)
	assert.Equal(t, uint32(2), maxDmg)

	// Equip Battleworn Hammer
	hammer := player.NewItem(2361, p.GUID())
	p.EquipItem(hammer)

	assert.Equal(t, 2900*time.Millisecond, p.GetWeaponSpeed(0))
	minDmg, maxDmg = p.GetWeaponDamage(0)
	assert.Equal(t, uint32(3), minDmg)
	assert.Equal(t, uint32(5), maxDmg)

	// Unequip weapon
	p.UnequipItem(player.EquipmentSlotMainHand)

	assert.Equal(t, 2000*time.Millisecond, p.GetWeaponSpeed(0))
	minDmg, maxDmg = p.GetWeaponDamage(0)
	assert.Equal(t, uint32(1), minDmg)
	assert.Equal(t, uint32(2), maxDmg)
}

// TestAttackPowerCalculation_Classes verifies level 1 base AP for various classes.
// Reference: TrinityCore StatSystem.cpp:340 (Player::UpdateAttackPowerAndDamage)
func TestAttackPowerCalculation_Classes(t *testing.T) {
	tests := []struct {
		name       string
		race       wow.PlayerRace
		class      wow.PlayerClass
		expectedAP uint32
	}{
		{"Tauren Warrior", wow.RaceTauren, wow.ClassWarior, 39}, // Str 28: 1*3 + 28*2 - 20 = 39
		{"Human Warrior", wow.RaceHuman, wow.ClassWarior, 29},   // Str 23: 1*3 + 23*2 - 20 = 29
		{"Human Paladin", wow.RaceHuman, wow.ClassPaladin, 27},  // Str 22: 1*3 + 22*2 - 20 = 27
		{"NightElf Rogue", wow.RaceNightElf, wow.ClassRogue, 28}, // Str 18, Agi 28: 1*2 + 18 + 28 - 20 = 28
		{"Human Mage", wow.RaceHuman, wow.ClassMage, 10},        // Str 20: 20 - 10 = 10
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := player.NewPlayer()
			p.Race = tt.race
			p.Class = tt.class
			p.Level = 1
			p.Init()

			assert.Equal(t, tt.expectedAP, p.GetAttackPower())
		})
	}
}

// TestScheduleNextSwing_RespectsWeaponSpeed verifies that swing scheduling uses
// the equipped weapon's speed (2900ms) rather than the default 2000ms.
func TestScheduleNextSwing_RespectsWeaponSpeed(t *testing.T) {
	p := setupBattlewornHammerPlayer(t, wow.RaceTauren, wow.ClassWarior)
	p.AttackState = AttackStateSwinging

	gc := &WorldSession{
		player: p,
	}

	now := time.Now()
	gc.scheduleNextSwing(now)

	expectedNext := now.UnixMilli() + 2900
	assert.Equal(t, expectedNext, p.NextAttackTime, "next attack time should be now + 2900ms")
}
