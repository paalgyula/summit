// Package bot implements a playable AI client for the Summit world server.
//
// The bot enters the world through pkg/summit/client, perceives nearby
// creatures from SMSG_UPDATE_OBJECT, walks to a hostile target, auto-attacks
// and casts spells, detects the kill (the server does not broadcast NPC health,
// so it is inferred from the bot's own melee hits and reported health updates),
// loots the corpse, resurrects after death and repeats.
package bot

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/paalgyula/summit/pkg/summit/client"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// moveFlagForward is MOVEMENTFLAG_FORWARD.
const moveFlagForward = 0x00000001

// spellFailureBackoff is how long a rejected spell is skipped for.
const spellFailureBackoff = 10 * time.Second

// Config tunes the bot's behaviour.
type Config struct {
	// Tick is the AI decision interval.
	Tick time.Duration
	// EngageRange is the furthest distance at which a target is picked up.
	EngageRange float32
	// MeleeRange is the distance at which the bot stops and attacks.
	MeleeRange float32
	// MoveSpeed is the yards/second the bot walks toward a target.
	MoveSpeed float32
	// WanderRadius is how far the bot roams while looking for a target.
	WanderRadius float32
	// LeashRadius is how far from its home point the bot will chase.
	LeashRadius float32
	// Loot controls whether corpses are looted after a kill.
	Loot bool
	// GroundFollow blends the bot's Z toward the target's height while moving.
	GroundFollow bool

	// Spells are cast at the current target. When empty and AutoCast is set,
	// every known spell is tried and the rejected ones are backed off.
	Spells []uint32
	// AutoCast enables casting the character's known spells when Spells is empty.
	AutoCast bool
	// SpellRange is the maximum distance at which spells are cast.
	SpellRange float32
	// GCD is the minimum interval between two casts.
	GCD time.Duration
}

// DefaultConfig returns sensible defaults for a melee/caster grinding bot.
func DefaultConfig() Config {
	return Config{
		Tick:         250 * time.Millisecond,
		EngageRange:  60,
		MeleeRange:   4,
		MoveSpeed:    7,
		WanderRadius: 12,
		LeashRadius:  80,
		Loot:         true,
		GroundFollow: true,
		AutoCast:     true,
		SpellRange:   25,
		GCD:          1500 * time.Millisecond,
	}
}

// Bot drives a WorldClient through a simple grinding loop.
type Bot struct {
	client *client.WorldClient
	cfg    Config
	log    zerolog.Logger

	home   player.WorldLocation
	pos    player.WorldLocation
	hasPos bool

	target   wow.GUID
	swinging bool
	moving   bool
	kills    int

	lastCast time.Time

	pendingLoot wow.GUID
	lootTimer   time.Time

	wanderTarget *player.WorldLocation
	lastWander   time.Time

	// Death / resurrection state machine.
	death deathState
}

// deathAction is what the bot should do about being dead.
type deathAction int

const (
	deathNone deathAction = iota
	deathRepop
	deathReclaim
	deathHealer
)

// deathState is the pure timing behind the death recovery sequence.
type deathState struct {
	active      bool
	startedAt   time.Time
	reclaimSent bool
	healerUsed  bool
}

// step advances the death sequence and returns the next action to take.
func (d *deathState) step(now time.Time, dead bool) deathAction {
	if !dead {
		d.reset()
		return deathNone
	}

	if !d.active {
		d.active = true
		d.startedAt = now

		return deathRepop
	}

	elapsed := now.Sub(d.startedAt)

	if elapsed > 2*time.Second && !d.reclaimSent {
		d.reclaimSent = true

		return deathReclaim
	}

	if elapsed > 8*time.Second && !d.healerUsed {
		d.healerUsed = true

		return deathHealer
	}

	return deathNone
}

func (d *deathState) reset() { *d = deathState{} }

// New creates a bot for an already-in-world client.
func New(wc *client.WorldClient, cfg Config) *Bot {
	b := &Bot{
		client: wc,
		cfg:    cfg,
		log:    log.With().Str("service", "bot").Logger(),
	}

	if p := wc.Player(); p != nil {
		b.pos = p.Location
		b.home = p.Location
		b.hasPos = true
	}

	return b
}

// Kills returns how many creatures the bot has slain.
func (b *Bot) Kills() int { return b.kills }

// Run drives the AI until ctx is cancelled or the connection drops.
func (b *Bot) Run(ctx context.Context) error {
	ticker := time.NewTicker(b.cfg.Tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-b.client.Closed():
			return errors.New("world connection closed")
		case <-ticker.C:
			b.tick()
		}
	}
}

