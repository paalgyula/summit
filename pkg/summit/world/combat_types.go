package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// DamageEffectType mirrors AC's DamageEffectType enum (Unit.h:253).
type DamageEffectType uint8

const (
	DirectDamage      DamageEffectType = 0 // normal weapon damage
	SpellDirectDamage DamageEffectType = 1 // spell/class abilities
	DOT               DamageEffectType = 2 // damage over time
	HealEffect        DamageEffectType = 3
	NoDamage          DamageEffectType = 4 // absorb/resist only
	SelfDamage        DamageEffectType = 5
)

// MeleeHitOutcome mirrors AC's MeleeHitOutcome enum (Unit.h:290).
type MeleeHitOutcome uint8

const (
	MeleeHitEvade    MeleeHitOutcome = 0
	MeleeHitMiss     MeleeHitOutcome = 1
	MeleeHitDodge    MeleeHitOutcome = 2
	MeleeHitBlock    MeleeHitOutcome = 3
	MeleeHitParry    MeleeHitOutcome = 4
	MeleeHitGlancing MeleeHitOutcome = 5
	MeleeHitCrit     MeleeHitOutcome = 6
	MeleeHitCrushing MeleeHitOutcome = 7
	MeleeHitNormal   MeleeHitOutcome = 8
)

// WeaponAttackType mirrors AC's WeaponAttackType (Unit.h:208).
type WeaponAttackType uint8

const (
	BaseAttack   WeaponAttackType = 0
	OffAttack    WeaponAttackType = 1
	RangedAttack WeaponAttackType = 2
)

// CombatRating mirrors AC's CombatRating enum (Unit.h:222).
type CombatRating uint8

const (
	CRWeaponSkill          CombatRating = 0
	CRDefenseSkill         CombatRating = 1
	CRDodge                CombatRating = 2
	CRParry                CombatRating = 3
	CRBlock                CombatRating = 4
	CRHitMelee             CombatRating = 5
	CRHitRanged            CombatRating = 6
	CRHitSpell             CombatRating = 7
	CRCritMelee            CombatRating = 8
	CRCritRanged           CombatRating = 9
	CRCritSpell            CombatRating = 10
	CRHitTakenMelee        CombatRating = 11
	CRHitTakenRanged       CombatRating = 12
	CRHitTakenSpell        CombatRating = 13
	CRCritTakenMelee       CombatRating = 14
	CRCritTakenRanged      CombatRating = 15
	CRCritTakenSpell       CombatRating = 16
	CRHasteMelee           CombatRating = 17
	CRHasteRanged          CombatRating = 18
	CRHasteSpell           CombatRating = 19
	CRWeaponSkillMainhand  CombatRating = 20
	CRWeaponSkillOffhand   CombatRating = 21
	CRWeaponSkillRanged    CombatRating = 22
	CRExpertise            CombatRating = 23
	CRArmorPenetration     CombatRating = 24
	MaxCombatRating        = 25
)

// DuelState mirrors AC's DuelState enum (Player.h:355).
type DuelState uint8

const (
	DuelStateChallenged DuelState = 0
	DuelStateCountdown  DuelState = 1
	DuelStateInProgress DuelState = 2
	DuelStateCompleted  DuelState = 3
)

// DuelCompleteType mirrors AC's DuelCompleteType (SharedDefines.h:3866).
type DuelCompleteType uint8

const (
	DuelInterrupted DuelCompleteType = 0
	DuelWon         DuelCompleteType = 1
	DuelFled        DuelCompleteType = 2
)

// CleanDamage represents damage that was mitigated (absorbed/resisted/blocked)
// and does not reduce health. Maps to AC's CleanDamage struct.
type CleanDamage struct {
	AbsorbedDamage  uint32
	MitigatedDamage uint32
	AttackType      WeaponAttackType
	HitOutcome      MeleeHitOutcome
}

// CalcDamageInfo holds the full state for a single melee damage calculation.
// Maps to AC's CalcDamageInfo struct (Unit.h:483).
type CalcDamageInfo struct {
	Attacker    CombatUnit
	Target      CombatUnit
	AttackType  WeaponAttackType
	HitInfo     uint32
	HitOutcome  MeleeHitOutcome

	Damages     [2]CalcDamage // [0]=main hand, [1]=off hand
	CleanDamage uint32        // total mitigated damage

	ProcAttacker uint32
	ProcVictim   uint32
}

// CalcDamage represents damage for a single weapon hand.
type CalcDamage struct {
	Damage   uint32
	Absorb   uint32
	Resist   uint32
	Block    uint32
	Clean    uint32 // damage mitigated but not to absorb/resist
}

