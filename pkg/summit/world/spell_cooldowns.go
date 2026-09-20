package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

// AddSpellCooldown adds a cooldown for a spell on the player.
// If category > 0, the cooldown is shared with other spells in the same category.
func AddSpellCooldown(p *player.Player, spellId, category uint32, end time.Time) {
	if p.SpellCooldowns == nil {
		p.SpellCooldowns = make(map[uint32]time.Time)
	}

	key := spellId
	if category > 0 {
		key = category // category cooldowns share the same key
	}

	p.SpellCooldowns[key] = end
}

// IsSpellOnCooldown checks if a spell is currently on cooldown.
func IsSpellOnCooldown(p *player.Player, spellId uint32) bool {
	if p.SpellCooldowns == nil {
		return false
	}

	end, ok := p.SpellCooldowns[spellId]
	if !ok {
		return false
	}

	return time.Now().Before(end)
}

// GetSpellCooldownDelay returns the remaining cooldown duration.
func GetSpellCooldownDelay(p *player.Player, spellId uint32) time.Duration {
	if p.SpellCooldowns == nil {
		return 0
	}

	end, ok := p.SpellCooldowns[spellId]
	if !ok {
		return 0
	}

	remaining := time.Until(end)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// RemoveSpellCooldown removes a spell's cooldown.
func RemoveSpellCooldown(p *player.Player, spellId uint32) {
	if p.SpellCooldowns == nil {
		return
	}

	delete(p.SpellCooldowns, spellId)
}

// ClearAllCooldowns removes all cooldowns from the player.
func ClearAllCooldowns(p *player.Player) {
	p.SpellCooldowns = make(map[uint32]time.Time)
}