func (b *Bot) tick() {
	self := b.client.SelfEntity()

	if self != nil && self.HasPos {
		b.syncPosition(self)
	}

	// Death and resurrection take priority over everything else.
	if b.client.IsDead() {
		b.handleDeath()

		return
	}

	if b.death.active {
		b.onResurrect(self)
	}

	b.updateLoot()

	target := b.validTarget()

	if target == nil {
		if b.target != 0 {
			b.target = 0
			b.swinging = false
		}

		// Leash: walk back home when we have strayed too far.
		if distance(b.pos, b.home) > b.cfg.LeashRadius {
			b.moveToward(b.home)

			return
		}

		mob := b.client.NearestAttackable(&b.pos, b.cfg.EngageRange)
		if mob == nil || distance(mob.Pos, b.home) > b.cfg.LeashRadius {
			b.wander()

			return
		}

		b.engage(mob)

		return
	}

	b.fight(target)
}

// validTarget returns the current target while it is still alive, otherwise
// resolves the kill and returns nil.
func (b *Bot) validTarget() *client.Entity {
	if b.target == 0 {
		return nil
	}

	t := b.client.Entity(b.target)
	if t == nil {
		b.log.Debug().Msg("target left view")
		b.target = 0
		b.swinging = false

		return nil
	}

	if t.SlainByDamage() || !t.IsAlive() {
		b.onKill(t)
		return nil
	}

	return t
}

func (b *Bot) onKill(t *client.Entity) {
	b.kills++

	b.log.Info().
		Str("name", b.nameOf(t)).
		Uint32("damage", t.DamageDealtBySelf).
		Int("kills", b.kills).
		Msg("target slain")

	b.stopAttack()
	b.stopMoving()
	b.target = 0
	b.swinging = false

	if b.cfg.Loot {
		b.startLoot(t)
	}
}

func (b *Bot) engage(mob *client.Entity) {
	b.target = mob.GUID

	b.log.Info().
		Str("name", b.nameOf(mob)).
		Uint32("level", mob.Level()).
		Uint32("health", mob.MaxHealth()).
		Float32("distance", distance(b.pos, mob.Pos)).
		Msg("engaging target")

	b.client.SetSelection(mob.GUID)
	b.client.AttackSwing(mob.GUID)
	b.swinging = true

	if distance(b.pos, mob.Pos) <= b.cfg.MeleeRange {
		b.stopMoving()
	}
}

func (b *Bot) fight(t *client.Entity) {
	dist := distance(b.pos, t.Pos)

	// Keep the target inside the leash.
	if distance(t.Pos, b.home) > b.cfg.LeashRadius {
		b.log.Debug().Str("name", b.nameOf(t)).Msg("target beyond leash, disengaging")
		b.stopAttack()
		b.target = 0
		b.swinging = false

		return
	}

	if dist > b.cfg.MeleeRange {
		b.moveToward(t.Pos)
	} else {
		b.stopMoving()
	}

	if !b.swinging {
		b.client.AttackSwing(t.GUID)
		b.swinging = true
	}

	b.maybeCast(t, dist)
}

// maybeCast casts an offensive spell at the target when in range and off the
// global cooldown. Rejected spells are backed off.
func (b *Bot) maybeCast(t *client.Entity, dist float32) {
	spells := b.spellsToUse()
	if len(spells) == 0 || dist > b.cfg.SpellRange {
		return
	}

	if time.Since(b.lastCast) < b.cfg.GCD {
		return
	}

	for _, spellID := range spells {
		if failure, failed := b.client.SpellFailed(spellID); failed &&
			time.Since(failure.At) < spellFailureBackoff {
			continue
		}

		b.client.CastSpell(spellID, t.GUID)
		b.lastCast = time.Now()

		b.log.Debug().Uint32("spell", spellID).Str("target", b.nameOf(t)).Msg("casting spell")

		return
	}
}

func (b *Bot) spellsToUse() []uint32 {
	if len(b.cfg.Spells) > 0 {
		return b.cfg.Spells
	}

	if !b.cfg.AutoCast {
		return nil
	}

	return b.client.KnownSpells()
}

