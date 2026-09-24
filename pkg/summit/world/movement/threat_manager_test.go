package movement

import (
	"testing"
)

func TestThreatManager_AddThreat(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player2", 50.0)

	if tm.GetThreat("player1") != 100.0 {
		t.Errorf("Expected threat 100.0, got %v", tm.GetThreat("player1"))
	}

	if tm.GetThreat("player2") != 50.0 {
		t.Errorf("Expected threat 50.0, got %v", tm.GetThreat("player2"))
	}
}

func TestThreatManager_AddThreat_Accumulates(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player1", 50.0)

	if tm.GetThreat("player1") != 150.0 {
		t.Errorf("Expected threat 150.0, got %v", tm.GetThreat("player1"))
	}
}

func TestThreatManager_ModifyThreat(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.ModifyThreat("player1", -30.0)

	if tm.GetThreat("player1") != 70.0 {
		t.Errorf("Expected threat 70.0, got %v", tm.GetThreat("player1"))
	}
}

func TestThreatManager_ModifyThreat_NegativeFloor(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 50.0)
	tm.ModifyThreat("player1", -100.0)

	if tm.GetThreat("player1") != 0 {
		t.Errorf("Expected threat 0 (floor), got %v", tm.GetThreat("player1"))
	}
}

func TestThreatManager_RemoveThreat(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.RemoveThreat("player1")

	if tm.GetThreat("player1") != 0 {
		t.Errorf("Expected threat 0 after remove, got %v", tm.GetThreat("player1"))
	}
}

func TestThreatManager_ClearThreat(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player2", 50.0)
	tm.ClearThreat()

	if tm.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", tm.Size())
	}
}

func TestThreatManager_SelectVictim(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player2", 50.0)

	// All targets valid
	victim := tm.SelectVictim(func(guid interface{}) bool {
		return true
	})

	if victim != "player1" {
		t.Errorf("Expected player1 as victim, got %v", victim)
	}
}

func TestThreatManager_SelectVictim_Filtered(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player2", 50.0)

	// Only player2 valid
	victim := tm.SelectVictim(func(guid interface{}) bool {
		return guid == "player2"
	})

	if victim != "player2" {
		t.Errorf("Expected player2 as victim, got %v", victim)
	}
}

func TestThreatManager_SelectVictim_Empty(t *testing.T) {
	tm := NewThreatManager(nil)

	victim := tm.SelectVictim(func(guid interface{}) bool {
		return true
	})

	if victim != nil {
		t.Errorf("Expected nil victim from empty list, got %v", victim)
	}
}

func TestThreatManager_GetSetVictim(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.SetVictim("player1")

	if tm.GetVictim() != "player1" {
		t.Errorf("Expected player1, got %v", tm.GetVictim())
	}
}

func TestThreatManager_IsOnThreatList(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)

	if !tm.IsOnThreatList("player1") {
		t.Error("player1 should be on threat list")
	}

	if tm.IsOnThreatList("player2") {
		t.Error("player2 should not be on threat list")
	}
}

func TestThreatManager_GetThreatList(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player2", 50.0)

	list := tm.GetThreatList()

	if len(list) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(list))
	}

	// Verify it's a copy
	delete(list, "player1")
	if tm.GetThreat("player1") != 100.0 {
		t.Error("Modifying returned list should not affect original")
	}
}

func TestThreatManager_Size(t *testing.T) {
	tm := NewThreatManager(nil)

	if tm.Size() != 0 {
		t.Errorf("Expected size 0, got %d", tm.Size())
	}

	tm.AddThreat("player1", 100.0)
	tm.AddThreat("player2", 50.0)

	if tm.Size() != 2 {
		t.Errorf("Expected size 2, got %d", tm.Size())
	}
}

func TestThreatManager_AddThreat_NilGuid(t *testing.T) {
	tm := NewThreatManager(nil)

	// Should not panic
	tm.AddThreat(nil, 100.0)

	if tm.Size() != 0 {
		t.Error("Should not add nil GUID to threat list")
	}
}

func TestThreatManager_AddThreat_ZeroAmount(t *testing.T) {
	tm := NewThreatManager(nil)

	tm.AddThreat("player1", 0)

	if tm.Size() != 0 {
		t.Error("Should not add zero threat")
	}
}

func TestThreatManager_ConcurrentAccess(t *testing.T) {
	tm := NewThreatManager(nil)

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				tm.AddThreat("player", float64(id*100+j))
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have some threat
	if tm.GetThreat("player") <= 0 {
		t.Error("Expected positive threat after concurrent access")
	}
}
