package movement

import (
	"math"
)

// DefaultCreatureAI is the default AI implementation for creatures.
// It handles basic combat behavior: chase on enter combat, return home on exit.
// It mirrors AzerothCore's CreatureAI / AggressorAI.
type DefaultCreatureAI struct {
	owner     MovementOwner
	mm        *MotionMaster
	tm        *ThreatManager
	engaged   bool
	victim    interface{}
	leashDist float32
}

// NewDefaultCreatureAI creates a DefaultCreatureAI for the given owner.
func NewDefaultCreatureAI(owner MovementOwner, mm *MotionMaster) *DefaultCreatureAI {
	return &DefaultCreatureAI{
		owner:     owner,
		mm:        mm,
		leashDist: 50.0, // default 50 yard leash range
	}
}

// SetThreatManager associates a ThreatManager with this AI.
func (ai *DefaultCreatureAI) SetThreatManager(tm *ThreatManager) {
	ai.tm = tm
}

// SetLeashDistance sets the max chase leash distance from spawn.
func (ai *DefaultCreatureAI) SetLeashDistance(dist float32) {
	if dist > 0 {
		ai.leashDist = dist
	}
}

// UpdateAI is called every tick to process combat and movement.
func (ai *DefaultCreatureAI) UpdateAI(diff uint32) {
	if ai.owner == nil || !ai.owner.IsAlive() {
		return
	}

	if !ai.engaged {
		return
	}

	type aliveChecker interface {
		IsAlive() bool
	}

	// 1. Check if current victim is valid and alive
	victimAlive := false
	if ai.victim != nil {
		if ac, ok := ai.victim.(aliveChecker); ok {
			victimAlive = ac.IsAlive()
		} else {
			victimAlive = true
		}
	}

	if !victimAlive {
		// Try to select another alive victim from threat list
		if ai.tm != nil {
			newVictim := ai.tm.SelectVictim(func(guid interface{}) bool {
				if ac, ok := guid.(aliveChecker); ok {
					return ac.IsAlive()
				}
				return true
			})
			if newVictim != nil {
				ai.victim = newVictim
				if ai.mm != nil {
					ai.mm.MoveChase(newVictim, ai.leashDist)
				}
				return
			}
		}

		// No more valid victims, evade and go home
		ai.ExitCombat()
		return
	}

	// 2. Check leash distance from spawn
	ox, oy, oz := ai.owner.GetCurrentPosition()
	sx, sy, sz := ai.owner.GetSpawnPosition()
	dx := sx - ox
	dy := sy - oy
	dz := sz - oz
	distFromSpawn := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if distFromSpawn > ai.leashDist {
		ai.ExitCombat()
		return
	}

	// 3. Ensure chase generator is active if engaged
	if ai.mm != nil && !ai.mm.HasMovementGeneratorType(MotionTypeCHASE) {
		if !ai.mm.HasMovementGeneratorType(MotionTypeHOME) && ai.victim != nil {
			ai.mm.MoveChase(ai.victim, ai.leashDist)
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

	if ai.tm != nil && who != nil {
		ai.tm.AddThreat(who, 100.0)
		ai.tm.SetVictim(who)
	}

	// Start chasing the target
	if ai.mm != nil && who != nil {
		ai.mm.MoveChase(who, ai.leashDist)
	}
}

// ExitCombat is called when the creature leaves combat.
func (ai *DefaultCreatureAI) ExitCombat() {
	ai.engaged = false
	ai.victim = nil

	if ai.tm != nil {
		ai.tm.ClearThreat()
	}

	// Start returning home
	if ai.mm != nil {
		ai.mm.MoveTargetedHome()
	}
}

// JustDied is called when the creature dies.
func (ai *DefaultCreatureAI) JustDied(killer interface{}) {
	ai.engaged = false
	ai.victim = nil

	if ai.tm != nil {
		ai.tm.ClearThreat()
	}

	// Clear all movement
	if ai.mm != nil {
		ai.mm.Clear(false)
	}
}

// MovementInform is called when a movement generator completes.
func (ai *DefaultCreatureAI) MovementInform(type_ MovementGeneratorType, pointID uint32) {
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
	if ai.tm != nil {
		ai.tm.SetVictim(victim)
	}
	if ai.mm != nil && victim != nil && ai.engaged {
		ai.mm.MoveChase(victim, ai.leashDist)
	}
}
