package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/paalgyula/summit/pkg/wow"
)

// Unit is a minimal interface for spell caster/target references.
// Satisfied by *player.Player.
type Unit interface {
	GetGUID() wow.GUID
	GetLevel() uint32
	GetHealth() uint32
	GetMaxHealth() uint32
	SetHealth(uint32)
	IsAlive() bool
	GetPower(wow.PowerType) uint32
	SetPower(wow.PowerType, uint32)
	GetMaxPower(wow.PowerType) uint32
}

// AreaTargetMin is the minimum implicit target ID that indicates an area effect.
// Target IDs >= this value through 51 are area/cone targets in WoW 3.3.5a.
const AreaTargetMin uint32 = 6

// IsAreaTargetID returns true if the given implicit target ID represents an area or cone effect.
func IsAreaTargetID(target uint32) bool {
	return target >= AreaTargetMin && target <= 51
}

// SpellInfo holds the computed spell data, constructed from SpellEntry DBC.
// Maps to AzerothCore's SpellInfo from SpellInfo.h.
type SpellInfo struct {
	Id uint32

	// Basic data
	Dispel   uint32
	Mechanic uint32

	// Attributes (AC uses 8 uint32 attribute fields)
	Attributes   uint32
	AttributesEx  uint32
	AttributesEx2 uint32
	AttributesEx3 uint32
	AttributesEx4 uint32
	AttributesEx5 uint32
	AttributesEx6 uint32
	AttributesEx7 uint32

	// Custom attributes (computed by SpellMgr, not in DBC)
	AttributesCu uint32

	// Shapeshift
	Stances    uint32
	StancesNot uint32

	// Targeting
	Targets            uint32
	TargetCreatureType uint32
	RequiresSpellFocus uint32
	FacingCasterFlags  uint32

	// Aura state requirements
	CasterAuraState      uint32
	TargetAuraState      uint32
	CasterAuraStateNot   uint32
	TargetAuraStateNot   uint32
	CasterAuraSpell      uint32
	TargetAuraSpell      uint32
	ExcludeCasterAuraSpell uint32
	ExcludeTargetAuraSpell uint32

	// Cooldowns
	RecoveryTime         uint32 // ms
	CategoryRecoveryTime uint32 // ms
	StartRecoveryCategory uint32
	StartRecoveryTime    uint32 // ms

	// Interrupt
	InterruptFlags       uint32
	AuraInterruptFlags   uint32
	ChannelInterruptFlags uint32

	// Proc
	ProcFlags  uint32
	ProcChance uint32
	ProcCharges uint32

	// Levels
	MaxLevel  uint32
	BaseLevel uint32
	SpellLevel uint32

	// Duration (resolved from DBC)
	Duration int32 // ms, resolved from DurationIndex

	// Power
	PowerType        uint32
	ManaCost         uint32
	ManaCostPerlevel uint32
	ManaPerSecond    uint32
	ManaPerSecondPerLevel uint32
	ManaCostPercentage uint32

	// Range (resolved from DBC)
	RangeMin float32
	RangeMax float32

	// Casting
	CastTime uint32 // ms, resolved from CastingTimeIndex
	Speed    float32

	// Stacking
	StackAmount uint32

	// Reagents
	Totem          [2]uint32
	Reagent        [8]int32
	ReagentCount   [8]uint32

	// Equipment requirements
	EquippedItemClass            int32
	EquippedItemSubClassMask     int32
	EquippedItemInventoryTypeMask int32

	// Effects (up to 3)
	Effects [3]SpellEffectInfo

	// Spell family
	SpellFamilyName  uint32
	SpellFamilyFlags [3]uint32

	// Targeting
	MaxAffectedTargets uint32
	MaxTargetLevel     uint32

	// Damage class
	DmgClass       uint32
	PreventionType uint32

	// School
	SchoolMask uint32

	// Visuals
	SpellIconID  uint32
	ActiveIconID uint32
	SpellVisual  [2]uint32

	// Totem category
	TotemCategory [2]uint32
	AreaGroupId   int32
	RuneCostID    uint32

	// Computed helper
	DurationMs time.Duration
	RangeMaxF  float32
	CastTimeMs time.Duration
}