// MeleeDamageInfo represents a single damage instance for DealDamage.
// Maps to AC's DamageInfo class (Unit.h:336).
// Note: Renamed to MeleeDamageInfo to avoid conflict with spell_proc.DamageInfo.
type MeleeDamageInfo struct {
	Attacker   CombatUnit
	Victim     CombatUnit
	Damage     uint32
	SpellInfo  *SpellInfo // nil for melee
	SchoolMask SpellSchoolMask
	DamageType DamageEffectType
	AttackType WeaponAttackType
	Absorb     uint32
	Resist     uint32
	Block      uint32
	HitMask    uint32
}

// GetDamage returns damage after absorb/resist.
func (d *MeleeDamageInfo) GetDamage() uint32 {
	if d.Damage <= d.Absorb+d.Resist {
		return 0
	}
	return d.Damage - d.Absorb - d.Resist
}

// GetUnmitigatedDamage returns raw damage before any mitigation.
func (d *MeleeDamageInfo) GetUnmitigatedDamage() uint32 {
	return d.Damage
}

// ModifyDamage adjusts the damage by amount (can be negative).
func (d *MeleeDamageInfo) ModifyDamage(amount int32) {
	newDmg := int32(d.Damage) + amount
	if newDmg < 0 {
		newDmg = 0
	}
	d.Damage = uint32(newDmg)
}

// AbsorbDamage reduces damage by the absorb amount.
func (d *MeleeDamageInfo) AbsorbDamage(amount uint32) {
	if amount > d.Damage {
		amount = d.Damage
	}
	d.Absorb += amount
	d.Damage -= amount
}

// ResistDamage reduces damage by the resist amount.
func (d *MeleeDamageInfo) ResistDamage(amount uint32) {
	if amount > d.Damage {
		amount = d.Damage
	}
	d.Resist += amount
	d.Damage -= amount
}

// BlockDamage reduces damage by the block amount.
func (d *MeleeDamageInfo) BlockDamage(amount uint32) {
	if amount > d.Damage {
		amount = d.Damage
	}
	d.Block += amount
	d.Damage -= amount
}

// DuelInfo holds duel state for a player.
// Maps to AC's DuelInfo struct (Player.h:363).
type DuelInfo struct {
	Opponent        *player.Player
	Initiator       *player.Player
	IsMounted       bool
	State           DuelState
	StartTime       time.Time
	OutOfBoundsTime time.Time
	FlagGUID        wow.GUID
}

// CombatUnit is the interface that both Player and NPC must implement
// to participate in the combat system.
type CombatUnit interface {
	GetGUID() wow.GUID
	GetLevel() uint32
	GetHealth() uint32
	GetMaxHealth() uint32
	SetHealth(uint32)
	IsAlive() bool
	IsPlayer() bool
	IsCreature() bool
	IsPet() bool
	IsTotem() bool

	// Position
	GetPositionX() float32
	GetPositionY() float32
	GetPositionZ() float32
	GetMapID() uint32

	// Combat stats
	GetAttackPower() uint32
	GetWeaponDamage(attackType uint8) (min, max uint32)
	GetWeaponSpeed(attackType uint8) time.Duration
	GetCombatRating(rating uint8) uint32
	GetArmor() uint32
	GetBlockChance() float32
	GetDodgeChance() float32
	GetParryChance() float32
	GetCritChance() float32

	// Power
	GetPrimaryPowerType() wow.PowerType
	GetPower(pt wow.PowerType) uint32
	SetPower(pt wow.PowerType, v uint32)

	// Combat state
	GetVictim() interface{}
	SetVictim(v interface{})
	AddThreat(attacker interface{}, threat float32)
	CombatStop()

	// Attack validation
	IsMounted() bool
	IsInSameMap(other interface{}) bool
	IsHostileTo(other interface{}) bool
	IsFriendlyTo(other interface{}) bool
	IsValidAttackTarget(target interface{}) bool

	// Combat state management
	AddAttacker(guid wow.GUID)
	RemoveAttacker(guid wow.GUID)
	HasAttacker(guid wow.GUID) bool
	GetAttackers() map[uint64]bool
	SetInCombat()
	ClearInCombat()
	IsInCombatState() bool

	// AI hooks
	OnDamageTaken(attacker interface{}, damage uint32)
	OnDamageDealt(victim interface{}, damage uint32)

	// Attack management
	Attack(victim interface{}, meleeAttack bool)
	AttackStop()

	// Duel
	GetDuelInfo() interface{}
	SetDuelInfo(info interface{})
}
