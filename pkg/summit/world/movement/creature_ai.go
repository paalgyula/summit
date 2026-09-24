package movement

// CreatureAI is the interface that creature AI scripts implement.
// It mirrors AzerothCore's CreatureAI class.
type CreatureAI interface {
	// UpdateAI is called every tick to process combat and movement.
	UpdateAI(diff uint32)

	// EnterCombat is called when the creature enters combat.
	EnterCombat(who interface{})

	// ExitCombat is called when the creature leaves combat.
	ExitCombat()

	// JustDied is called when the creature dies.
	JustDied(killer interface{})

	// MovementInform is called when a movement generator completes.
	MovementInform(type_ MovementGeneratorType, pointID uint32)

	// IsEngaged returns true if the creature is in combat.
	IsEngaged() bool
}

// CombatUnit is the interface for units that can participate in combat.
type CombatUnit interface {
	GetGUID() interface{}
	IsAlive() bool
	GetPositionX() float32
	GetPositionY() float32
	GetPositionZ() float32
}
