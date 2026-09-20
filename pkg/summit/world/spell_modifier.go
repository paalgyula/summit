package world

// SpellModOp represents the type of spell modification.
// Values from AzerothCore SpellDefines.h.
type SpellModOp uint32

const (
	SpellModDamage                SpellModOp = 0
	SpellModDuration              SpellModOp = 1
	SpellModThreat                SpellModOp = 2
	SpellModEffect1               SpellModOp = 3
	SpellModCharges               SpellModOp = 4
	SpellModRange                 SpellModOp = 5
	SpellModRadius                SpellModOp = 6
	SpellModCriticalChance        SpellModOp = 7
	SpellModAllEffects            SpellModOp = 8
	SpellModNotLoseCastingTime    SpellModOp = 9
	SpellModCastingTime           SpellModOp = 10
	SpellModCooldown              SpellModOp = 11
	SpellModEffect2               SpellModOp = 12
	SpellModIgnoreArmor           SpellModOp = 13
	SpellModCost                  SpellModOp = 14
	SpellModCritDamageBonus       SpellModOp = 15
	SpellModResistMissChance      SpellModOp = 16
	SpellModJumpTargets           SpellModOp = 17
	SpellModChanceOfSuccess       SpellModOp = 18
	SpellModActivationTime        SpellModOp = 19
	SpellModDamageMultiplier      SpellModOp = 20
	SpellModGlobalCooldown        SpellModOp = 21
	SpellModDot                   SpellModOp = 22
	SpellModEffect3               SpellModOp = 23
	SpellModBonusMultiplier       SpellModOp = 24
	SpellModProcPerMinute         SpellModOp = 26
	SpellModValueMultiplier       SpellModOp = 27
	SpellModResistDispelChance    SpellModOp = 28
	SpellModCritDamageBonus2      SpellModOp = 29
	SpellModSpellCostRefundOnFail SpellModOp = 30

	MaxSpellMod SpellModOp = 32
)

// SpellModType indicates flat vs percent modification.
type SpellModType uint32

const (
	SpellModFlat SpellModType = 0
	SpellModPct  SpellModType = 1
)

// SpellModifier represents a talent/gear effect that modifies spells.
type SpellModifier struct {
	Op          SpellModOp
	Type        SpellModType
	Value       int32
	Mask        [3]uint32 // SpellFamilyFlags mask (flag96)
	SpellFamily SpellFamily
	SpellID     uint32
}

// ModifierList is a slice of spell modifiers.
type ModifierList []*SpellModifier

// IsAffectedBySpellMod checks whether a spell matches the modifier's family mask.
func IsAffectedBySpellMod(mod *SpellModifier, spell *SpellInfo) bool {
	if mod == nil || spell == nil {
		return false
	}

	// Generic family (0) matches everything
	if mod.SpellFamily == SpellFamilyGeneric {
		return true
	}

	// Spell family must match
	if mod.SpellFamily != SpellFamily(spell.SpellFamilyName) {
		return false
	}

	// If mask is all zeros, matches all spells in this family
	if mod.Mask[0] == 0 && mod.Mask[1] == 0 && mod.Mask[2] == 0 {
		return true
	}

	// Check family flags overlap
	return (spell.SpellFamilyFlags[0]&mod.Mask[0]) != 0 ||
		(spell.SpellFamilyFlags[1]&mod.Mask[1]) != 0 ||
		(spell.SpellFamilyFlags[2]&mod.Mask[2]) != 0
}

// ApplySpellMod applies all matching modifiers from the list to the given value.
// For SpellModCost, SpellModCastingTime, and SpellModDuration:
//
//	basevalue = (basevalue + totalflat) * totalmul  (clamped to 0)
//
// For other ops:
//
//	basevalue = (basevalue * totalmul) + totalflat
func ApplySpellMod(mods ModifierList, spell *SpellInfo, op SpellModOp, value *int32) {
	if spell == nil || value == nil {
		return
	}

	var totalmul float32 = 1.0
	var totalflat int32

	for _, mod := range mods {
		if mod.Op != op {
			continue
		}

		if !IsAffectedBySpellMod(mod, spell) {
			continue
		}

		switch mod.Type {
		case SpellModFlat:
			totalflat += mod.Value
		case SpellModPct:
			if *value == 0 || totalmul == 0 {
				continue
			}
			totalmul += float32(mod.Value) / 100.0
		}
	}

	base := float32(*value)
	if op == SpellModCastingTime || op == SpellModDuration || op == SpellModCost {
		v := (base + float32(totalflat)) * totalmul
		if v < 0 {
			v = 0
		}

		*value = int32(v)
	} else {
		*value = int32(base*totalmul) + totalflat
	}
}

// AddSpellModifier appends a modifier to the list.
func AddSpellModifier(mods *ModifierList, mod *SpellModifier) {
	if mod == nil {
		return
	}

	*mods = append(*mods, mod)
}

// RemoveSpellModifier removes a modifier from the list.
func RemoveSpellModifier(mods *ModifierList, mod *SpellModifier) {
	if mod == nil {
		return
	}

	for i, m := range *mods {
		if m == mod {
			*mods = append((*mods)[:i], (*mods)[i+1:]...)

			return
		}
	}
}
