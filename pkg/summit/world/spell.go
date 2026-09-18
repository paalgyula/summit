package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

// SpellTemplate defines a spell's properties.
type SpellTemplate struct {
	ID            uint32
	Name          string
	IconID        uint32
	SchoolMask    SpellSchoolMask
	CastTime      time.Duration // time to cast
	Cooldown      time.Duration // cooldown after cast
	Range         float32       // max range
	MaxStack      uint8         // max stack count (1 = no stacking)
	Duration      time.Duration // buff duration (0 = instant)
	TriggerChance float32       // chance to trigger (0-1)

	// Effects (up to 3 effects per spell)
	Effects [3]SpellEffectData

	// Power cost
	PowerType SpellPowerType
	PowerCost uint32

	// Target type
	TargetType SpellTargetType

	// Description
	Description string
}

// SpellEffectData defines a single effect of a spell.
type SpellEffectData struct {
	Type       SpellEffect
	BasePoints int32  // base value for the effect
	RealPoints int32  // final calculated value (base + level scaling)
	TriggerAura AuraType // if type is ApplyAura, which aura
	ApplyAuraPeriod time.Duration // for periodic auras
	Duration    time.Duration // total duration for the effect (for auras)
}

// ActiveSpell tracks a spell currently being cast.
type ActiveSpell struct {
	Spell       *SpellTemplate
	Caster      *player.Player
	Target      *player.Player // can be nil for self-targeted spells
	TargetGUID  uint64
	State       SpellState
	StartTime   time.Time
	CastEndTime time.Time
	CastTime    time.Duration
	Interrupted bool
}

// Aura represents an active buff/debuff on a player.
type Aura struct {
	SpellID    uint32
	Spell      *SpellTemplate
	CasterGUID uint64
	Duration   time.Duration
	Remaining  time.Duration
	StackCount uint8
	StartTime  time.Time
	Effects    []AuraEffect
}

// AuraEffect represents a single effect of an aura.
type AuraEffect struct {
	Type       AuraType
	Value      int32
	Period     time.Duration
	LastTick   time.Time
}

// SpellManager holds all spell templates and manages casting.
type SpellManager struct {
	templates map[uint32]*SpellTemplate
}

// NewSpellManager creates a new spell manager with default spells.
func NewSpellManager() *SpellManager {
	sm := &SpellManager{
		templates: make(map[uint32]*SpellTemplate),
	}

	sm.registerDefaultSpells()

	return sm
}

