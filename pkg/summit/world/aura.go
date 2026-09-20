package world

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// AuraRemoveMode defines how an aura was removed.
type AuraRemoveMode uint8

const (
	AuraRemoveNone          AuraRemoveMode = 0
	AuraRemoveByDefault     AuraRemoveMode = 1
	AuraRemoveByCancel      AuraRemoveMode = 2
	AuraRemoveByEnemySpell  AuraRemoveMode = 3
	AuraRemoveByExpire      AuraRemoveMode = 4
	AuraRemoveByDeath       AuraRemoveMode = 5
)

// AuraApplication tracks a single aura applied to a specific target.
type AuraApplication struct {
	Aura       *Aura
	Target     Unit
	Flags      uint8
	RemoveMode AuraRemoveMode
	EffectMask uint8
}

// Aura represents an active buff/debuff on a unit.
type Aura struct {
	SpellInfo  *SpellInfo
	Owner      Unit
	Caster     Unit
	CasterGUID uint64
	MaxDuration int32
	Duration   int32
	StackAmount uint8
	ProcCharges uint8
	Effects    [3]*AuraEffect
	Applications map[uint64]*AuraApplication
	IsPassive  bool
	RemoveMsg  AuraRemoveMode

	// Proc fields
	ProcFlags  uint32 // PROC_FLAG_* bitmask from SpellMgr.h
	ProcChance uint32 // 0-100 percent
}

// Update ticks the aura duration and updates all effects.
func (a *Aura) Update(diff uint32) {
	if a.Duration <= 0 && !a.IsPassive {
		return
	}

	if a.Duration > 0 {
		a.Duration -= int32(diff)
		if a.Duration < 0 {
			a.Duration = 0
		}
	}

	caster := a.Caster

	for _, eff := range a.Effects {
		if eff != nil {
			eff.Update(diff, caster)
		}
	}

	if a.IsExpired() && !a.IsPassive {
		a.Remove(AuraRemoveByExpire)
	}
}

// RefreshDuration resets Duration to MaxDuration.
func (a *Aura) RefreshDuration() {
	a.Duration = a.MaxDuration
}

// IsExpired returns true when duration has reached zero.
func (a *Aura) IsExpired() bool {
	return a.Duration <= 0
}

// SetDuration sets the aura duration in milliseconds.
func (a *Aura) SetDuration(d int32) {
	a.Duration = d
}

// Remove marks the aura for removal with the given mode.
func (a *Aura) Remove(mode AuraRemoveMode) {
	a.RemoveMsg = mode
}

// GetEffect returns the effect at the given index, or nil if out of range.
func (a *Aura) GetEffect(idx int) *AuraEffect {
	if idx < 0 || idx >= 3 {
		return nil
	}
	return a.Effects[idx]
}

// GetEffectMask returns a bitmask of which effect slots are populated.
func (a *Aura) GetEffectMask() uint8 {
	var mask uint8
	for i := 0; i < 3; i++ {
		if a.Effects[i] != nil {
			mask |= 1 << uint(i)
		}
	}
	return mask
}

// AuraEffect represents a single effect of an aura (e.g. periodic damage, stat modifier).
type AuraEffect struct {
	Base       *Aura
	SpellInfo  *SpellInfo
	EffIndex   uint8
	AuraType   AuraType
	Amount     int32
	BaseAmount int32
	DieSides   int32
	PeriodicTimer int32
	Amplitude  int32
	TickNumber uint32
	IsPeriodic bool
	CanBeRecalculated bool
}

// CalculateAmount computes the initial effect value.
// Simplified: returns BaseAmount (AC applies level scaling, spell mods, etc.).
func (e *AuraEffect) CalculateAmount(caster Unit) int32 {
	return e.BaseAmount
}

// CalculatePeriodic determines whether this effect is periodic and sets amplitude.
func (e *AuraEffect) CalculatePeriodic(caster Unit, create bool) {
	e.IsPeriodic = false

	if e.SpellInfo == nil {
		return
	}

	switch e.AuraType {
	case AuraPeriodicDamage, AuraPeriodicHeal, AuraPeriodicEnergize,
		AuraPeriodicTriggerSpell, AuraPeriodicTriggerSpellFromClient,
		AuraPeriodicLeech, AuraPeriodicManaLeech,
		AuraPeriodicDamagePercent, AuraPeriodicDummy,
		AuraPeriodicTriggerSpellWithValue,
		AuraObsModHealth, AuraObsModPower,
		AuraPeriodicHealthFunnel, AuraPowerBurn:
		e.IsPeriodic = true
	}

	if !e.IsPeriodic {
		return
	}

	// Load amplitude from spell effect data
	if e.EffIndex < 3 {
		e.Amplitude = int32(e.SpellInfo.Effects[e.EffIndex].Amplitude)
	}

	// Fallback for missing amplitude
	if e.Amplitude <= 0 {
		e.Amplitude = 1000
	}

	if create {
		e.TickNumber = 0
		e.PeriodicTimer = e.Amplitude
	}
}

