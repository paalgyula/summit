package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
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
	// CastCount / CastItem echo the client's CMSG_CAST_SPELL / CMSG_USE_ITEM so
	// the completion packet (SMSG_SPELL_GO) can be built when the cast ends.
	CastCount uint32
	CastItem  wow.GUID
}

// Aura/AuraEffect types are defined in aura.go.
