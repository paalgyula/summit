package movement

// DefaultCreatureAI is the default AI implementation for creatures.
// It handles basic combat behavior: chase on enter combat, return home on exit.
type DefaultCreatureAI struct {
	owner   MovementOwner
	mm      *MotionMaster
	engaged bool
	victim  interface{}
}

// NewDefaultCreatureAI creates a DefaultCreatureAI for the given owner.
func NewDefaultCreatureAI(owner MovementOwner, mm *MotionMaster) *DefaultCreatureAI {
	return &DefaultCreatureAI{
		owner: owner,
		mm:    mm,
	}
}

// UpdateAI is called every tick to process combat and movement.
func (ai *DefaultCreatureAI) UpdateAI(diff uint32) {
	if ai.owner == nil || !ai.owner.IsAlive() {
		return
	}

	// If engaged, ensure we have a chase target
	if ai.engaged && ai.mm != nil {
		// Check if we have a chase generator
		if !ai.mm.HasMovementGeneratorType(MotionTypeCHASE) {
			// Add chase generator for current victim
			if ai.victim != nil {
				ai.mm.Mutate(NewChaseGenerator(ai.victim, 50.0), MotionSlotACTIVE)
			}
		}
	}
}

// EnterCombat is called when the creature enters combat.
func (ai *DefaultCreatureAI) EnterCombat(who interface{}) {
	if ai.owner == nil || !ai.owner.IsAlive() {
		return
	}

	ai.engaged = true
	ai.victim = who

	// Start chasing the target
	if ai.mm != nil && who != nil {
		ai.mm.Mutate(NewChaseGenerator(who, 50.0), MotionSlotACTIVE)
	}
}

// ExitCombat is called when the creature leaves combat.
func (ai *DefaultCreatureAI) ExitCombat() {
	ai.engaged = false
	ai.victim = nil

	// Clear active slot (chase generator)
	if ai.mm != nil {
		ai.mm.Clear(false)
	}

	// Start returning home
	if ai.mm != nil {
		ai.mm.Mutate(NewHomeGenerator(false), MotionSlotACTIVE)
	}
}

// JustDied is called when the creature dies.
func (ai *DefaultCreatureAI) JustDied(killer interface{}) {
	ai.engaged = false
	ai.victim = nil

	// Clear all movement
	if ai.mm != nil {
		ai.mm.Clear(false)
	}
}

// MovementInform is called when a movement generator completes.
func (ai *DefaultCreatureAI) MovementInform(type_ MovementGeneratorType, pointID uint32) {
	// Default implementation does nothing; scripts can override
}

// IsEngaged returns true if the creature is in combat.
func (ai *DefaultCreatureAI) IsEngaged() bool {
	return ai.engaged
}

// GetVictim returns the current combat target.
func (ai *DefaultCreatureAI) GetVictim() interface{} {
	return ai.victim
}

// SetVictim sets the current combat target.
func (ai *DefaultCreatureAI) SetVictim(victim interface{}) {
	ai.victim = victim
}