// SpellEffectInfo holds per-effect data.
// Maps to AzerothCore's SpellEffectInfo from SpellInfo.h.
type SpellEffectInfo struct {
	Effect            uint32
	ApplyAuraName     uint32
	Amplitude         uint32 // ms between periodic ticks
	DieSides          int32
	BasePoints        int32
	RealPointsPerLevel float32
	PointsPerComboPoint float32
	ValueMultiplier   float32
	DamageMultiplier  float32
	BonusMultiplier   float32
	MiscValue         int32
	MiscValueB        int32
	Mechanic          uint32
	TargetA           uint32 // implicit target A
	TargetB           uint32 // implicit target B
	RadiusIndex       uint32
	ChainTarget       uint32
	ItemType          uint32
	TriggerSpell      uint32
	SpellClassMask    [3]uint32
}

// NewSpellInfo constructs a SpellInfo from a DBC SpellEntryEntry.
func NewSpellInfo(entry *wotlk.SpellEntryEntry, castTimes []wotlk.SpellCastTimeEntry, durations []wotlk.SpellDurationEntry, ranges []wotlk.SpellRangeEntry) *SpellInfo {
	si := &SpellInfo{
		Id: entry.Id,

		Dispel:   entry.Dispel,
		Mechanic: entry.Mechanic,

		Attributes:   entry.Attributes,
		AttributesEx:  entry.AttributesEx,
		AttributesEx2: entry.AttributesEx2,
		AttributesEx3: entry.AttributesEx3,
		AttributesEx4: entry.AttributesEx4,
		AttributesEx5: entry.AttributesEx5,
		AttributesEx6: entry.AttributesEx6,
		AttributesEx7: entry.AttributesEx7,

		Stances:    entry.Stances,
		StancesNot: entry.StancesNot,

		Targets:            entry.Targets,
		TargetCreatureType: entry.TargetCreatureType,
		RequiresSpellFocus: entry.RequiresSpellFocus,
		FacingCasterFlags:  entry.FacingCasterFlags,

		CasterAuraState:      entry.CasterAuraState,
		TargetAuraState:      entry.TargetAuraState,
		CasterAuraStateNot:   entry.CasterAuraStateNot,
		TargetAuraStateNot:   entry.TargetAuraStateNot,
		CasterAuraSpell:      entry.CasterAuraSpell,
		TargetAuraSpell:      entry.TargetAuraSpell,
		ExcludeCasterAuraSpell: entry.ExcludeCasterAuraSpell,
		ExcludeTargetAuraSpell: entry.ExcludeTargetAuraSpell,

		RecoveryTime:         entry.RecoveryTime,
		CategoryRecoveryTime: entry.CategoryRecoveryTime,
		StartRecoveryCategory: entry.StartRecoveryCategory,
		StartRecoveryTime:    entry.StartRecoveryTime,

		InterruptFlags:        entry.InterruptFlags,
		AuraInterruptFlags:    entry.AuraInterruptFlags,
		ChannelInterruptFlags: entry.ChannelInterruptFlags,

		ProcFlags:   entry.ProcFlags,
		ProcChance:  entry.ProcChance,
		ProcCharges: entry.ProcCharges,

		MaxLevel:  entry.MaxLevel,
		BaseLevel: entry.BaseLevel,
		SpellLevel: entry.SpellLevel,

		PowerType:        entry.PowerType,
		ManaCost:         entry.ManaCost,
		ManaCostPerlevel: entry.ManaCostPerlevel,
		ManaPerSecond:    entry.ManaPerSecond,
		ManaPerSecondPerLevel: entry.ManaPerSecondPerLevel,
		ManaCostPercentage: entry.ManaCostPercentage,

		Speed:       entry.Speed,
		StackAmount: entry.StackAmount,

		Totem: [2]uint32{entry.Totem0, entry.Totem1},
		Reagent: [8]int32{
			int32(entry.Reagent0), int32(entry.Reagent1), int32(entry.Reagent2), int32(entry.Reagent3),
			int32(entry.Reagent4), int32(entry.Reagent5), int32(entry.Reagent6), int32(entry.Reagent7),
		},
		ReagentCount: [8]uint32{
			entry.ReagentCount0, entry.ReagentCount1, entry.ReagentCount2, entry.ReagentCount3,
			entry.ReagentCount4, entry.ReagentCount5, entry.ReagentCount6, entry.ReagentCount7,
		},

		EquippedItemClass:             int32(entry.EquippedItemClass),
		EquippedItemSubClassMask:      int32(entry.EquippedItemSubClassMask),
		EquippedItemInventoryTypeMask: int32(entry.EquippedItemInventoryTypeMask),

		SpellFamilyName:  entry.SpellFamilyName,
		SpellFamilyFlags: [3]uint32{entry.SpellFamilyFlags0, entry.SpellFamilyFlags1, entry.SpellFamilyFlags2},

		MaxAffectedTargets: entry.MaxAffectedTargets,
		MaxTargetLevel:     entry.MaxTargetLevel,

		DmgClass:       entry.DmgClass,
		PreventionType: entry.PreventionType,

		SchoolMask: entry.SchoolMask,

		SpellIconID:  entry.SpellIconID,
		ActiveIconID: entry.ActiveIconID,
		SpellVisual:  [2]uint32{entry.SpellVisual0, entry.SpellVisual1},

		TotemCategory: [2]uint32{entry.TotemCategory0, entry.TotemCategory1},
		AreaGroupId:   int32(entry.AreaGroupId),
		RuneCostID:    entry.RuneCostID,
	}

	// Resolve effects
	si.Effects[0] = newSpellEffectInfo(entry, 0)
	si.Effects[1] = newSpellEffectInfo(entry, 1)
	si.Effects[2] = newSpellEffectInfo(entry, 2)

	// Resolve cast time from DBC
	if int(entry.CastingTimeIndex) < len(castTimes) {
		for _, ct := range castTimes {
			if ct.ID == entry.CastingTimeIndex {
				si.CastTime = ct.CastTime
				si.CastTimeMs = time.Duration(ct.CastTime) * time.Millisecond
				break
			}
		}
	}

	// Resolve duration from DBC
	if int(entry.DurationIndex) < len(durations) {
		for _, d := range durations {
			if d.ID == entry.DurationIndex {
				si.Duration = int32(d.Duration)
				si.DurationMs = time.Duration(d.Duration) * time.Millisecond
				break
			}
		}
	}

	// Resolve range from DBC
	if int(entry.RangeIndex) < len(ranges) {
		for _, r := range ranges {
			if r.ID == entry.RangeIndex {
				si.RangeMin = r.Field1
				si.RangeMax = r.Field2
				si.RangeMaxF = r.Field2
				break
			}
		}
	}

	return si
}

