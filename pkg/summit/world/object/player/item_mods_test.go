package player_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

func TestApplyItemMods_Stats(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Helmet has +10 str, +20 sta (entry 1003)
	helmet := player.NewItem(1003, p.GUID())
	p.Inventory.SetEquipment(player.EquipmentSlotHead, helmet)

	// Read initial stats
	initialStr := p.Object.GetUInt32Value(object.UnitFieldStat0) // strength
	initialSta := p.Object.GetUInt32Value(object.UnitFieldStat2) // stamina

	// Apply mods
	p.ApplyItemMods(helmet, player.EquipmentSlotHead, true)

	// Check stats increased
	newStr := p.Object.GetUInt32Value(object.UnitFieldStat0)
	newSta := p.Object.GetUInt32Value(object.UnitFieldStat2)

	assert.Equal(t, initialStr+10, newStr, "strength should increase by 10")
	assert.Equal(t, initialSta+20, newSta, "stamina should increase by 20")

	// Remove mods
	p.ApplyItemMods(helmet, player.EquipmentSlotHead, false)

	// Check stats restored
	finalStr := p.Object.GetUInt32Value(object.UnitFieldStat0)
	finalSta := p.Object.GetUInt32Value(object.UnitFieldStat2)

	assert.Equal(t, initialStr, finalStr, "strength should be restored")
	assert.Equal(t, initialSta, finalSta, "stamina should be restored")
}

func TestApplyItemMods_Armor(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Shield has 100 armor (entry 1002)
	shield := player.NewItem(1002, p.GUID())
	p.Inventory.SetEquipment(player.EquipmentSlotOffHand, shield)

	// Read initial armor (at UnitFieldResistances + 0 = armor)
	initialArmor := p.Object.GetUInt32Value(object.UnitFieldResistances)

	// Apply mods
	p.ApplyItemMods(shield, player.EquipmentSlotOffHand, true)

	newArmor := p.Object.GetUInt32Value(object.UnitFieldResistances)
	assert.Equal(t, initialArmor+100, newArmor, "armor should increase by 100")

	// Remove mods
	p.ApplyItemMods(shield, player.EquipmentSlotOffHand, false)

	finalArmor := p.Object.GetUInt32Value(object.UnitFieldResistances)
	assert.Equal(t, initialArmor, finalArmor, "armor should be restored")
}

func TestApplyItemMods_Resistances(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Cloak has 10 fire res, 5 frost res (entry 1012)
	cloak := player.NewItem(1012, p.GUID())
	p.Inventory.SetEquipment(player.EquipmentSlotBack, cloak)

	// Read initial resistances
	initialFire := p.Object.GetUInt32Value(object.UnitFieldResistances + 2) // fire
	initialFrost := p.Object.GetUInt32Value(object.UnitFieldResistances + 4) // frost

	// Apply mods
	p.ApplyItemMods(cloak, player.EquipmentSlotBack, true)

	newFire := p.Object.GetUInt32Value(object.UnitFieldResistances + 2)
	newFrost := p.Object.GetUInt32Value(object.UnitFieldResistances + 4)

	assert.Equal(t, initialFire+10, newFire, "fire res should increase by 10")
	assert.Equal(t, initialFrost+5, newFrost, "frost res should increase by 5")

	// Remove mods
	p.ApplyItemMods(cloak, player.EquipmentSlotBack, false)

	finalFire := p.Object.GetUInt32Value(object.UnitFieldResistances + 2)
	finalFrost := p.Object.GetUInt32Value(object.UnitFieldResistances + 4)

	assert.Equal(t, initialFire, finalFire, "fire res should be restored")
	assert.Equal(t, initialFrost, finalFrost, "frost res should be restored")
}

func TestApplyItemMods_WeaponDamage(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Sword has 10-20 damage (entry 1001)
	sword := player.NewItem(1001, p.GUID())
	p.Inventory.SetEquipment(player.EquipmentSlotMainHand, sword)

	// Apply mods
	p.ApplyItemMods(sword, player.EquipmentSlotMainHand, true)

	minDmg := p.Object.GetFloatValue(object.UnitFieldMindamage)
	maxDmg := p.Object.GetFloatValue(object.UnitFieldMaxdamage)

	assert.Equal(t, float32(10), minDmg, "min damage should be 10")
	assert.Equal(t, float32(20), maxDmg, "max damage should be 20")

	// Remove mods
	p.ApplyItemMods(sword, player.EquipmentSlotMainHand, false)

	minDmg = p.Object.GetFloatValue(object.UnitFieldMindamage)
	maxDmg = p.Object.GetFloatValue(object.UnitFieldMaxdamage)

	assert.Equal(t, float32(0), minDmg, "min damage should be 0")
	assert.Equal(t, float32(0), maxDmg, "max damage should be 0")
}

func TestApplyItemMods_NilItem(t *testing.T) {
	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	// Should not panic
	p.ApplyItemMods(nil, player.EquipmentSlotHead, true)
}

func TestApplyItemMods_InvalidSlot(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	item := player.NewItem(1001, p.GUID())

	// Should not panic with invalid slot
	p.ApplyItemMods(item, -1, true)
	p.ApplyItemMods(item, 100, true)
}

func TestApplyItemMods_NoTemplate(t *testing.T) {
	basedata.SetInstance(testBaseData())
	defer basedata.SetInstance(nil)

	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)
	item := player.NewItem(9999, p.GUID()) // unknown item

	// Should not panic
	p.ApplyItemMods(item, player.EquipmentSlotHead, true)
}

func TestModifyStat(t *testing.T) {
	p := testPlayer(80, wow.ClassWarior, wow.RaceHuman)

	initial := p.Object.GetUInt32Value(object.UnitFieldStat0)
	p.ModifyStat(object.UnitFieldStat0, 50)
	newVal := p.Object.GetUInt32Value(object.UnitFieldStat0)
	assert.Equal(t, initial+50, newVal)

	p.ModifyStat(object.UnitFieldStat0, -20)
	final := p.Object.GetUInt32Value(object.UnitFieldStat0)
	assert.Equal(t, initial+30, final)
}
