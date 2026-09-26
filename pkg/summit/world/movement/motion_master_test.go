package movement

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/wow"
)

// mockOwner implements MovementOwner for testing.
type mockOwner struct {
	x, y, z         float32
	spawnX, spawnY, spawnZ float32
	alive           bool
	inCombat        bool
	movementType    uint8
	wanderDist      float32
	entry           uint32
}

func newMockOwner() *mockOwner {
	return &mockOwner{
		x: 100, y: 200, z: 30,
		spawnX: 100, spawnY: 200, spawnZ: 30,
		alive:    true,
		inCombat: false,
		movementType: 0, // idle
		wanderDist: 5.0,
		entry:    1234,
	}
}

func (m *mockOwner) GetSpawnPosition() (float32, float32, float32) {
	return m.spawnX, m.spawnY, m.spawnZ
}

func (m *mockOwner) GetCurrentPosition() (float32, float32, float32) {
	return m.x, m.y, m.z
}

func (m *mockOwner) IsAlive() bool     { return m.alive }
func (m *mockOwner) IsInCombat() bool  { return m.inCombat }
func (m *mockOwner) GetDefaultMovementType() uint8 { return m.movementType }
func (m *mockOwner) GetWanderDistance() float32     { return m.wanderDist }
func (m *mockOwner) GetEntry() uint32               { return m.entry }
func (m *mockOwner) GetGUID() wow.GUID              { return wow.NewGUID(wow.UnitGUID, 1) }
func (m *mockOwner) GetCurrentSpeed(_ wow.MoveType) float32 { return 7.0 }
func (m *mockOwner) SendPacket(_ *wow.Packet) {}
func (m *mockOwner) MoveTo(_ float32, _ float32, _ float32, _ time.Time, _ uint32, _ func(*wow.Packet)) {}
func (m *mockOwner) StopMoving(_ func(*wow.Packet)) {}
func (m *mockOwner) SetOrientation(_ float32) {}
func (m *mockOwner) GetOrientation() float32 { return 0 }
func (m *mockOwner) HasActiveMovement() bool { return false }

func TestMotionMaster_InitialState(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)

	if !mm.empty() {
		t.Error("MotionMaster should be empty after creation")
	}

	if mm.Top() != nil {
		t.Error("Top should be nil when empty")
	}

	if mm.size() != 0 {
		t.Errorf("size() should be 0, got %d", mm.size())
	}
}

func TestMotionMaster_InitDefault_Idle(t *testing.T) {
	owner := newMockOwner()
	owner.movementType = 0 // idle
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	if mm.empty() {
		t.Error("MotionMaster should not be empty after InitDefault")
	}

	top := mm.Top()
	if top == nil {
		t.Fatal("Top should not be nil after InitDefault")
	}

	if top.Type() != MotionTypeIDLE {
		t.Errorf("Expected IDLE generator, got %v", top.Type())
	}
}

func TestMotionMaster_InitDefault_Random(t *testing.T) {
	owner := newMockOwner()
	owner.movementType = 1 // random
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	top := mm.Top()
	if top == nil {
		t.Fatal("Top should not be nil after InitDefault")
	}

	if top.Type() != MotionTypeRANDOM {
		t.Errorf("Expected RANDOM generator, got %v", top.Type())
	}
}

func TestMotionMaster_MoveIdle(t *testing.T) {
	owner := newMockOwner()
	owner.movementType = 1 // random
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	// Should have random generator
	if mm.Top().Type() != MotionTypeRANDOM {
		t.Errorf("Expected RANDOM, got %v", mm.Top().Type())
	}

	// MoveIdle should replace with idle
	mm.MoveIdle()

	if mm.Top().Type() != MotionTypeIDLE {
		t.Errorf("Expected IDLE after MoveIdle, got %v", mm.Top().Type())
	}
}

func TestMotionMaster_MoveRandom(t *testing.T) {
	owner := newMockOwner()
	owner.movementType = 0 // idle
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	// Should have idle generator
	if mm.Top().Type() != MotionTypeIDLE {
		t.Errorf("Expected IDLE, got %v", mm.Top().Type())
	}

	// MoveRandom should replace with random
	mm.MoveRandom(10.0)

	if mm.Top().Type() != MotionTypeRANDOM {
		t.Errorf("Expected RANDOM after MoveRandom, got %v", mm.Top().Type())
	}
}

func TestMotionMaster_Mutate_HigherSlot(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	// Should have idle at slot 0
	if mm.Top().Type() != MotionTypeIDLE {
		t.Errorf("Expected IDLE, got %v", mm.Top().Type())
	}

	// Mutate ACTIVE slot (slot 1) with a chase generator
	chaseGen := NewChaseGenerator(nil, 50.0)
	mm.Mutate(chaseGen, MotionSlotACTIVE)

	// Top should now be the chase generator
	if mm.Top().Type() != MotionTypeCHASE {
		t.Errorf("Expected CHASE as top, got %v", mm.Top().Type())
	}

	// Idle should still be in slot 0
	idleGen := mm.GetMotionSlot(MotionSlotIDLE)
	if idleGen == nil {
		t.Error("IDLE slot should still have a generator")
	} else if idleGen.Type() != MotionTypeIDLE {
		t.Errorf("IDLE slot should have IDLE generator, got %v", idleGen.Type())
	}
}