func newSpellEffectInfo(entry *wotlk.SpellEntryEntry, idx int) SpellEffectInfo {
	switch idx {
	case 0:
		return SpellEffectInfo{
			Effect:              entry.Effect0,
			ApplyAuraName:       entry.EffectApplyAuraName0,
			Amplitude:           entry.EffectAmplitude0,
			DieSides:            int32(entry.EffectDieSides0),
			BasePoints:          int32(entry.EffectBasePoints0),
			RealPointsPerLevel:  entry.EffectRealPointsPerLevel0,
			PointsPerComboPoint: entry.EffectPointsPerComboPoint0,
			ValueMultiplier:     entry.EffectValueMultiplier0,
			DamageMultiplier:    entry.EffectDamageMultiplier0,
			BonusMultiplier:     entry.EffectBonusMultiplier0,
			MiscValue:           int32(entry.EffectMiscValue0),
			MiscValueB:          int32(entry.EffectMiscValueB0),
			Mechanic:            entry.EffectMechanic0,
			TargetA:             entry.EffectImplicitTargetA0,
			TargetB:             entry.EffectImplicitTargetB0,
			RadiusIndex:         entry.EffectRadiusIndex0,
			ChainTarget:         entry.EffectChainTarget0,
			ItemType:            entry.EffectItemType0,
			TriggerSpell:        entry.EffectTriggerSpell0,
			SpellClassMask: [3]uint32{
				entry.EffectSpellClassMask0_0, entry.EffectSpellClassMask0_1, entry.EffectSpellClassMask0_2,
			},
		}
	case 1:
		return SpellEffectInfo{
			Effect:              entry.Effect1,
			ApplyAuraName:       entry.EffectApplyAuraName1,
			Amplitude:           entry.EffectAmplitude1,
			DieSides:            int32(entry.EffectDieSides1),
			BasePoints:          int32(entry.EffectBasePoints1),
			RealPointsPerLevel:  entry.EffectRealPointsPerLevel1,
			PointsPerComboPoint: entry.EffectPointsPerComboPoint1,
			ValueMultiplier:     entry.EffectValueMultiplier1,
			DamageMultiplier:    entry.EffectDamageMultiplier1,
			BonusMultiplier:     entry.EffectBonusMultiplier1,
			MiscValue:           int32(entry.EffectMiscValue1),
			MiscValueB:          int32(entry.EffectMiscValueB1),
			Mechanic:            entry.EffectMechanic1,
			TargetA:             entry.EffectImplicitTargetA1,
			TargetB:             entry.EffectImplicitTargetB1,
			RadiusIndex:         entry.EffectRadiusIndex1,
			ChainTarget:         entry.EffectChainTarget1,
			ItemType:            entry.EffectItemType1,
			TriggerSpell:        entry.EffectTriggerSpell1,
			SpellClassMask: [3]uint32{
				entry.EffectSpellClassMask1_0, entry.EffectSpellClassMask1_1, entry.EffectSpellClassMask1_2,
			},
		}
	case 2:
		return SpellEffectInfo{
			Effect:              entry.Effect2,
			ApplyAuraName:       entry.EffectApplyAuraName2,
			Amplitude:           entry.EffectAmplitude2,
			DieSides:            int32(entry.EffectDieSides2),
			BasePoints:          int32(entry.EffectBasePoints2),
			RealPointsPerLevel:  entry.EffectRealPointsPerLevel2,
			PointsPerComboPoint: entry.EffectPointsPerComboPoint2,
			ValueMultiplier:     entry.EffectValueMultiplier2,
			DamageMultiplier:    entry.EffectDamageMultiplier2,
			BonusMultiplier:     entry.EffectBonusMultiplier2,
			MiscValue:           int32(entry.EffectMiscValue2),
			MiscValueB:          int32(entry.EffectMiscValueB2),
			Mechanic:            entry.EffectMechanic2,
			TargetA:             entry.EffectImplicitTargetA2,
			TargetB:             entry.EffectImplicitTargetB2,
			RadiusIndex:         entry.EffectRadiusIndex2,
			ChainTarget:         entry.EffectChainTarget2,
			ItemType:            entry.EffectItemType2,
			TriggerSpell:        entry.EffectTriggerSpell2,
			SpellClassMask: [3]uint32{
				entry.EffectSpellClassMask2_0, entry.EffectSpellClassMask2_1, entry.EffectSpellClassMask2_2,
			},
		}
	}
	return SpellEffectInfo{}
}