// registerDefaultSpells registers example spells for testing.
func (sm *SpellManager) registerDefaultSpells() {
	// Mage: Fireball (Direct damage + DoT)
	sm.templates[133] = &SpellTemplate{
		ID:         133,
		Name:       "Fireball",
		IconID:     134,
		SchoolMask: SpellSchoolMaskFire,
		CastTime:   1500 * time.Millisecond,
		Cooldown:   0,
		Range:      30.0,
		MaxStack:   1,
		Duration:   0,
		PowerType:  SpellPowerMana,
		PowerCost:  20,
		TargetType: SpellTargetEnemy,
		Description: "Launches a ball of fire at the target, dealing Fire damage.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectSchoolDamage, BasePoints: 25, TriggerAura: AuraNone},
			{Type: SpellEffectApplyAura, BasePoints: 5, TriggerAura: AuraType(AuraPeriodicDamage), ApplyAuraPeriod: 2 * time.Second, Duration: 6 * time.Second},
		},
	}

	// Priest: Heal (Direct heal)
	sm.templates[2061] = &SpellTemplate{
		ID:         2061,
		Name:       "Flash Heal",
		IconID:     135,
		SchoolMask: SpellSchoolMaskHoly,
		CastTime:   1500 * time.Millisecond,
		Cooldown:   0,
		Range:      40.0,
		MaxStack:   1,
		Duration:   0,
		PowerType:  SpellPowerMana,
		PowerCost:  25,
		TargetType: SpellTargetFriendly,
		Description: "Heals a friendly target for a moderate amount.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectHeal, BasePoints: 80, TriggerAura: AuraNone},
		},
	}

	// Priest: Power Word: Shield (Absorb buff)
	sm.templates[17] = &SpellTemplate{
		ID:         17,
		Name:       "Power Word: Shield",
		IconID:     136,
		SchoolMask: SpellSchoolMaskHoly,
		CastTime:   0, // instant
		Cooldown:   0,
		Range:      30.0,
		MaxStack:   1,
		Duration:   15 * time.Second,
		PowerType:  SpellPowerMana,
		PowerCost:  30,
		TargetType: SpellTargetFriendly,
		Description: "Shields a friendly target, absorbing damage.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectApplyAura, BasePoints: 50, TriggerAura: AuraType(AuraModModDamagePercentDone)},
		},
	}

	// Warrior: Battle Shout (Self buff)
	sm.templates[6673] = &SpellTemplate{
		ID:         6673,
		Name:       "Battle Shout",
		IconID:     137,
		SchoolMask: SpellSchoolMaskPhysical,
		CastTime:   0, // instant
		Cooldown:   0,
		Range:      0, // self only
		MaxStack:   1,
		Duration:   120 * time.Second,
		PowerType:  SpellPowerRage,
		PowerCost:  10,
		TargetType: SpellTargetSelf,
		Description: "Increase melee attack power.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectApplyAura, BasePoints: 10, TriggerAura: AuraModModDamageDone},
		},
	}

	// Rogue: Eviscerate (Direct damage)
	sm.templates[2098] = &SpellTemplate{
		ID:         2098,
		Name:       "Eviscerate",
		IconID:     138,
		SchoolMask: SpellSchoolMaskPhysical,
		CastTime:   0, // instant
		Cooldown:   0,
		Range:      5.0, // melee range
		MaxStack:   1,
		Duration:   0,
		PowerType:  SpellPowerEnergy,
		PowerCost:  35,
		TargetType: SpellTargetEnemy,
		Description: "Finishing move that causes Physical damage.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectSchoolDamage, BasePoints: 30, TriggerAura: AuraNone},
		},
	}

	// Warlock: Shadow Bolt (Direct damage)
	sm.templates[686] = &SpellTemplate{
		ID:         686,
		Name:       "Shadow Bolt",
		IconID:     139,
		SchoolMask: SpellSchoolMaskShadow,
		CastTime:   2000 * time.Millisecond,
		Cooldown:   0,
		Range:      30.0,
		MaxStack:   1,
		Duration:   0,
		PowerType:  SpellPowerMana,
		PowerCost:  25,
		TargetType: SpellTargetEnemy,
		Description: "Sends a shadow bolt at the target.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectSchoolDamage, BasePoints: 30, TriggerAura: AuraNone},
		},
	}

	// Shaman: Lightning Bolt (Direct damage)
	sm.templates[403] = &SpellTemplate{
		ID:         403,
		Name:       "Lightning Bolt",
		IconID:     140,
		SchoolMask: SpellSchoolMaskNature,
		CastTime:   1500 * time.Millisecond,
		Cooldown:   0,
		Range:      30.0,
		MaxStack:   1,
		Duration:   0,
		PowerType:  SpellPowerMana,
		PowerCost:  20,
		TargetType: SpellTargetEnemy,
		Description: "Blasts the target with lightning.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectSchoolDamage, BasePoints: 20, TriggerAura: AuraNone},
		},
	}

	// Druid: Wrath (Direct damage)
	sm.templates[5176] = &SpellTemplate{
		ID:         5176,
		Name:       "Wrath",
		IconID:     141,
		SchoolMask: SpellSchoolMaskNature,
		CastTime:   1500 * time.Millisecond,
		Cooldown:   0,
		Range:      30.0,
		MaxStack:   1,
		Duration:   0,
		PowerType:  SpellPowerMana,
		PowerCost:  20,
		TargetType: SpellTargetEnemy,
		Description: "Causes Nature damage to the target.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectSchoolDamage, BasePoints: 18, TriggerAura: AuraNone},
		},
	}

	// Paladin: Holy Light (Direct heal)
	sm.templates[635] = &SpellTemplate{
		ID:         635,
		Name:       "Holy Light",
		IconID:     142,
		SchoolMask: SpellSchoolMaskHoly,
		CastTime:   2000 * time.Millisecond,
		Cooldown:   0,
		Range:      40.0,
		MaxStack:   1,
		Duration:   0,
		PowerType:  SpellPowerMana,
		PowerCost:  30,
		TargetType: SpellTargetFriendly,
		Description: "Heals a friendly target.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectHeal, BasePoints: 60, TriggerAura: AuraNone},
		},
	}

	// Hunter: Serpent Sting (DoT)
	sm.templates[1978] = &SpellTemplate{
		ID:         1978,
		Name:       "Serpent Sting",
		IconID:     143,
		SchoolMask: SpellSchoolMaskNature,
		CastTime:   0, // instant
		Cooldown:   0,
		Range:      30.0,
		MaxStack:   1,
		Duration:   15 * time.Second,
		PowerType:  SpellPowerMana,
		PowerCost:  15,
		TargetType: SpellTargetEnemy,
		Description: "Stings the target, causing Nature damage over time.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectApplyAura, BasePoints: 8, TriggerAura: AuraType(AuraPeriodicDamage), ApplyAuraPeriod: 3 * time.Second, Duration: 15 * time.Second},
		},
	}

	// Example: Regeneration (HoT)
	sm.templates[10917] = &SpellTemplate{
		ID:         10917,
		Name:       "Rejuvenation",
		IconID:     144,
		SchoolMask: SpellSchoolMaskNature,
		CastTime:   0, // instant
		Cooldown:   0,
		Range:      40.0,
		MaxStack:   1,
		Duration:   12 * time.Second,
		PowerType:  SpellPowerMana,
		PowerCost:  35,
		TargetType: SpellTargetFriendly,
		Description: "Heals the target over time.",
		Effects: [3]SpellEffectData{
			{Type: SpellEffectApplyAura, BasePoints: 20, TriggerAura: AuraType(AuraPeriodicHeal), ApplyAuraPeriod: 3 * time.Second, Duration: 12 * time.Second},
		},
	}
}

// GetSpell returns a spell template by ID.
func (sm *SpellManager) GetSpell(id uint32) *SpellTemplate {
	return sm.templates[id]
}

// RegisterSpell adds a spell template to the manager.
func (sm *SpellManager) RegisterSpell(spell *SpellTemplate) {
	sm.templates[spell.ID] = spell
}
