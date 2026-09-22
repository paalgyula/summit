package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
)

// Update advances the game object's loot/respawn state machine by one tick.
// Mirrors AzerothCore GameObject::Update (GameObject.cpp:438).
func (g *GameObject) Update(diff uint32, ctx GameObjectUseContext) {
	now := time.Now()

	// Delayed despawn (summoned/owned GOs with a lifetime).
	if !g.DespawnAt.IsZero() && !now.Before(g.DespawnAt) {
		g.DespawnAt = time.Time{}
		g.SetLootState(GO_JUST_DEACTIVATED)
	}

	switch g.LootState {
	case GO_NOT_READY:
		g.updateNotReady(now, ctx)
	case GO_READY:
		g.updateReady(now, ctx)
	case GO_ACTIVATED:
		g.updateActivated(now, ctx)
	case GO_JUST_DEACTIVATED:
		g.updateJustDeactivated(now, ctx)
	}
}

// updateNotReady arms traps and restocks chests; everything else becomes ready.
func (g *GameObject) updateNotReady(now time.Time, _ GameObjectUseContext) {
	switch g.Type {
	case GameObjectTypeTrap:
		// Arming time for traps (bombs use a hard-coded 10s tooltip value).
		trapType := g.Template.Data[4]
		if trapType == 2 {
			g.CooldownAt = now.Add(10 * time.Second)
		} else if g.OwnerGUID != 0 {
			delay := time.Duration(g.Template.Data[7]) * time.Millisecond
			g.CooldownAt = now.Add(delay)
		}

		g.SetLootState(GO_READY)
	case GameObjectTypeChest:
		if !g.RestockAt.IsZero() && now.Before(g.RestockAt) {
			return
		}

		g.RestockAt = time.Time{}
		g.SetLootState(GO_READY)
	default:
		// Wait out a pending respawn before the GO becomes usable again.
		if !g.RespawnAt.IsZero() && now.Before(g.RespawnAt) {
			return
		}

		g.RespawnAt = time.Time{}
		g.SetLootState(GO_READY)
	}
}

// updateReady handles the respawn timer of deactivated GOs.
func (g *GameObject) updateReady(now time.Time, _ GameObjectUseContext) {
	if g.RespawnAt.IsZero() || now.Before(g.RespawnAt) {
		return
	}

	g.Respawn()
}

// updateActivated resolves timed activations (door/button auto-close, goober
// reset, trap cooldown).
func (g *GameObject) updateActivated(now time.Time, _ GameObjectUseContext) {
	switch g.Type {
	case GameObjectTypeDoor, GameObjectTypeButton:
		if g.autoCloseDuration() > 0 && !now.Before(g.CooldownAt) {
			g.ResetDoorOrButton()
		}
	case GameObjectTypeGoober:
		if !now.Before(g.CooldownAt) {
			g.RemoveGameObjectFlag(basedata.GOFlagInUse)
			g.SetLootState(GO_JUST_DEACTIVATED)
		}
	case GameObjectTypeTrap:
		if !now.Before(g.CooldownAt) {
			g.SetLootState(GO_READY)
		}
	}
}

// updateJustDeactivated either restocks, resets or (for summoned GOs) despawns.
func (g *GameObject) updateJustDeactivated(_ time.Time, _ GameObjectUseContext) {
	// Summoned/owned GOs are deleted rather than reset.
	if !g.SpawnedByDefault || g.SpellID != 0 || g.OwnerGUID != 0 {
		g.Loot = nil
		g.LootRecipient = 0

		return
	}

	switch g.Type {
	case GameObjectTypeChest:
		if !g.Template.IsDespawnAtAction() && g.Template.GetChestRestockTime() > 0 {
			g.RestockAt = time.Now().Add(time.Duration(g.Template.GetChestRestockTime()) * time.Second)
			g.SetLootState(GO_NOT_READY)

			return
		}
	case GameObjectTypeGoober:
		if !g.Template.IsDespawnAtAction() {
			g.SetGoState(GOStateReady)
			g.SetLootState(GO_READY)

			return
		}
	}

	// Consumed / despawn-at-action GO: wait out the respawn delay.
	g.RespawnAt = time.Now().Add(g.GetRespawnDelay())
	g.SetLootState(GO_NOT_READY)
}

// ResetDoorOrButton returns a door/button to its default state and flags.
// Mirrors GameObject::ResetDoorOrButton.
func (g *GameObject) ResetDoorOrButton() {
	g.RemoveGameObjectFlag(basedata.GOFlagInUse)
	g.SetGoState(GOStateReady)
	g.SetLootState(GO_READY)
}

// Respawn resets the GO to its spawned state after the respawn delay elapsed.
// Mirrors GameObject::Respawn.
func (g *GameObject) Respawn() {
	g.RespawnAt = time.Time{}
	g.RestockAt = time.Time{}
	g.CooldownAt = time.Time{}
	g.UseCount = 0
	g.Loot = nil
	g.LootRecipient = 0
	g.RemoveGameObjectFlag(basedata.GOFlagInUse)

	state := GOStateReady
	if g.Type == GameObjectTypeDoor && g.Template != nil && g.Template.GetDoorStartOpen() != 0 {
		state = GOState(0) // active / open
	}

	g.SetGoState(state)
	g.SetLootState(GO_READY)
}

// Update advances every spawned game object by one tick.
func (gm *GameObjectManager) Update(diff uint32, ctx GameObjectUseContext) {
	for _, gobj := range gm.spawns {
		gobj.Update(diff, ctx)
	}
}
