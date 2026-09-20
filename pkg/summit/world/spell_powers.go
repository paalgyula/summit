package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// CalcSpellPowerCost calculates the power cost of a spell for a caster.
// Uses the DBC SpellInfo data — handles base cost + percentage cost.
// TODO: Apply spell power cost modifiers (Phase 6).
func CalcSpellPowerCost(spell *SpellInfo, caster *player.Player, school SpellSchoolMask) int32 {
	if spell == nil || caster == nil {
		return 0
	}

	cost := int32(spell.ManaCost)

	if cost == 0 && spell.ManaCostPercentage == 0 {
		return 0
	}

	// Apply mana cost percentage from base/max power
	if spell.ManaCostPercentage > 0 {
		pt := wow.PowerType(spell.PowerType)

		switch pt {
		case wow.PowerTypeMana:
			baseMana := caster.GetMaxPower(wow.PowerTypeMana)
			cost += int32(uint32(baseMana)*spell.ManaCostPercentage) / 100
		case wow.PowerTypeRage, wow.PowerTypeFocus, wow.PowerTypeEnergy, wow.PowerTypeHappiness:
			maxPower := caster.GetMaxPower(pt)
			cost += int32(uint32(maxPower)*spell.ManaCostPercentage) / 100
		}
	}

	if cost < 0 {
		cost = 0
	}

	return cost
}

// HasPowerForSpell checks if the player has enough power to cast the spell.
func HasPowerForSpell(spell *SpellInfo, caster *player.Player) bool {
	if spell == nil || caster == nil {
		return false
	}

	cost := CalcSpellPowerCost(spell, caster, SpellSchoolMask(spell.SchoolMask))
	if cost <= 0 {
		return true
	}

	pt := wow.PowerType(spell.PowerType)
	currentPower := caster.GetPower(pt)

	return currentPower >= uint32(cost)
}

// ConsumePowerForSpell deducts the power cost from the player.
func ConsumePowerForSpell(spell *SpellInfo, caster *player.Player) {
	if spell == nil || caster == nil {
		return
	}

	cost := CalcSpellPowerCost(spell, caster, SpellSchoolMask(spell.SchoolMask))
	if cost <= 0 {
		return
	}

	pt := wow.PowerType(spell.PowerType)
	currentPower := caster.GetPower(pt)
	if currentPower >= uint32(cost) {
		caster.SetPower(pt, currentPower-uint32(cost))
	}
}
