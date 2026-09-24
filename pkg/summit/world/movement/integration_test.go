package movement

import (
	"testing"

	"github.com/paalgyula/summit/pkg/store"
)

// TestMotionMaster_FullLifecycle tests the complete lifecycle of a creature:
// spawn → idle → aggro → chase → evade → return home
func TestMotionMaster_FullLifecycle(t *testing.T) {
	owner := newMockOwner()
	owner.x = 100
	owner.y = 200
	owner.z = 30
	owner.spawnX = 100
	owner.spawnY = 200
	owner.spawnZ = 30
	owner.movementType = 1 // random
	owner.wanderDist = 5.0

	mm := NewMotionMaster(owner)
	mm.InitDefault()
	ai := NewDefaultCreatureAI(owner, mm)

	// 1. Verify idle state
	if mm.GetCurrentMovementGeneratorType() != MotionTypeRANDOM {
		t.Errorf("Expected RANDOM after init, got %v", mm.GetCurrentMovementGeneratorType())
	}

	// 2. Simulate aggro
	victim := newMockOwner()
	victim.x = 120
	victim.y = 200
	victim.z = 30

	ai.EnterCombat(victim)

	// Should now have chase generator in ACTIVE slot
	if !mm.HasMovementGeneratorType(MotionTypeCHASE) {
		t.Error("Should have CHASE generator after entering combat")
	}

	// 3. Simulate combat ticks
	for i := 0; i < 10; i++ {
		mm.Update(100)
	}

	// 4. Simulate evade
	ai.ExitCombat()

	// Should have home generator
	if !mm.HasMovementGeneratorType(MotionTypeHOME) {
		t.Error("Should have HOME generator after exiting combat")
	}

	// 5. Simulate return home
	for i := 0; i < 50; i++ {
		mm.Update(100)
	}
}

// TestThreatManager_InCombat tests threat management during combat
func TestThreatManager_InCombat(t *testing.T) {
	tm := NewThreatManager(nil)

	// Multiple players attack
	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player2", 150.0)
	tm.AddThreat("player3", 50.0)

	// Select highest threat
	victim := tm.SelectVictim(func(guid interface{}) bool {
		return true
	})

	if victim != "player2" {
		t.Errorf("Expected player2 (highest threat), got %v", victim)
	}

	// Player2 dies, remove from list
	tm.RemoveThreat("player2")

	// Now player1 should be highest
	victim = tm.SelectVictim(func(guid interface{}) bool {
		return guid != "player2" // player2 is dead
	})

	if victim != "player1" {
		t.Errorf("Expected player1 after player2 removed, got %v", victim)
	}
}

// TestWaypointGenerator_FullPath tests following a complete waypoint path
func TestWaypointGenerator_FullPath(t *testing.T) {
	owner := newMockOwner()
	owner.x = 100
	owner.y = 200
	owner.z = 30

	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 110, Y: 200, Z: 30, Delay: 0}, // no delay for test
			{Point: 2, X: 110, Y: 210, Z: 30, Delay: 0},
			{Point: 3, X: 100, Y: 210, Z: 30, Delay: 0},
			{Point: 4, X: 100, Y: 200, Z: 30, Delay: 0}, // back to start
		},
	}

	gen := NewWaypointGenerator(1, path, false) // not repeating
	gen.Initialize(nil)

	// Follow the path - each update advances one waypoint (no delays)
	waypointsVisited := 0
	for i := 0; i < 10; i++ {
		result := gen.Update(owner, 100)
		if !result {
			break // path complete
		}
		waypointsVisited++
	}

	// Should have visited all 4 waypoints
	if waypointsVisited != 4 {
		t.Errorf("Expected to visit all 4 waypoints, visited %d", waypointsVisited)
	}

	// Path should be complete
	result := gen.Update(owner, 100)
	if result {
		t.Error("Path should be complete after visiting all waypoints")
	}
}

// TestRandomGenerator_MovementPattern tests that random movement
// stays within wander distance of spawn
func TestRandomGenerator_MovementPattern(t *testing.T) {
	owner := newMockOwner()
	owner.spawnX = 100
	owner.spawnY = 200
	owner.spawnZ = 30
	owner.wanderDist = 5.0

	gen := NewRandomGenerator(5.0)
	gen.Initialize(nil)

	// Simulate many updates
	for i := 0; i < 1000; i++ {
		gen.Update(owner, 100)
	}

	// The mock doesn't actually move, but we can verify the generator
	// is still active and not crashing
	if gen.Type() != MotionTypeRANDOM {
		t.Error("Generator should still be RANDOM type")
	}
}
