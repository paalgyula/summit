package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

// ActiveSpell tracks a spell currently being cast.
type ActiveSpell struct {
	Spell       *Spell
	Caster      *player.Player
	Target      CombatUnit // can be nil for self-targeted spells
	TargetGUID  uint64
	State       SpellState
	StartTime   time.Time
	CastEndTime time.Time
	CastTime    time.Duration
	Interrupted bool
}

// Aura/AuraEffect types are defined in aura.go.
