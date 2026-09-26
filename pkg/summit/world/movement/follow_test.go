package movement

import (
	"math"
	"testing"
)

type mockFollowTarget struct {
	x, y, z     float32
	orientation float32
	alive       bool
}

func (m *mockFollowTarget) GetPositionX() float32   { return m.x }
func (m *mockFollowTarget) GetPositionY() float32   { return m.y }
func (m *mockFollowTarget) GetPositionZ() float32   { return m.z }
func (m *mockFollowTarget) GetOrientation() float32 { return m.orientation }
func (m *mockFollowTarget) IsAlive() bool           { return m.alive }

func TestFollowGenerator_Type(t *testing.T) {
	target := &mockFollowTarget{x: 100, y: 100, z: 0, orientation: 0, alive: true}
	gen := NewFollowGenerator(target, 2.0, math.Pi)
	if gen.Type() != MotionTypeFOLLOW {
		t.Errorf("Expected FOLLOW type, got %v", gen.Type())
	}
}

func TestFollowGenerator_DeadTarget(t *testing.T) {
	owner := newMockOwner()
	target := &mockFollowTarget{x: 100, y: 100, z: 0, orientation: 0, alive: false}
	gen := NewFollowGenerator(target, 2.0, math.Pi)
	gen.Initialize(nil)

	if gen.Update(owner, 100) {
		t.Error("Update should return false when target is dead")
	}
}

func TestFollowGenerator_WithinRange(t *testing.T) {
	owner := newMockOwner()
	owner.x = 100
	owner.y = 100
	owner.z = 0

	// Target 1.5 yards away, well within range (2.0 + 0.5)
	target := &mockFollowTarget{x: 101.5, y: 100, z: 0, orientation: 0, alive: true}
	gen := NewFollowGenerator(target, 2.0, 0)
	gen.Initialize(nil)

	if !gen.Update(owner, 100) {
		t.Error("Update should return true when following within range")
	}
}

func TestFollowGenerator_MoveFollow_MotionMaster(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	target := &mockFollowTarget{x: 120, y: 100, z: 0, orientation: 0, alive: true}

	mm.MoveFollow(target, 3.0, 0)

	if !mm.HasMovementGeneratorType(MotionTypeFOLLOW) {
		t.Error("MotionMaster should have FOLLOW generator after MoveFollow")
	}
	if mm.GetCurrentMovementGeneratorType() != MotionTypeFOLLOW {
		t.Errorf("Expected FOLLOW on top, got %v", mm.GetCurrentMovementGeneratorType())
	}
}