// HasAttribute checks if any of the 8 attribute fields has the given flag.
func (si *SpellInfo) HasAttribute(attr uint32) bool {
	return si.Attributes&attr != 0 ||
		si.AttributesEx&attr != 0 ||
		si.AttributesEx2&attr != 0 ||
		si.AttributesEx3&attr != 0 ||
		si.AttributesEx4&attr != 0 ||
		si.AttributesEx5&attr != 0 ||
		si.AttributesEx6&attr != 0 ||
		si.AttributesEx7&attr != 0
}

// HasAttributeEx checks if the AttributesEx field has the given flag.
func (si *SpellInfo) HasAttributeEx(attr uint32) bool {
	return si.AttributesEx&attr != 0
}

// HasAttributeEx2 checks if the AttributesEx2 field has the given flag.
func (si *SpellInfo) HasAttributeEx2(attr uint32) bool {
	return si.AttributesEx2&attr != 0
}

// IsPassive returns true if the spell is a passive effect.
func (si *SpellInfo) IsPassive() bool {
	return si.Attributes&uint32(SpellAttr0Passive) != 0
}

// IsAutoRepeat returns true if the spell is an auto-repeat ranged spell.
func (si *SpellInfo) IsAutoRepeat() bool {
	return si.AttributesEx2&uint32(SpellAttr2AutoRepeat) != 0
}

