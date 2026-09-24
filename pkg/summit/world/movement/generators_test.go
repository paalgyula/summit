package movement

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/store"
)

func TestRandomGenerator_Type(t *testing.T) {
	gen := NewRandomGenerator(5.0)
	if gen.Type() != MotionTypeRANDOM {
		t.Errorf("Expected RANDOM type, got %v", gen.Type())
	}
}

func TestRandomGenerator_Initialize(t *testing.T) {
	gen := NewRandomGenerator(5.0)
	gen.Initialize(nil)

	if gen.nextMoveTime != 0 {
		t.Error("nextMoveTime should be 0 after Initialize")
	}

	if gen.moveCount != 0 {
		t.Error("moveCount should be 0 after Initialize")
	}
}

func TestRandomGenerator_Update_SetsTimer(t *testing.T) {
	owner := newMockOwner()
	gen := NewRandomGenerator(5.0)
	gen.Initialize(nil)

	// First update should set the timer
	result := gen.Update(owner, 100)
	if !result {
		t.Error("Update should return true")
	}

	if gen.nextMoveTime == 0 {
		t.Error("nextMoveTime should be set after first update")
	}
}

func TestRandomGenerator_Type_String(t *testing.T) {
	gen := NewRandomGenerator(5.0)
	if gen.Type().String() != "Random" {
		t.Errorf("Expected 'Random', got %v", gen.Type().String())
	}
}

func TestWaypointGenerator_Type(t *testing.T) {
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30},
			{Point: 2, X: 110, Y: 210, Z: 30},
		},
	}
	gen := NewWaypointGenerator(1, path, true)
	if gen.Type() != MotionTypeWAYPOINT {
		t.Errorf("Expected WAYPOINT type, got %v", gen.Type())
	}
}

func TestWaypointGenerator_Initialize(t *testing.T) {
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30},
		},
	}
	gen := NewWaypointGenerator(1, path, true)
	gen.Initialize(nil)

	if gen.waypointIndex != 0 {
		t.Error("waypointIndex should be 0 after Initialize")
	}

	if gen.delayEnd != 0 {
		t.Error("delayEnd should be 0 after Initialize")
	}
}

func TestWaypointGenerator_Update_EmptyPath(t *testing.T) {
	owner := newMockOwner()
	gen := NewWaypointGenerator(1, nil, true)
	gen.Initialize(nil)

	// Should return false with nil path
	result := gen.Update(owner, 100)
	if result {
		t.Error("Update should return false with nil path")
	}
}

func TestWaypointGenerator_Update_FollowsPath(t *testing.T) {
	owner := newMockOwner()
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30, Delay: 1000},
			{Point: 2, X: 110, Y: 210, Z: 30},
		},
	}
	gen := NewWaypointGenerator(1, path, true)
	gen.Initialize(nil)

	// First update should move to first waypoint
	result := gen.Update(owner, 100)
	if !result {
		t.Error("Update should return true")
	}

	// Should have advanced to next waypoint
	if gen.waypointIndex != 1 {
		t.Errorf("waypointIndex should be 1, got %d", gen.waypointIndex)
	}

	// Should have set delay
	if gen.delayEnd == 0 {
		t.Error("delayEnd should be set")
	}
}

func TestWaypointGenerator_Update_WaitsForDelay(t *testing.T) {
	owner := newMockOwner()
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30, Delay: 5000},
		},
	}
	gen := NewWaypointGenerator(1, path, true)
	gen.Initialize(nil)

	// First update
	gen.Update(owner, 100)

	// Set delay to far future
	gen.delayEnd = time.Now().UnixMilli() + 10000

	// Second update should wait
	result := gen.Update(owner, 100)
	if !result {
		t.Error("Update should return true while waiting")
	}

	// Waypoint index should not have advanced
	if gen.waypointIndex != 1 {
		t.Errorf("waypointIndex should still be 1, got %d", gen.waypointIndex)
	}
}

func TestWaypointGenerator_Update_CompletePath(t *testing.T) {
	owner := newMockOwner()
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30},
		},
	}
	gen := NewWaypointGenerator(1, path, false) // not repeating
	gen.Initialize(nil)

	// First update moves to waypoint
	gen.Update(owner, 100)

	// Second update should complete
	result := gen.Update(owner, 100)
	if result {
		t.Error("Update should return false when path complete")
	}
}

func TestWaypointGenerator_Update_RepeatingPath(t *testing.T) {
	owner := newMockOwner()
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30},
		},
	}
	gen := NewWaypointGenerator(1, path, true) // repeating
	gen.Initialize(nil)

	// First update moves to waypoint
	gen.Update(owner, 100)

	// Second update should wrap around
	result := gen.Update(owner, 100)
	if !result {
		t.Error("Update should return true for repeating path")
	}

	// Should have wrapped around
	if gen.waypointIndex != 1 {
		t.Errorf("waypointIndex should be 1 after wrap, got %d", gen.waypointIndex)
	}
}

func TestWaypointGenerator_GetCurrentNode(t *testing.T) {
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30},
			{Point: 2, X: 110, Y: 210, Z: 30},
		},
	}
	gen := NewWaypointGenerator(1, path, true)

	if gen.GetCurrentNode() != 0 {
		t.Error("GetCurrentNode should return 0 initially")
	}
}

func TestWaypointGenerator_GetPathID(t *testing.T) {
	path := &store.WaypointPath{
		PathID: 42,
		Points: []store.Waypoint{},
	}
	gen := NewWaypointGenerator(42, path, true)

	if gen.GetPathID() != 42 {
		t.Errorf("GetPathID should return 42, got %d", gen.GetPathID())
	}
}

func TestWaypointGenerator_Reset(t *testing.T) {
	path := &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 100, Y: 200, Z: 30},
		},
	}
	gen := NewWaypointGenerator(1, path, true)
	gen.Initialize(nil)

	// Move forward
	gen.Update(newMockOwner(), 100)

	// Reset
	gen.Reset(nil)

	if gen.waypointIndex != 0 {
		t.Error("waypointIndex should be 0 after Reset")
	}

	if gen.delayEnd != 0 {
		t.Error("delayEnd should be 0 after Reset")
	}
}
