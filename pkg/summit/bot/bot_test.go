package bot

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/summit/client"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

func TestNameOf(t *testing.T) {
	b := &Bot{}

	if got := b.nameOf(&client.Entity{Name: "Training Dummy"}); got != "Training Dummy" {
		t.Fatalf("expected the resolved name, got %q", got)
	}

	// OBJECT_FIELD_ENTRY is update-field word 3.
	if got := b.nameOf(&client.Entity{Fields: []uint32{0, 0, 0, 100001}}); got != "entry#100001" {
		t.Fatalf("expected entry fallback, got %q", got)
	}
}

func TestDistance(t *testing.T) {
	a := player.WorldLocation{X: 0, Y: 0}
	b := player.WorldLocation{X: 3, Y: 4}

	if got := distance(a, b); got != 5 {
		t.Fatalf("expected 5, got %v", got)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MeleeRange <= 0 || cfg.MoveSpeed <= 0 || cfg.LeashRadius <= 0 {
		t.Fatal("default config must have positive movement values")
	}
}

func TestDeathSequence(t *testing.T) {
	var d deathState
	now := time.Now()

	if got := d.step(now, true); got != deathRepop {
		t.Fatalf("first dead tick should repop, got %v", got)
	}

	if got := d.step(now.Add(time.Second), true); got != deathNone {
		t.Fatalf("should wait before reclaiming, got %v", got)
	}

	if got := d.step(now.Add(3*time.Second), true); got != deathReclaim {
		t.Fatalf("after 2s should reclaim, got %v", got)
	}

	if got := d.step(now.Add(5*time.Second), true); got != deathNone {
		t.Fatalf("reclaim sent once, got %v", got)
	}

	if got := d.step(now.Add(9*time.Second), true); got != deathHealer {
		t.Fatalf("after 8s should use the healer, got %v", got)
	}

	if got := d.step(now.Add(20*time.Second), false); got != deathNone || d.active {
		t.Fatalf("being alive should reset the sequence (got %v, active=%v)", got, d.active)
	}

	if got := d.step(now.Add(21*time.Second), true); got != deathRepop {
		t.Fatalf("a new death should restart the sequence, got %v", got)
	}
}

func TestSpellsToUse(t *testing.T) {
	b := &Bot{cfg: Config{Spells: []uint32{133, 116}}}

	got := b.spellsToUse()
	if len(got) != 2 || got[0] != 133 || got[1] != 116 {
		t.Fatalf("expected configured spells, got %v", got)
	}

	// With no configured spells and auto-cast disabled the bot casts nothing
	// (and must not touch the client).
	off := &Bot{cfg: Config{AutoCast: false}}
	if off.spellsToUse() != nil {
		t.Fatal("expected no spells when auto-cast is off")
	}
}