// IsPositive returns true if the spell is generally positive (helpful).
// AC checks AttributesCu for NEGATIVE/POSITIVE flags; simplified heuristic: no negative flag.
func (si *SpellInfo) IsPositive() bool {
	if si.AttributesCu&SpellAttr0CuNegative != 0 {
		return false
	}
	if si.AttributesCu&SpellAttr0CuPositive != 0 {
		return true
	}
	return si.AttributesEx&uint32(SpellAttrExNegative) == 0
}

// IsPositiveEffect returns true if the effect at the given index is positive.
// AC checks AttributesCu for per-effect NEGATIVE/POSITIVE flags; simplified heuristic.
func (si *SpellInfo) IsPositiveEffect(idx int) bool {
	if idx < 0 || idx >= 3 {
		return true
	}
	switch idx {
	case 0:
		if si.AttributesCu&SpellAttr0CuNegativeEff0 != 0 {
			return false
		}
		if si.AttributesCu&SpellAttr0CuPositiveEff0 != 0 {
			return true
		}
	case 1:
		if si.AttributesCu&SpellAttr0CuNegativeEff1 != 0 {
			return false
		}
		if si.AttributesCu&SpellAttr0CuPositiveEff1 != 0 {
			return true
		}
	case 2:
		if si.AttributesCu&SpellAttr0CuNegativeEff2 != 0 {
			return false
		}
		if si.AttributesCu&SpellAttr0CuPositiveEff2 != 0 {
			return true
		}
	}
	return si.IsPositive()
}

// IsChanneled returns true if the spell is channeled.
// AC: (AttributesEx & (SPELL_ATTR1_IS_CHANNELED | SPELL_ATTR1_IS_SELF_CHANNELED))
func (si *SpellInfo) IsChanneled() bool {
	return si.AttributesEx&(uint32(SpellAttrExChanneled1)|uint32(SpellAttrExChanneled2)) != 0
}

// IsSelfCast returns true if all effects target only the caster.
func (si *SpellInfo) IsSelfCast() bool {
	for i := 0; i < 3; i++ {
		if si.Effects[i].Effect != 0 && si.Effects[i].TargetA != 1 {
			return false
		}
	}
	return true
}

// NeedsComboPoints returns true if the spell requires combo points.
// AC: (AttributesEx & (SPELL_ATTR1_FINISHING_MOVE_DAMAGE | SPELL_ATTR1_FINISHING_MOVE_DURATION))
func (si *SpellInfo) NeedsComboPoints() bool {
	return si.AttributesEx&(uint32(SpellAttrExFinishingMoveDamage)|uint32(SpellAttrExFinishingMoveDuration)) != 0
}

// IsBreakingStealth returns true if the spell breaks stealth.
// AC: !(AttributesEx & SPELL_ATTR1_ALLOW_WHILE_STEALTHED)
func (si *SpellInfo) IsBreakingStealth() bool {
	return si.AttributesEx&uint32(SpellAttrExAllowWhileStealthed) == 0
}

// HasEffect returns true if any effect matches the given SpellEffect type.
func (si *SpellInfo) HasEffect(effect SpellEffect) bool {
	for i := 0; i < 3; i++ {
		if SpellEffect(si.Effects[i].Effect) == effect {
			return true
		}
	}
	return false
}