func TestMotionMaster_Mutate_LowerSlot(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)

	// Start with chase in ACTIVE slot
	chaseGen := NewChaseGenerator(nil, 50.0)
	mm.Mutate(chaseGen, MotionSlotACTIVE)

	if mm.Top().Type() != MotionTypeCHASE {
		t.Errorf("Expected CHASE, got %v", mm.Top().Type())
	}

	// Mutate IDLE slot (slot 0) - should not change top
	idleGen := NewIdleGenerator()
	mm.Mutate(idleGen, MotionSlotIDLE)

	if mm.Top().Type() != MotionTypeCHASE {
		t.Errorf("Top should still be CHASE, got %v", mm.Top().Type())
	}
}

func TestMotionMaster_Update_PopsFinishedGenerator(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)

	// Create a generator that finishes immediately
	finishGen := &finishGenerator{finished: true}
	mm.Mutate(finishGen, MotionSlotACTIVE)

	// Also have an idle generator below
	idleGen := NewIdleGenerator()
	mm.Mutate(idleGen, MotionSlotIDLE)

	// Top should be the finish generator
	if mm.Top().Type() != MotionTypeEFFECT {
		t.Errorf("Expected EFFECT, got %v", mm.Top().Type())
	}

	// Update should pop the finished generator
	mm.Update(100)

	// Top should now be idle
	if mm.Top().Type() != MotionTypeIDLE {
		t.Errorf("Expected IDLE after update, got %v", mm.Top().Type())
	}
}

func TestMotionMaster_HasMovementGeneratorType(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)
	mm.InitDefault()

	if !mm.HasMovementGeneratorType(MotionTypeIDLE) {
		t.Error("Should have IDLE generator type")
	}

	if mm.HasMovementGeneratorType(MotionTypeCHASE) {
		t.Error("Should not have CHASE generator type")
	}

	// Add chase generator
	chaseGen := NewChaseGenerator(nil, 50.0)
	mm.Mutate(chaseGen, MotionSlotACTIVE)

	if !mm.HasMovementGeneratorType(MotionTypeCHASE) {
		t.Error("Should have CHASE generator type after mutation")
	}
}

func TestMotionMaster_GetCurrentMovementGeneratorType(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)

	// Empty should return IDLE
	if mm.GetCurrentMovementGeneratorType() != MotionTypeIDLE {
		t.Error("Empty MotionMaster should return IDLE type")
	}

	mm.InitDefault()
	if mm.GetCurrentMovementGeneratorType() != MotionTypeIDLE {
		t.Error("Should return IDLE type after init")
	}

	chaseGen := NewChaseGenerator(nil, 50.0)
	mm.Mutate(chaseGen, MotionSlotACTIVE)
	if mm.GetCurrentMovementGeneratorType() != MotionTypeCHASE {
		t.Error("Should return CHASE type after mutation")
	}
}

func TestMotionMaster_Clear(t *testing.T) {
	owner := newMockOwner()
	mm := NewMotionMaster(owner)

	// Add chase and idle generators
	chaseGen := NewChaseGenerator(nil, 50.0)
	mm.Mutate(chaseGen, MotionSlotACTIVE)
	idleGen := NewIdleGenerator()
	mm.Mutate(idleGen, MotionSlotIDLE)

	// Clear should remove all but idle
	mm.Clear(true)

	// Should have idle generator
	if mm.Top().Type() != MotionTypeIDLE {
		t.Errorf("Expected IDLE after Clear, got %v", mm.Top().Type())
	}

	// Should not have chase
	if mm.HasMovementGeneratorType(MotionTypeCHASE) {
		t.Error("Should not have CHASE after Clear")
	}
}

func TestIdleGenerator(t *testing.T) {
	gen := NewIdleGenerator()

	if gen.Type() != MotionTypeIDLE {
		t.Errorf("Expected IDLE type, got %v", gen.Type())
	}

	gen.Initialize(nil)

	// Update should always return true
	if !gen.Update(nil, 100) {
		t.Error("IdleGenerator.Update should always return true")
	}

	gen.Finalize(nil)
	gen.Reset(nil)
}

func TestMovementGeneratorType_String(t *testing.T) {
	tests := []struct {
		t    MovementGeneratorType
		want string
	}{
		{MotionTypeIDLE, "Idle"},
		{MotionTypeRANDOM, "Random"},
		{MotionTypeWAYPOINT, "Waypoint"},
		{MotionTypeCHASE, "Chase"},
		{MotionTypeHOME, "Home"},
		{MotionTypePOINT, "Point"},
		{MotionTypeFLEEING, "Fleeing"},
		{MotionTypeCONFUSED, "Confused"},
		{MovementGeneratorType(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestIsStatic(t *testing.T) {
	idleGen := NewIdleGenerator()
	if !IsStatic(idleGen) {
		t.Error("IdleGenerator should be static")
	}

	chaseGen := NewChaseGenerator(nil, 50.0)
	if IsStatic(chaseGen) {
		t.Error("ChaseGenerator should not be static")
	}

	if IsStatic(nil) {
		t.Error("nil should not be static")
	}
}

// finishGenerator is a test generator that finishes immediately.
type finishGenerator struct {
	finished bool
}

func (g *finishGenerator) Initialize(_ interface{})         {}
func (g *finishGenerator) Update(_ interface{}, _ uint32) bool { return !g.finished }
func (g *finishGenerator) Finalize(_ interface{})           {}
func (g *finishGenerator) Reset(_ interface{})              {}
func (g *finishGenerator) Type() MovementGeneratorType      { return MotionTypeEFFECT }
func (g *finishGenerator) GetSplineId() uint32              { return 0 }
