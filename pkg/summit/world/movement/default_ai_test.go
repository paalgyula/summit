package movement

import (
	"testing"
)

func TestDefaultCreatureAI_Type(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	ai := NewDefaultCreatureAI(owner, mm)

	if ai.IsEngaged() {
		t.Error("Should not be engaged initially")
	}
}

func TestDefaultCreatureAI_EnterCombat(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	ai := NewDefaultCreatureAI(owner, mm)
	victim := newMockOwner()

	ai.EnterCombat(victim)

	if !ai.IsEngaged() {
		t.Error("Should be engaged after EnterCombat")
	}

	if ai.GetVictim() != victim {
		t.Error("Victim should be set after EnterCombat")
	}
}

func TestDefaultCreatureAI_ExitCombat(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	ai := NewDefaultCreatureAI(owner, mm)
	victim := newMockOwner()

	ai.EnterCombat(victim)
	ai.ExitCombat()

	if ai.IsEngaged() {
		t.Error("Should not be engaged after ExitCombat")
	}

	if ai.GetVictim() != nil {
		t.Error("Victim should be nil after ExitCombat")
	}
}

func TestDefaultCreatureAI_JustDied(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	ai := NewDefaultCreatureAI(owner, mm)
	victim := newMockOwner()

	ai.EnterCombat(victim)
	ai.JustDied(victim)

	if ai.IsEngaged() {
		t.Error("Should not be engaged after JustDied")
	}

	if ai.GetVictim() != nil {
		t.Error("Victim should be nil after JustDied")
	}
}

func TestDefaultCreatureAI_UpdateAI_NilOwner(t *testing.T) {
	ai := &DefaultCreatureAI{}

	// Should not panic
	ai.UpdateAI(100)
}

func TestDefaultCreatureAI_EnterCombat_NilWho(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	ai := NewDefaultCreatureAI(owner, mm)

	// Should not panic
	ai.EnterCombat(nil)

	if !ai.IsEngaged() {
		t.Error("Should still be engaged even with nil target")
	}
}

func TestDefaultCreatureAI_SetVictim(t *testing.T) {
	ai := &DefaultCreatureAI{}
	victim := newMockOwner()

	ai.SetVictim(victim)

	if ai.GetVictim() != victim {
		t.Error("GetVictim should return the set victim")
	}
}

func TestDefaultCreatureAI_MovementInform(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	ai := NewDefaultCreatureAI(owner, mm)

	// Should not panic
	ai.MovementInform(MotionTypeCHASE, 0)
}
