package world

import (
	"math"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// SpellCastTargets holds the explicit target data for a spell cast.
type SpellCastTargets struct {
	UnitTarget       Unit
	GameObjectTarget Unit
	ItemTarget       interface{}
	SrcPosition      Position
	DstPosition      Position
	TargetMask       uint32
}

// Position represents a world location.
type Position struct {
	X, Y, Z, O float32
	MapID       uint32
}

// TargetInfo stores per-target damage/heal results for a spell.
type TargetInfo struct {
	Target     Unit
	Damage     int32
	HealAmount int32
	Absorb     int32
	Resist     int32
	MissInfo   uint32
	Crit       bool
}

// Spell is the execution engine for a single spell cast.
// Maps to AzerothCore's Spell class from Spell.h/Spell.cpp.
type Spell struct {
	Info         *SpellInfo
	Caster       *player.Player
	CastItem     interface{}
	OriginalCaster *player.Player
	TriggerFlags TriggerCastFlags
	Targets      SpellCastTargets

	// State
	State        SpellState
	CastTimeLeft int32
	PowerCost    int32
	SchoolMask   SpellSchoolMask

	// Target lists
	UniqueTargetInfo   []TargetInfo
	UniqueGOTargetInfo []TargetInfo

	// Effects
	EffectDamage [3]int32
	EffectHeal   [3]int32

	// Internal
	needComboPoints      bool
	triggeredByAuraSpell *SpellInfo
}

// NewSpell creates a new Spell execution engine.
func NewSpell(caster *player.Player, info *SpellInfo, flags TriggerCastFlags) *Spell {
	if caster == nil || info == nil {
		return nil
	}

	return &Spell{
		Info:         info,
		Caster:       caster,
		OriginalCaster: caster,
		TriggerFlags: flags,
		State:        SpellStateNone,
		SchoolMask:   info.GetSchoolMask(),
	}
}

// Prepare starts the spell cast (AC's Spell::prepare).
// Returns SpellCastSuccess on success, or an error code.
func (s *Spell) Prepare(targets *SpellCastTargets) SpellCastResult {
	if targets != nil {
		s.Targets = *targets
	}

	s.State = SpellStateCasting

	// Calculate power cost
	if !s.TriggerFlags.isSet(TriggeredIgnorePowerCostReagents) {
		s.PowerCost = CalcSpellPowerCost(s.Info, s.Caster, s.SchoolMask)
	}

	// Check if spell can be cast
	result := s.CheckCast(true)
	if result != SpellCastSuccess {
		s.Finish(false)
		return result
	}

	// Calculate cast time
	if s.TriggerFlags.isSet(TriggeredCastDirectly) {
		s.CastTimeLeft = 0
	} else {
		s.CastTimeLeft = int32(s.Info.CalcCastTime(s.Caster))
	}

	// Instant spells cast immediately
	if s.CastTimeLeft <= 0 {
		s.Cast(false)
	}

	return SpellCastSuccess
}

// CheckCast validates whether the spell can be cast (AC's Spell::CheckCast).
// Returns SpellCastSuccess or an error code.
func (s *Spell) CheckCast(strict bool) SpellCastResult {
	// Check if caster is alive (unless spell can be cast while dead)
	if !s.TriggerFlags.isSet(TriggeredTriggered) && !s.Caster.IsAlive() {
		if !s.Info.HasAttribute(SpellAttr0Passive) {
			return SpellCastFailedCasterIsDead
		}
	}

	// Check cooldown
	if strict && !s.TriggerFlags.isSet(TriggeredIgnoreCooldowns) {
		if IsSpellOnCooldown(s.Caster, s.Info.Id) {
			return SpellCastFailedSpellOnCooldown
		}
	}

	// Check power
	if !s.TriggerFlags.isSet(TriggeredIgnoreSpellCost) {
		if !HasPowerForSpell(s.Info, s.Caster) {
			return SpellCastNoMana
		}
	}

	// Check range (only for targeted spells)
	if s.Targets.UnitTarget != nil {
		maxRange := s.Info.GetMaxRange(false, s.Caster)
		if maxRange > 0 {
			casterLoc := s.CasterLocation()
			targetLoc := s.targetLocation()
			dist := distance2D(casterLoc, targetLoc)
			if dist > maxRange {
				return SpellCastTargetTooFar
			}
		}
	}

	// Check explicit target
	if s.Targets.UnitTarget != nil {
		result := s.Info.CheckExplicitTarget(s.Caster, s.Targets.UnitTarget)
		if result != SpellCastSuccess {
			return result
		}
	}

	return SpellCastSuccess
}

// Cast applies the spell effects (AC's Spell::cast).
// skipCheck: if true, skip re-validation (for instant spells already checked in Prepare).
func (s *Spell) Cast(skipCheck bool) {
	if !skipCheck {
		result := s.CheckCast(false)
		if result != SpellCastSuccess {
			s.Finish(false)
			return
		}
	}

	s.State = SpellStateCastTime

	// Consume power cost
	if !s.TriggerFlags.isSet(TriggeredIgnorePowerCostReagents) && !s.TriggerFlags.isSet(TriggeredIgnoreSpellCost) {
		ConsumePowerForSpell(s.Info, s.Caster)
	}

	// Handle immediate effects
	s.handleEffects()

	// Finish the spell
	s.Finish(true)
}

// Finish cleans up after cast (AC's Spell::finish).
func (s *Spell) Finish(success bool) {
	if s.State == SpellStateFinished {
		return
	}

	s.State = SpellStateFinished

	if !success {
		return
	}

	// Add cooldowns
	s.applyCooldowns()
}

// Update reduces cast time and triggers Cast when ready.
// Returns true if the spell is still casting, false if finished or cancelled.
func (s *Spell) Update(ms int32) bool {
	if s.State != SpellStateCasting {
		return false
	}

	s.CastTimeLeft -= ms
	if s.CastTimeLeft <= 0 {
		s.CastTimeLeft = 0
		s.Cast(false)
		return false
	}

	return true
}

// IsTriggered returns true if the spell was triggered (not player-initiated).
func (s *Spell) IsTriggered() bool {
	return s.TriggerFlags.isSet(TriggeredTriggered)
}

// IsAutoRepeat returns true if the spell is an auto-repeat ranged spell.
func (s *Spell) IsAutoRepeat() bool {
	return s.Info.IsAutoRepeat()
}

// --- internal helpers ---

// handleEffects processes all spell effects on targets.
func (s *Spell) handleEffects() {
	for i := 0; i < 3; i++ {
		effect := s.Info.GetEffect(i)
		if effect == nil {
			continue
		}

		target := s.Targets.UnitTarget

		switch SpellEffect(effect.Effect) {
		case SpellEffectSchoolDamage:
			s.effectSchoolDamage(i, target)
		case SpellEffectHeal:
			s.effectHeal(i, target)
		case SpellEffectEnergize:
			s.effectEnergize(i, target)
		case SpellEffectApplyAura:
			s.effectApplyAura(i, target)
		case SpellEffectTriggerSpell:
			s.effectTriggerSpell(i, target)
		case SpellEffectDummy:
			s.effectDummy(i, target)
		}
	}
}

// effectSchoolDamage applies direct spell damage to the target.
func (s *Spell) effectSchoolDamage(idx int, target Unit) {
	if target == nil {
		return
	}

	effect := s.Info.GetEffect(idx)
	if effect == nil {
		return
	}

	damage := effect.CalcValue(s.Caster)
	if damage <= 0 {
		damage = 1
	}

	s.EffectDamage[idx] = damage

	// Apply damage directly for now (full damage, no absorption/resistance)
	currentHealth := target.GetHealth()
	absorb := uint32(damage)
	if absorb > currentHealth {
		absorb = currentHealth
	}

	target.SetHealth(currentHealth - absorb)
}

// effectHeal applies direct healing to the target.
func (s *Spell) effectHeal(idx int, target Unit) {
	if target == nil {
		target = s.Caster
	}

	effect := s.Info.GetEffect(idx)
	if effect == nil {
		return
	}

	heal := effect.CalcValue(s.Caster)
	if heal <= 0 {
		heal = 1
	}

	s.EffectHeal[idx] = heal

	currentHealth := target.GetHealth()
	maxHealth := target.GetMaxHealth()
	newHealth := currentHealth + uint32(heal)
	if newHealth > maxHealth {
		newHealth = maxHealth
	}

	target.SetHealth(newHealth)
}

// effectEnergize restores power to the target.
func (s *Spell) effectEnergize(idx int, target Unit) {
	if target == nil {
		target = s.Caster
	}

	effect := s.Info.GetEffect(idx)
	if effect == nil {
		return
	}

	amount := effect.CalcValue(s.Caster)
	pt := wow.PowerType(s.Info.PowerType)

	currentPower := target.GetPower(pt)
	maxPower := target.GetMaxPower(pt)
	newPower := currentPower + uint32(amount)
	if newPower > maxPower {
		newPower = maxPower
	}

	target.SetPower(pt, newPower)
}

// effectApplyAura applies an aura (buff/debuff) to the target.
func (s *Spell) effectApplyAura(idx int, target Unit) {
	if target == nil {
		target = s.Caster
	}

	// Stub: full aura application is Phase 3
	ApplyAuraToTarget(s.Info, target, s.Caster)
}

// effectTriggerSpell triggers another spell from this spell.
func (s *Spell) effectTriggerSpell(idx int, target Unit) {
	effect := s.Info.GetEffect(idx)
	if effect == nil || effect.TriggerSpell == 0 {
		return
	}

	// Stub: trigger spell implementation will be added later
}

// effectDummy handles dummy spell effects (scripted behavior).
func (s *Spell) effectDummy(idx int, target Unit) {
	// Stub: dummy effects are spell-specific
}

// applyCooldowns adds the spell's cooldowns to the caster.
func (s *Spell) applyCooldowns() {
	if s.Info.RecoveryTime > 0 {
		end := time.Now().Add(time.Duration(s.Info.RecoveryTime) * time.Millisecond)
		AddSpellCooldown(s.Caster, s.Info.Id, 0, end)
	}

	if s.Info.CategoryRecoveryTime > 0 && s.Info.StartRecoveryCategory > 0 {
		end := time.Now().Add(time.Duration(s.Info.CategoryRecoveryTime) * time.Millisecond)
		AddSpellCooldown(s.Caster, s.Info.Id, s.Info.StartRecoveryCategory, end)
	}
}

// CasterLocation returns the caster's position as a Position.
func (s *Spell) CasterLocation() Position {
	return Position{
		X:     s.Caster.Location.X,
		Y:     s.Caster.Location.Y,
		Z:     s.Caster.Location.Z,
		MapID: s.Caster.Location.Map,
	}
}

// targetLocation returns the unit target's position as a Position.
func (s *Spell) targetLocation() Position {
	if p, ok := s.Targets.UnitTarget.(*player.Player); ok {
		return Position{
			X:     p.Location.X,
			Y:     p.Location.Y,
			Z:     p.Location.Z,
			MapID: p.Location.Map,
		}
	}
	return s.CasterLocation()
}

// distance2D calculates the 2D distance between two positions.
func distance2D(a, b Position) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// isSet returns true if the flag is set.
func (f TriggerCastFlags) isSet(flag TriggerCastFlags) bool {
	return f&flag != 0
}

// ApplyAuraToTarget is a stub for aura application.
// Full implementation is in Phase 3 (aura system).
func ApplyAuraToTarget(spell *SpellInfo, target Unit, caster Unit) {
	// Stub: aura application will be implemented in Phase 3
}