// Update decrements the periodic timer and fires ticks.
func (e *AuraEffect) Update(diff uint32, caster Unit) {
	if !e.IsPeriodic {
		return
	}

	if e.Base != nil && e.Base.Duration <= 0 && !e.Base.IsPassive {
		return
	}

	e.PeriodicTimer -= int32(diff)
	for e.PeriodicTimer <= 0 {
		if e.Base != nil && !e.Base.IsPassive && !e.Base.IsExpired() {
			if int32(e.TickNumber+1) > e.GetTotalTicks() {
				break
			}
		}

		e.TickNumber++
		e.PeriodicTimer += e.Amplitude
		e.UpdatePeriodic(caster)
	}
}

// UpdatePeriodic dispatches to the appropriate handler for this aura type.
func (e *AuraEffect) UpdatePeriodic(caster Unit) {
	target := e.getEffectiveTarget()
	if target == nil {
		return
	}

	switch e.AuraType {
	case AuraPeriodicDamage, AuraPeriodicDamagePercent:
		handlePeriodicDamage(target, caster, e)
	case AuraPeriodicHeal, AuraObsModHealth:
		handlePeriodicHeal(target, caster, e)
	case AuraPeriodicEnergize, AuraObsModPower:
		handlePeriodicEnergize(target, caster, e)
	case AuraPeriodicTriggerSpell:
		handlePeriodicTriggerSpell(target, caster, e)
	case AuraPeriodicManaLeech:
		handlePeriodicManaLeech(target, caster, e)
	case AuraPeriodicLeech:
		handlePeriodicDamage(target, caster, e)
	}
}

// GetTotalTicks returns the expected number of periodic ticks.
func (e *AuraEffect) GetTotalTicks() int32 {
	if e.Amplitude > 0 && e.Base != nil && e.Base.Duration > 0 {
		return e.Base.Duration / e.Amplitude
	}
	return 0
}

// HandleEffect applies or removes the non-periodic portion of this aura effect.
func (e *AuraEffect) HandleEffect(target Unit, apply bool) {
	// Stub: will be filled in later phases.
}

// getEffectiveTarget returns the target for periodic ticks.
// For now, uses the aura owner if available.
func (e *AuraEffect) getEffectiveTarget() Unit {
	if e.Base != nil {
		return e.Base.Owner
	}
	return nil
}

// TryCreateAura creates a new Aura with initialized effects.
func TryCreateAura(info *SpellInfo, owner, caster Unit, baseAmount *int32) *Aura {
	if info == nil {
		return nil
	}

	aura := &Aura{
		SpellInfo:    info,
		Owner:        owner,
		Caster:       caster,
		MaxDuration:  info.GetMaxDuration(),
		Duration:     info.GetDuration(),
		StackAmount:  1,
		Applications: make(map[uint64]*AuraApplication),
		IsPassive:    info.IsPassive(),
	}

	if caster != nil {
		aura.CasterGUID = uint64(caster.GetGUID())
	}

	for i := 0; i < 3; i++ {
		effectInfo := info.GetEffect(i)
		if effectInfo == nil || !effectInfo.IsAura() {
			continue
		}

		eff := &AuraEffect{
			Base:       aura,
			SpellInfo:  info,
			EffIndex:   uint8(i),
			AuraType:   AuraType(effectInfo.ApplyAuraName),
			DieSides:   effectInfo.DieSides,
			CanBeRecalculated: true,
		}

		if baseAmount != nil {
			eff.BaseAmount = *baseAmount
		} else {
			eff.BaseAmount = effectInfo.BasePoints
		}

		eff.CalculatePeriodic(caster, true)
		eff.Amount = eff.CalculateAmount(caster)

		aura.Effects[i] = eff
	}

	return aura
}

// getGUID returns the GUID of a Unit (interface helper).
func getGUID(u Unit) wow.GUID {
	if u == nil {
		return 0
	}
	return u.GetGUID()
}
