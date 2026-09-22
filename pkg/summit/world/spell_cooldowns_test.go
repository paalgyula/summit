package world

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

// Category cooldowns are stored under the recovery category; the lookup has to
// check it as well as the spell id, otherwise the hearthstone (30 minute
// category cooldown) can be used again immediately.
func TestCategoryCooldownBlocksSpellAndSiblings(t *testing.T) {
	p := player.NewPlayer()
	AddSpellCooldown(p, 8690, 133, time.Now().Add(30*time.Minute))

	if !IsSpellOnCooldown(p, 8690, 133) {
		t.Fatal("the hearthstone should be on cooldown")
	}

	// Another spell in the same recovery category shares the cooldown.
	if !IsSpellOnCooldown(p, 99999, 133) {
		t.Fatal("a spell sharing the category should be on cooldown")
	}

	// An unrelated spell is unaffected.
	if IsSpellOnCooldown(p, 12345, 0) {
		t.Fatal("an unrelated spell should not be on cooldown")
	}
}

func TestExpiredCooldownIsNotBlocking(t *testing.T) {
	p := player.NewPlayer()
	AddSpellCooldown(p, 8690, 133, time.Now().Add(-time.Second))

	if IsSpellOnCooldown(p, 8690, 133) {
		t.Fatal("an expired cooldown should not block")
	}
}