// handleDeath releases the spirit, reclaims the corpse and falls back to the
// spirit healer if the reclaim never completes.
func (b *Bot) handleDeath() {
	b.stopAttack()
	b.stopMoving()
	b.target = 0
	b.swinging = false

	switch b.death.step(time.Now(), true) {
	case deathRepop:
		b.log.Warn().Msg("died, releasing spirit")
		b.client.RepopRequest()
	case deathReclaim:
		b.log.Info().Msg("reclaiming corpse")
		b.client.ReclaimCorpse(0)
	case deathHealer:
		b.log.Info().Msg("corpse reclaim timed out, using spirit healer")
		b.client.SpiritHealerActivate()
	}
}

func (b *Bot) onResurrect(self *client.Entity) {
	b.log.Info().Msg("resurrected")

	b.death.reset()

	if self != nil && self.HasPos {
		b.pos = self.Pos
		b.home = self.Pos
		b.hasPos = true
	}
}

// syncPosition keeps the bot's local position in step with the server's view.
func (b *Bot) syncPosition(self *client.Entity) {
	if !b.hasPos {
		b.home = self.Pos
		b.hasPos = true
		b.log.Info().
			Uint32("map", self.Pos.Map).
			Float32("x", self.Pos.X).
			Float32("y", self.Pos.Y).
			Msg("bot spawned")
	}
}

func (b *Bot) moveToward(dest player.WorldLocation) {
	o := client.OrientationTo(b.pos, dest)
	step := b.cfg.MoveSpeed * float32(b.cfg.Tick.Seconds())

	dx := dest.X - b.pos.X
	dy := dest.Y - b.pos.Y
	dist := float32(math.Hypot(float64(dx), float64(dy)))

	if dist < 0.01 {
		return
	}

	if step > dist {
		step = dist
	}

	b.pos.X += float32(math.Cos(float64(o))) * step
	b.pos.Y += float32(math.Sin(float64(o))) * step
	b.pos.O = o

	if b.cfg.GroundFollow {
		// Blend the height toward the destination so the bot follows the ground
		// instead of hovering at its spawn height.
		frac := min(float32(1), step/max(dist, 1))
		b.pos.Z += (dest.Z - b.pos.Z) * frac
	}

	if b.moving {
		b.client.SendMovement(wow.MsgMoveHeartbeat, moveFlagForward, b.pos)
	} else {
		b.client.SendMovement(wow.MsgMoveStartForward, moveFlagForward, b.pos)
		b.moving = true
	}
}

func (b *Bot) stopMoving() {
	if b.moving {
		b.client.SendMovement(wow.MsgMoveStop, 0, b.pos)
		b.moving = false
	}
}

func (b *Bot) stopAttack() {
	if b.swinging {
		b.client.AttackStop()
		b.swinging = false
	}
}

// wander roams around the spawn point while no target is available.
func (b *Bot) wander() {
	if b.wanderTarget != nil {
		if distance(b.pos, *b.wanderTarget) > 1.5 {
			b.moveToward(*b.wanderTarget)
			return
		}

		b.stopMoving()
		b.wanderTarget = nil
	}

	now := time.Now()
	if now.Sub(b.lastWander) < 3*time.Second {
		b.stopMoving()
		return
	}

	b.lastWander = now

	angle := rand.Float64() * 2 * math.Pi
	radius := rand.Float64() * float64(b.cfg.WanderRadius)

	b.wanderTarget = &player.WorldLocation{
		Map: b.pos.Map,
		X:   b.home.X + float32(math.Cos(angle))*float32(radius),
		Y:   b.home.Y + float32(math.Sin(angle))*float32(radius),
		Z:   b.pos.Z,
	}
}

func (b *Bot) startLoot(t *client.Entity) {
	b.log.Info().Str("name", b.nameOf(t)).Msg("looting corpse")
	b.client.Loot(t.GUID)
	b.pendingLoot = t.GUID
	b.lootTimer = time.Now().Add(time.Second)
}

// updateLoot releases an open loot window after a short delay.
func (b *Bot) updateLoot() {
	if b.pendingLoot == 0 {
		return
	}

	if time.Now().After(b.lootTimer) {
		b.client.LootRelease()
		b.pendingLoot = 0
	}
}

func (b *Bot) nameOf(e *client.Entity) string {
	if e.Name != "" {
		return e.Name
	}

	if e.Entry() != 0 {
		return fmt.Sprintf("entry#%d", e.Entry())
	}

	return fmt.Sprintf("0x%x", uint64(e.GUID))
}

func distance(a, b player.WorldLocation) float32 {
	dx := b.X - a.X
	dy := b.Y - a.Y

	return float32(math.Hypot(float64(dx), float64(dy)))
}
