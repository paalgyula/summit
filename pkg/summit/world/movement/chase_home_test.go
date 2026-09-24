package movement

import (
	"testing"
)

// mockCombatTarget implements positionGetter for testing chase.
type mockCombatTarget struct {
	x, y, z float32
	alive   bool
}

func (m *mockCombatTarget) GetPositionX() float32 { return m.x }
func (m *mockCombatTarget) GetPositionY() float32 { return m.y }
func (m *mockCombatTarget) GetPositionZ() float32 { return m.z }
func (m *mockCombatTarget) IsAlive() bool          { return m.alive }

func TestChaseGenerator_Type(t *testing.T) {
	target := &mockCombatTarget{x: 110, y: 210, z: 30, alive: true}
	gen := NewChaseGenerator(target, 50.0)
	if gen.Type() != MotionTypeCHASE {
		t.Errorf("Expected CHASE type, got %v", gen.Type())
	}
}

func TestChaseGenerator_Initialize(t *testing.T) {
	target := &mockCombatTarget{x: 110, y: 210, z: 30, alive: true}
	gen := NewChaseGenerator(target, 50.0)
	gen.Initialize(nil)

	if gen.recalculateTimer != 0 {
		t.Error("recalculateTimer should be 0 after Initialize")
	}
}

func TestChaseGenerator_Update_DeadTarget(t *testing.T) {
	owner := newMockOwner()
	target := &mockCombatTarget{x: 110, y: 210, z: 30, alive: false}
	gen := NewChaseGenerator(target, 50.0)
	gen.Initialize(nil)

	// Should return false when target is dead
	result := gen.Update(owner, 100)
	if result {
		t.Error("Update should return false when target is dead")
	}
}

func TestChaseGenerator_Update_InMeleeRange(t *testing.T) {
	owner := newMockOwner()
	owner.x = 100
	owner.y = 200
	owner.z = 30

	// Target very close (within 3.5 yards)
	target := &mockCombatTarget{x: 101, y: 200, z: 30, alive: true}
	gen := NewChaseGenerator(target, 50.0)
	gen.Initialize(nil)

	// Should return true (still active) when in melee range
	result := gen.Update(owner, 100)
	if !result {
		t.Error("Update should return true when in melee range")
	}
}

func TestChaseGenerator_Update_FarAway(t *testing.T) {
	owner := newMockOwner()
	owner.x = 100
	owner.y = 200
	owner.z = 30

	// Target far away (20 yards)
	target := &mockCombatTarget{x: 120, y: 200, z: 30, alive: true}
	gen := NewChaseGenerator(target, 50.0)
	gen.Initialize(nil)

	// First update with timer expired should move
	gen.recalculateTimer = gen.recalcInterval + 1
	result := gen.Update(owner, 100)
	if !result {
		t.Error("Update should return true when chasing")
	}

	// Should have updated last target position
	if gen.lastTargetPos[0] != 120 {
		t.Errorf("lastTargetPos[0] should be 120, got %v", gen.lastTargetPos[0])
	}
}

func TestChaseGenerator_Update_ExceedsLeash(t *testing.T) {
	owner := newMockOwner()
	owner.x = 100
	owner.y = 200
	owner.z = 30

	// Target beyond leash range (60 yards, leash is 50)
	target := &mockCombatTarget{x: 160, y: 200, z: 30, alive: true}
	gen := NewChaseGenerator(target, 50.0)
	gen.Initialize(nil)

	// Should return false when exceeding leash
	result := gen.Update(owner, 100)
	if result {
		t.Error("Update should return false when exceeding leash range")
	}
}

func TestChaseGenerator_GetTarget(t *testing.T) {
	target := &mockCombatTarget{x: 110, y: 210, z: 30, alive: true}
	gen := NewChaseGenerator(target, 50.0)

	if gen.GetTarget() != target {
		t.Error("GetTarget should return the target")
	}
}

func TestChaseGenerator_SetTarget(t *testing.T) {
	target1 := &mockCombatTarget{x: 110, y: 210, z: 30, alive: true}
	target2 := &mockCombatTarget{x: 120, y: 220, z: 30, alive: true}
	gen := NewChaseGenerator(target1, 50.0)

	gen.SetTarget(target2)

	if gen.GetTarget() != target2 {
		t.Error("GetTarget should return the new target after SetTarget")
	}
}

func TestHomeGenerator_Type(t *testing.T) {
	gen := NewHomeGenerator(false)
	if gen.Type() != MotionTypeHOME {
		t.Errorf("Expected HOME type, got %v", gen.Type())
	}
}

func TestHomeGenerator_Initialize(t *testing.T) {
	gen := NewHomeGenerator(false)
	gen.Initialize(nil)

	if gen.arrived {
		t.Error("arrived should be false after Initialize")
	}
}

func TestHomeGenerator_Update_AtSpawn(t *testing.T) {
	owner := newMockOwner()
	owner.x = owner.spawnX
	owner.y = owner.spawnY
	owner.z = owner.spawnZ

	gen := NewHomeGenerator(false)
	gen.Initialize(nil)

	// Should return false when already at spawn
	result := gen.Update(owner, 100)
	if result {
		t.Error("Update should return false when at spawn")
	}

	if !gen.HasArrived() {
		t.Error("HasArrived should be true when at spawn")
	}
}

func TestHomeGenerator_Update_MovingHome(t *testing.T) {
	owner := newMockOwner()
	owner.x = 150
	owner.y = 250
	owner.z = 30

	gen := NewHomeGenerator(false)
	gen.Initialize(nil)

	// Should return true while moving home
	result := gen.Update(owner, 100)
	if !result {
		t.Error("Update should return true while moving home")
	}
}

func TestHomeGenerator_Update_DeadOwner(t *testing.T) {
	owner := newMockOwner()
	owner.alive = false

	gen := NewHomeGenerator(false)
	gen.Initialize(nil)

	// Should return false when owner is dead
	result := gen.Update(owner, 100)
	if result {
		t.Error("Update should return false when owner is dead")
	}
}

func TestHomeGenerator_HasArrived(t *testing.T) {
	gen := NewHomeGenerator(false)

	if gen.HasArrived() {
		t.Error("HasArrived should be false initially")
	}

	gen.arrived = true

	if !gen.HasArrived() {
		t.Error("HasArrived should be true after setting arrived")
	}
}

func TestHomeGenerator_Reset(t *testing.T) {
	gen := NewHomeGenerator(false)
	gen.arrived = true

	gen.Reset(nil)

	if gen.arrived {
		t.Error("arrived should be false after Reset")
	}
}

func TestHomeGenerator_WalkFlag(t *testing.T) {
	gen := NewHomeGenerator(true)

	if !gen.walk {
		t.Error("walk should be true when created with walk=true")
	}
}
