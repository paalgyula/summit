package world

// SpellSchoolMask defines the spell school (elemental type).
type SpellSchoolMask uint32

const (
	SpellSchoolMaskNone      SpellSchoolMask = 0
	SpellSchoolMaskPhysical  SpellSchoolMask = 1
	SpellSchoolMaskHoly      SpellSchoolMask = 2
	SpellSchoolMaskFire      SpellSchoolMask = 4
	SpellSchoolMaskNature    SpellSchoolMask = 8
	SpellSchoolMaskFrost     SpellSchoolMask = 16
	SpellSchoolMaskShadow    SpellSchoolMask = 32
	SpellSchoolMaskArcane    SpellSchoolMask = 64

	SpellSchoolMaskAllMagic = SpellSchoolMaskFire | SpellSchoolMaskNature | SpellSchoolMaskFrost | SpellSchoolMaskShadow | SpellSchoolMaskArcane
	SpellSchoolMaskAll      = SpellSchoolMaskPhysical | SpellSchoolMaskAllMagic | SpellSchoolMaskHoly
)

// SpellEffect defines what a spell does.
type SpellEffect uint32

const (
	SpellEffectNone               SpellEffect = 0
	SpellEffectInstantKill        SpellEffect = 1
	SpellEffectSchoolDamage       SpellEffect = 2  // Direct damage
	SpellEffectDummy              SpellEffect = 3
	SpellEffectPortalTeleport     SpellEffect = 4
	SpellEffectTeleportUnit       SpellEffect = 5
	SpellEffectHeal               SpellEffect = 10 // Direct heal
	SpellEffectBind                SpellEffect = 11
	SpellEffectSummonPet          SpellEffect = 12
	SpellEffectSummonCreature     SpellEffect = 56
	SpellEffectApplyAura          SpellEffect = 23 // Apply buff/debuff
	SpellEffectWeaponDmgNOSchool  SpellEffect = 54
	SpellEffectWeaponDmg          SpellEffect = 78
	SpellEffectEnergize           SpellEffect = 30 // Restore power
	SpellEffectSchoolDamage2      SpellEffect = 109
	SpellEffectPowerBurn          SpellEffect = 128 // Drain mana and deal damage
)

// AuraType defines what kind of buff/debuff an aura applies.
type AuraType uint32

const (
	AuraNone                        AuraType = 0
	AuraModShapeshift              AuraType = 36
	AuraModMechanicImmunity        AuraType = 49
	AuraModStun                    AuraType = 12
	AuraModRoot                    AuraType = 13
	AuraModFear                    AuraType = 7
	AuraModTaunt                   AuraType = 11
	AuraModSpeed                   AuraType = 31
	AuraModMeleeHaste              AuraType = 65
	AuraModModDamageDone           AuraType = 79
	AuraModModDamagePercentDone    AuraType = 108
	AuraPeriodicDamage             AuraType = 3
	AuraPeriodicHeal               AuraType = 8
	AuraPeriodicEnergize           AuraType = 24
)

// SpellCastResult defines the result of a spell cast attempt.
type SpellCastResult uint32

const (
	SpellCastSuccess            SpellCastResult = 0
	SpellCastFailedSpellFailed  SpellCastResult = 6
	SpellCastNoMana             SpellCastResult = 12
	SpellCastTargetTooFar       SpellCastResult = 51
	SpellCastAlreadyActive      SpellCastResult = 10 // Already have this buff
	SpellCastCantDoThatYet     SpellCastResult = 165
)

// SpellState defines the state of a spell cast.
type SpellState uint32

const (
	SpellStateNone      SpellState = 0
	SpellStateCasting   SpellState = 1 // Currently casting
	SpellStateCastTime  SpellState = 2 // Waiting for cast time
	SpellStateActive    SpellState = 3 // Spell is active (channeled)
	SpellStateFinished  SpellState = 4 // Finished
)

// SpellTargetType defines who the spell can target.
type SpellTargetType uint32

const (
	SpellTargetNone            SpellTargetType = 0
	SpellTargetUnit            SpellTargetType = 1 // Any unit
	SpellTargetEnemy           SpellTargetType = 2 // Enemy unit
	SpellTargetFriendly        SpellTargetType = 3 // Friendly unit
	SpellTargetSelf            SpellTargetType = 4 // Self only
	SpellTargetArea            SpellTargetType = 5 // Area around target
	SpellTargetAll             SpellTargetType = 6 // All targets
)

// PowerType for spell cost.
type SpellPowerType uint32

const (
	SpellPowerMana    SpellPowerType = 0
	SpellPowerRage    SpellPowerType = 1
	SpellPowerFocus   SpellPowerType = 2
	SpellPowerEnergy  SpellPowerType = 3
	SpellPowerHappiness SpellPowerType = 4
)
