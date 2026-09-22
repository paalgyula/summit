package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

// LootState tracks a game object's interaction lifecycle. Mirrors AzerothCore's
// LootState enum in GameObject.h.
type LootState uint8

const (
	// GO_NOT_READY: the GO is arming/restocking and cannot be used yet.
	GO_NOT_READY LootState = iota
	// GO_READY: the GO is ready to be used (or spawned and waiting).
	GO_READY
	// GO_ACTIVATED: the GO is in use (loot window open, trap triggered, ...).
	GO_ACTIVATED
	// GO_JUST_DEACTIVATED: the GO finished its action and will reset/despawn.
	GO_JUST_DEACTIVATED
)

// defaultGameObjectRespawn is used when a spawn has no explicit respawn delay.
const defaultGameObjectRespawn = 25 * time.Second

// SetLootState updates the loot state and stores the triggering unit, if any.
func (g *GameObject) SetLootState(state LootState) {
	g.LootState = state
}

// GetLootState returns the current loot state.
func (g *GameObject) GetLootState() LootState {
	return g.LootState
}

// SetGoState updates GAMEOBJECT_BYTES_1[0] and the cached state.
func (g *GameObject) SetGoState(state GOState) {
	g.GOState = state
	g.Object.SetByteValue(object.GameobjectBytes_1, 0, byte(state))
}

// SetGoArtKit updates GAMEOBJECT_BYTES_1[2] and the cached art kit.
func (g *GameObject) SetGoArtKit(kit uint8) {
	g.ArtKit = kit
	g.Object.SetByteValue(object.GameobjectBytes_1, 2, kit)
}

// SetGoAnimProgress updates GAMEOBJECT_BYTES_1[3].
func (g *GameObject) SetGoAnimProgress(progress uint8) {
	g.AnimProgress = uint32(progress)
	g.Object.SetByteValue(object.GameobjectBytes_1, 3, progress)
}

// SetGameObjectFlag sets bits in GAMEOBJECT_FLAGS.
func (g *GameObject) SetGameObjectFlag(flag basedata.GameObjectFlags) {
	g.Object.SetFlag(object.GameobjectFlags, uint32(flag))
}

// RemoveGameObjectFlag clears bits in GAMEOBJECT_FLAGS.
func (g *GameObject) RemoveGameObjectFlag(flag basedata.GameObjectFlags) {
	g.Object.RemoveFlag(object.GameobjectFlags, uint32(flag))
}

// ReplaceAllGameObjectFlags overwrites GAMEOBJECT_FLAGS.
func (g *GameObject) ReplaceAllGameObjectFlags(flag basedata.GameObjectFlags) {
	g.Flags = uint32(flag)
	g.Object.SetUInt32Value(object.GameobjectFlags, uint32(flag))
}

// HasGameObjectFlag reports whether all given bits are set in GAMEOBJECT_FLAGS.
func (g *GameObject) HasGameObjectFlag(flag basedata.GameObjectFlags) bool {
	return g.Object.HasFlag(object.GameobjectFlags, uint32(flag))
}

// SetGameObjectDynamicFlags sets the low half of GAMEOBJECT_DYNAMIC (the
// client-facing dynamic flags such as sparkle/activate). The high half carries
// path progress, which stays 0 for non-transports.
func (g *GameObject) SetGameObjectDynamicFlags(flags uint16) {
	current := uint32(g.Object.GetUInt32Value(object.GameobjectDynamic))
	g.Object.SetUInt32Value(object.GameobjectDynamic, (current&0xFFFF0000)|uint32(flags))
}

// GetGameObjectDynamicFlags returns the low half of GAMEOBJECT_DYNAMIC.
func (g *GameObject) GetGameObjectDynamicFlags() uint16 {
	return uint16(g.Object.GetUInt32Value(object.GameobjectDynamic))
}

// SetSpellID records the spell a summoned GO should cast (0 clears it).
func (g *GameObject) SetSpellID(id uint32) {
	g.SpawnedByDefault = false
	g.SpellID = id
}

// GetSpellID returns the spell associated with the GO, if any.
func (g *GameObject) GetSpellID() uint32 {
	return g.SpellID
}

// SetOwnerGUID sets the GO's owner (summoner), making it despawn after use.
func (g *GameObject) SetOwnerGUID(owner wow.GUID) {
	g.SpawnedByDefault = false
	g.OwnerGUID = owner
}

// GetOwnerGUID returns the GO's owner GUID (0 when unowned).
func (g *GameObject) GetOwnerGUID() wow.GUID {
	return g.OwnerGUID
}

// SetLinkedTrap records the linked trap this GO triggers.
func (g *GameObject) SetLinkedTrap(trap *GameObject) {
	g.LinkedTrap = trap.GetGUID()
}

// SetRespawnTime schedules the next respawn. A zero time means "no timer".
func (g *GameObject) SetRespawnTime(at time.Time) {
	g.RespawnAt = at
}

// SetRespawnDelay overrides the respawn delay.
func (g *GameObject) SetRespawnDelay(delay time.Duration) {
	g.RespawnDelay = delay
}

// GetRespawnDelay returns the configured respawn delay, or the default.
func (g *GameObject) GetRespawnDelay() time.Duration {
	if g.RespawnDelay <= 0 {
		return defaultGameObjectRespawn
	}

	return g.RespawnDelay
}

// IsSpawned reports whether the GO is currently present in the world.
// Mirrors GameObject::isSpawned.
func (g *GameObject) IsSpawned() bool {
	if g.RespawnDelay <= 0 {
		return true
	}

	if g.RespawnAt.IsZero() {
		return g.SpawnedByDefault
	}

	return !g.SpawnedByDefault
}

// IsLooted reports whether the GO's loot has been fully taken/deactivated.
func (g *GameObject) IsLooted() bool {
	return g.LootState == GO_JUST_DEACTIVATED
}
