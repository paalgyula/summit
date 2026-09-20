package world

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// handlePeriodicDamage deals spell school damage to target each tick.
// Mirrors AC's AuraEffect::HandlePeriodicDamageAurasTick (simplified).
func handlePeriodicDamage(target, caster Unit, eff *AuraEffect) {
	if target == nil || !target.IsAlive() {
		return
	}

	if eff.Amount <= 0 {
		return
	}

	damage := uint32(eff.Amount)

	// Cap damage to target health
	health := target.GetHealth()
	if damage > health {
		damage = health
	}

	if damage == 0 {
		return
	}

	target.SetHealth(health - damage)
}

// handlePeriodicHeal heals the target each tick.
// Mirrors AC's AuraEffect::HandlePeriodicHealAurasTick (simplified).
func handlePeriodicHeal(target, caster Unit, eff *AuraEffect) {
	if target == nil || !target.IsAlive() {
		return
	}

	if eff.Amount <= 0 {
		return
	}

	heal := uint32(eff.Amount)

	health := target.GetHealth()
	maxHealth := target.GetMaxHealth()

	if health >= maxHealth {
		return
	}

	newHealth := health + heal
	if newHealth > maxHealth {
		newHealth = maxHealth
	}

	target.SetHealth(newHealth)
}

// handlePeriodicEnergize restores power to the target each tick.
// Mirrors AC's AuraEffect::HandlePeriodicEnergizeAuraTick (simplified).
func handlePeriodicEnergize(target, caster Unit, eff *AuraEffect) {
	if target == nil || !target.IsAlive() {
		return
	}

	if eff.Amount <= 0 {
		return
	}

	// Determine power type from MiscValue (EffectMiscValueN in DBC).
	// In AC, periodic energize stores the power type in MiscValue.
	powerType := wow.PowerTypeMana
	if eff.EffIndex < 3 && eff.SpellInfo != nil {
		powerType = wow.PowerType(eff.SpellInfo.Effects[eff.EffIndex].MiscValue)
	}

	currentPower := target.GetPower(powerType)
	maxPower := target.GetMaxPower(powerType)

	if currentPower >= maxPower {
		return
	}

	amount := uint32(eff.Amount)
	newPower := currentPower + amount
	if newPower > maxPower {
		newPower = maxPower
	}

	target.SetPower(powerType, newPower)
}

// handlePeriodicTriggerSpell triggers another spell each tick.
// Stub: in AC this looks up TriggerSpell from the effect data and casts it.
func handlePeriodicTriggerSpell(target, caster Unit, eff *AuraEffect) {
	// TODO: look up eff.SpellInfo.Effects[eff.EffIndex].TriggerSpell
	// and cast it on the target. Requires spell cast infrastructure.
	_ = target
	_ = caster
	_ = eff
}

// handlePeriodicManaLeech drains mana from the target and gives it to the caster.
// Mirrors AC's AuraEffect::HandlePeriodicManaLeechAuraTick (simplified).
func handlePeriodicManaLeech(target, caster Unit, eff *AuraEffect) {
	if caster == nil || !caster.IsAlive() || target == nil || !target.IsAlive() {
		return
	}

	if eff.Amount <= 0 {
		return
	}

	drainAmount := uint32(eff.Amount)

	// Determine power type from MiscValue
	powerType := wow.PowerTypeMana
	if eff.EffIndex < 3 && eff.SpellInfo != nil {
		powerType = wow.PowerType(eff.SpellInfo.Effects[eff.EffIndex].MiscValue)
	}

	// Drain from target
	targetPower := target.GetPower(powerType)
	drained := drainAmount
	if drained > targetPower {
		drained = targetPower
	}

	if drained == 0 {
		return
	}

	target.SetPower(powerType, targetPower-drained)

	// Give to caster (mana feed)
	casterPower := caster.GetPower(powerType)
	casterMaxPower := caster.GetMaxPower(powerType)
	newCasterPower := casterPower + drained
	if newCasterPower > casterMaxPower {
		newCasterPower = casterMaxPower
	}

	caster.SetPower(powerType, newCasterPower)
}
