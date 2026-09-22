package bot

import (
	"testing"

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
	if cfg.MeleeRange <= 0 || cfg.MoveSpeed <= 0 {
		t.Fatal("default config must have positive movement values")
	}
}
