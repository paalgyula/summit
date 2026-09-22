package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

// AddSpellCooldown adds a cooldown for a spell on the player. It is keyed by
// the spell id and, when the spell has a recovery category, by that category
// too so sibling spells share it. An existing, longer cooldown is kept (the
// global cooldown must not shorten a real one).
func AddSpellCooldown(p *player.Player, spellId, category uint32, end time.Time) {
	if p.SpellCooldowns == nil {
		p.SpellCooldowns = make(map[uint32]time.Time)
	}

	if cur, ok := p.SpellCooldowns[spellId]; !ok || end.After(cur) {
		p.SpellCooldowns[spellId] = end
	}
	if category > 0 {
		if cur, ok := p.SpellCooldowns[category]; !ok || end.After(cur) {
			p.SpellCooldowns[category] = end
		}
	}
}

// IsSpellOnCooldown checks the spell's own cooldown and its recovery category.
func IsSpellOnCooldown(p *player.Player, spellId, category uint32) bool {
	if p.SpellCooldowns == nil {
		return false
	}

	now := time.Now()
	if end, ok := p.SpellCooldowns[spellId]; ok && now.Before(end) {
		return true
	}

	if category > 0 {
		if end, ok := p.SpellCooldowns[category]; ok && now.Before(end) {
			return true
		}
	}

	return false
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
