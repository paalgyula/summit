package world

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

func TestGetClosestGraveyard(t *testing.T) {
	// Near Northshire in Elwynn Forest (Map 0, -8949, -132, 83)
	gy := GetClosestGraveyard(0, -8900.0, -100.0, 80.0)

	if gy.MapID != 0 {
		t.Fatalf("expected map 0, got %d", gy.MapID)
	}
	if gy.X != -8949.95 || gy.Y != -132.66 {
		t.Fatalf("expected Northshire graveyard coordinates, got X:%f Y:%f", gy.X, gy.Y)
	}

	// Durotar (Map 1)
	gyDurotar := GetClosestGraveyard(1, -600.0, -4200.0, 35.0)
	if gyDurotar.MapID != 1 {
		t.Fatalf("expected map 1, got %d", gyDurotar.MapID)
	}
	if gyDurotar.X != -618.5 {
		t.Fatalf("expected Durotar graveyard, got X:%f", gyDurotar.X)
	}
}

func TestPlayerResurrect(t *testing.T) {
	p := player.NewPlayer()
	p.MaxHealth = 200
	p.Health = 0
	p.IsGhost = true
	p.CharFlags |= 0x10
	p.PlayerFlags |= wow.PlayerFlagsGhost
	p.Power[wow.PowerTypeMana] = 0
	p.MaxPower[wow.PowerTypeMana] = 300

	p.Resurrect(0.5, 0.5)

	if p.IsGhost {
		t.Fatal("expected player to no longer be a ghost")
	}
	if p.Health != 100 {
		t.Fatalf("expected 50%% health (100), got %d", p.Health)
	}
	if p.Power[wow.PowerTypeMana] != 150 {
		t.Fatalf("expected 50%% mana (150), got %d", p.Power[wow.PowerTypeMana])
	}
	if p.CharFlags&0x10 != 0 {
		t.Fatal("expected dead flag 0x10 to be cleared")
	}
}
