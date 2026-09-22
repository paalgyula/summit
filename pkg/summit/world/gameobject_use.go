package world

import (
	"errors"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/rs/zerolog/log"
)

// Game object interaction errors.
var (
	// ErrGameObjectLocked is returned when the GO requires a key/spell/event.
	ErrGameObjectLocked = errors.New("game object is locked")
	// ErrGameObjectNotInteractable is returned for GO_FLAG_NOT_SELECTABLE GOs.
	ErrGameObjectNotInteractable = errors.New("game object is not interactable")
	// ErrGameObjectNotReady is returned while the GO is arming or restocking.
	ErrGameObjectNotReady = errors.New("game object is not ready")
	// ErrGameObjectCooldown is returned while the GO cooldown is active.
	ErrGameObjectCooldown = errors.New("game object is on cooldown")
)

// GameObjectUseContext bundles the references a use/update handler needs. The
// Session and Player may be nil when the GO is triggered without a player (for
// example by a linked trap or an AI event).
type GameObjectUseContext struct {
	Server  *Server
	Session *WorldSession
	Player  *player.Player
}

// Use processes a player (or script) using the game object and dispatches to
// the per-type behavior. Mirrors AzerothCore's GameObject::Use.
func (g *GameObject) Use(ctx GameObjectUseContext) error {
	if g.HasGameObjectFlag(basedata.GOFlagNotSelectable) {
		return ErrGameObjectNotInteractable
	}

	if g.Template != nil && g.Template.GetCooldown() > 0 && time.Now().Before(g.CooldownAt) {
		return ErrGameObjectCooldown
	}

	switch g.Type {
	case GameObjectTypeChest:
		return g.useChest(ctx)
	case GameObjectTypeDoor, GameObjectTypeButton:
		return g.useDoorOrButton(ctx, false)
	case GameObjectTypeTrap:
		return g.useTrap(ctx)
	case GameObjectTypeGoober:
		return g.useGoober(ctx)
	case GameObjectTypeSpellcaster:
		return g.useSpellCaster(ctx)
	default:
		return nil
	}
}

// useChest opens a chest and, when a session is present, sends the loot window.
func (g *GameObject) useChest(ctx GameObjectUseContext) error {
	if g.HasGameObjectFlag(basedata.GOFlagLocked) {
		return ErrGameObjectLocked
	}

	switch g.LootState {
	case GO_READY:
		if ctx.Session != nil {
			ctx.Session.openGameObjectLoot(g)
		}

		g.SetLootState(GO_ACTIVATED)
	case GO_NOT_READY:
		return ErrGameObjectNotReady
	default:
		// Already activated / deactivated: nothing to do.
	}

	return nil
}

// useDoorOrButton toggles a door/button and arms its auto-close timer.
// Mirrors GameObject::UseDoorOrButton.
func (g *GameObject) useDoorOrButton(ctx GameObjectUseContext, alternative bool) error {
	if g.HasGameObjectFlag(basedata.GOFlagLocked) {
		return ErrGameObjectLocked
	}

	switch g.GOState {
	case GOStateReady:
		g.SetGoState(GOStateActive)
	default:
		g.SetGoState(GOStateReady)
	}

	g.UseCount++
	g.CooldownAt = time.Now().Add(g.autoCloseDuration())
	g.SetLootState(GO_ACTIVATED)

	g.triggerLinkedTrap(ctx)

	return nil
}

// useTrap casts the trap's spell at the user and arms its cooldown.
func (g *GameObject) useTrap(ctx GameObjectUseContext) error {
	if spellID := g.Template.GetSpellID(); spellID != 0 {
		g.castSpell(ctx, spellID)
	}

	cooldown := time.Duration(g.Template.GetCooldown()) * time.Millisecond
	if cooldown <= 0 {
		cooldown = 4 * time.Second
	}

	g.CooldownAt = time.Now().Add(cooldown)
	g.UseCount++

	// Trap type 1 deactivates (despawns) after triggering.
	if g.Type == GameObjectTypeTrap && g.Template.Data[4] == 1 {
		g.SetLootState(GO_JUST_DEACTIVATED)
	}

	return nil
}

// useGoober handles the "big gun" clickable object: page text, gossip, event,
// quest credit, linked trap, custom animation and spell.
func (g *GameObject) useGoober(ctx GameObjectUseContext) error {
	if g.HasGameObjectFlag(basedata.GOFlagInUse) {
		return nil
	}

	if pageID := g.Template.GetGooberPageText(); pageID != 0 && ctx.Session != nil {
		ctx.Session.sendGameObjectPageText(g, pageID)
	}

	// TODO(quests): gossip menu, quest objective gating and KillCreditGO.

	if spellID := g.Template.GetSpellID(); spellID != 0 {
		g.castSpell(ctx, spellID)
	}

	g.triggerLinkedTrap(ctx)

	if autoClose := g.Template.GetAutoCloseTime(); autoClose > 0 {
		g.SetGameObjectFlag(basedata.GOFlagInUse)
		g.SetLootState(GO_ACTIVATED)

		if g.Template.GetGooberCustomAnim() == 0 {
			g.SetGoState(GOStateActive)
		}
	}

	g.CooldownAt = time.Now().Add(time.Duration(g.Template.GetCooldown()) * time.Millisecond)

	return nil
}

// useSpellCaster casts the configured spell when the GO is used.
func (g *GameObject) useSpellCaster(ctx GameObjectUseContext) error {
	if spellID := g.Template.GetSpellID(); spellID != 0 {
		g.castSpell(ctx, spellID)
		g.UseCount++
	}

	return nil
}

// castSpell is the integration point for the spell system. For now it only
// logs; wiring to the spell cast engine is tracked as a follow-up.
func (g *GameObject) castSpell(_ GameObjectUseContext, spellID uint32) {
	log.Debug().
		Uint32("goEntry", g.Entry).
		Uint32("spell", spellID).
		Msg("game object casts spell")
}

// triggerLinkedTrap activates a linked trap game object, if one is configured.
func (g *GameObject) triggerLinkedTrap(ctx GameObjectUseContext) {
	entry := g.Template.GetLinkedGameObjectEntry()
	if entry == 0 || ctx.Server == nil || ctx.Server.gameObjects == nil {
		return
	}

	trap := ctx.Server.gameObjects.GetByEntry(entry)
	if trap == nil || trap == g {
		return
	}

	g.SetLinkedTrap(trap)

	if err := trap.Use(ctx); err != nil {
		log.Debug().Err(err).Uint32("trap", entry).Msg("linked trap use failed")
	}
}

// autoCloseDuration returns the template auto-close time as a duration.
func (g *GameObject) autoCloseDuration() time.Duration {
	if g.Template == nil {
		return 0
	}

	return time.Duration(g.Template.GetAutoCloseTime()) * time.Millisecond
}