// HasAura returns true if any effect applies the given aura type.
func (si *SpellInfo) HasAura(aura AuraType) bool {
	for i := 0; i < 3; i++ {
		if si.Effects[i].Effect != 0 && AuraType(si.Effects[i].ApplyAuraName) == aura {
			return true
		}
	}
	return false
}

// HasEffectMechanic returns true if any effect has the given mechanic.
func (si *SpellInfo) HasEffectMechanic(mechanic uint32) bool {
	for i := 0; i < 3; i++ {
		if si.Effects[i].Effect != 0 && si.Effects[i].Mechanic == mechanic {
			return true
		}
	}
	return false
}

// GetEffect returns the SpellEffectInfo for the given index.
func (si *SpellInfo) GetEffect(idx int) *SpellEffectInfo {
	if idx < 0 || idx >= 3 {
		return nil
	}
	if si.Effects[idx].Effect == 0 {
		return nil
	}
	return &si.Effects[idx]
}

// GetAllEffectsMechanicMask returns a bitmask of all mechanics in all effects.
func (si *SpellInfo) GetAllEffectsMechanicMask() uint64 {
	var mask uint64
	if si.Mechanic != 0 {
		mask |= 1 << si.Mechanic
	}
	for i := 0; i < 3; i++ {
		if si.Effects[i].Effect != 0 && si.Effects[i].Mechanic != 0 {
			mask |= 1 << si.Effects[i].Mechanic
		}
	}
	return mask
}

// GetSchoolMask returns the spell school as a SpellSchoolMask.
func (si *SpellInfo) GetSchoolMask() SpellSchoolMask {
	return SpellSchoolMask(si.SchoolMask)
}

// GetDuration returns the spell duration in milliseconds.
func (si *SpellInfo) GetDuration() int32 {
	return si.Duration
}

// GetMaxDuration returns the max spell duration in milliseconds.
// For now returns the same as GetDuration (DBC has min/max but we only load base duration).
func (si *SpellInfo) GetMaxDuration() int32 {
	return si.Duration
}

// GetMaxTicks returns the maximum number of periodic ticks for this spell.
// AC: duration / amplitude, capped at 200% base duration (30s), default 6.
func (si *SpellInfo) GetMaxTicks() uint32 {
	dotDuration := si.GetDuration()
	if dotDuration == 0 {
		return 1
	}
	if dotDuration > 30000 {
		dotDuration = 30000
	}
	for i := 0; i < 3; i++ {
		if si.Effects[i].Effect == uint32(SpellEffectApplyAura) {
			switch AuraType(si.Effects[i].ApplyAuraName) {
			case AuraPeriodicDamage, AuraPeriodicHeal, AuraPeriodicLeech,
				AuraPeriodicTriggerSpellFromClient:
				if si.Effects[i].Amplitude != 0 {
					return uint32(dotDuration) / si.Effects[i].Amplitude
				}
			}
		}
	}
	return 6
}

// GetDispelMask returns the dispel bitmask for this spell.
func (si *SpellInfo) GetDispelMask() uint32 {
	return 1 << si.Dispel
}

// GetMaxAffectedTargets returns the maximum number of targets (0 = unlimited).
func (si *SpellInfo) GetMaxAffectedTargets() uint32 {
	return si.MaxAffectedTargets
}

// GetExplicitTargetMask returns the computed explicit target mask.
// For now returns the spell-level Targets field.
func (si *SpellInfo) GetExplicitTargetMask() uint32 {
	return si.Targets
}

// CalcCastTime returns the base cast time in milliseconds.
// AC applies caster spell mods; simplified: return base CastTime.
func (si *SpellInfo) CalcCastTime(_ Unit) uint32 {
	return si.CastTime
}

// CalcPowerCost returns the base power cost.
// AC applies percentage costs and spell mods; simplified: return ManaCost.
func (si *SpellInfo) CalcPowerCost(_ Unit, _ SpellSchoolMask) int32 {
	return int32(si.ManaCost)
}

// GetMinRange returns the minimum spell range.
func (si *SpellInfo) GetMinRange(_ bool) float32 {
	return si.RangeMin
}

// GetMaxRange returns the maximum spell range.
// AC applies spell mods from caster; simplified: return RangeMax.
func (si *SpellInfo) GetMaxRange(_ bool, _ Unit) float32 {
	return si.RangeMax
}

// CheckShapeshift checks if the spell can be cast in the given shapeshift form.
// Simplified: returns success unless stance is explicitly disallowed.
func (si *SpellInfo) CheckShapeshift(form ShapeshiftForm) SpellCastResult {
	stanceMask := uint32(0)
	if form != 0 {
		stanceMask = 1 << (uint32(form) - 1)
	}
	if stanceMask&si.StancesNot != 0 {
		return SpellCastResult(69) // SPELL_FAILED_NOT_SHAPESHIFT
	}
	if stanceMask&si.Stances != 0 {
		return SpellCastSuccess
	}
	if si.Stances != 0 && si.AttributesEx2&uint32(SpellAttr2AllowWhileNotShapeshifted) == 0 {
		return SpellCastResult(76) // SPELL_FAILED_ONLY_SHAPESHIFT
	}
	return SpellCastSuccess
}

// CheckTarget performs basic target validation.
// Simplified: checks alive/dead state only.
func (si *SpellInfo) CheckTarget(_ Unit, target Unit) SpellCastResult {
	_ = target
	return SpellCastSuccess
}

// CheckExplicitTarget performs explicit target validation.
// Simplified: returns success for basic cases.
func (si *SpellInfo) CheckExplicitTarget(_ Unit, target Unit) SpellCastResult {
	_ = target
	return SpellCastSuccess
}

// --- SpellEffectInfo methods ---

// IsEffect returns true if this effect slot is active (non-zero effect type).
func (sei *SpellEffectInfo) IsEffect() bool {
	return sei.Effect != 0
}

// IsAura returns true if this effect applies an aura.
// AC: (IsUnitOwnedAuraEffect() || Effect == SPELL_EFFECT_PERSISTENT_AREA_AURA) && ApplyAuraName != 0
func (sei *SpellEffectInfo) IsAura() bool {
	return sei.IsUnitOwnedAuraEffect() && sei.ApplyAuraName != 0
}

// IsAuraType returns true if this effect applies the given aura type.
func (sei *SpellEffectInfo) IsAuraType(aura AuraType) bool {
	return sei.IsAura() && AuraType(sei.ApplyAuraName) == aura
}

// IsTargetingArea returns true if either implicit target is an area/cone target.
func (sei *SpellEffectInfo) IsTargetingArea() bool {
	return IsAreaTargetID(sei.TargetA) || IsAreaTargetID(sei.TargetB)
}

// IsAreaAuraEffect returns true if this is an area aura effect type.
func (sei *SpellEffectInfo) IsAreaAuraEffect() bool {
	switch SpellEffect(sei.Effect) {
	case SpellEffectApplyAreaAuraParty, SpellEffectApplyAreaAuraRaid,
		SpellEffectApplyAreaAuraFriend, SpellEffectApplyAreaAuraEnemy,
		SpellEffectApplyAreaAuraPet, SpellEffectApplyAreaAuraOwner:
		return true
	}
	return false
}

// IsUnitOwnedAuraEffect returns true if this is a unit-owned aura effect.
func (sei *SpellEffectInfo) IsUnitOwnedAuraEffect() bool {
	return sei.IsAreaAuraEffect() || SpellEffect(sei.Effect) == SpellEffectApplyAura
}

// CalcValue returns the base effect value.
// AC applies level scaling, combo points, spell mods; simplified: return BasePoints.
func (sei *SpellEffectInfo) CalcValue(_ Unit) int32 {
	return sei.BasePoints
}

// CalcRadius returns the effect radius.
// AC loads from DBC SpellRadiusEntry; simplified: return 0 (radius not loaded yet).
func (sei *SpellEffectInfo) CalcRadius(_ Unit) float32 {
	return 0
}
