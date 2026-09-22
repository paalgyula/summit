// Package bot implements a playable AI client for the Summit world server.
//
// The bot enters the world through pkg/summit/client, perceives nearby
// creatures from SMSG_UPDATE_OBJECT, walks to a hostile target, auto-attacks
// it, detects the kill (the server does not broadcast NPC health, so it is
// inferred from the bot's own melee hits), loots the corpse and repeats.
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
	// Loot controls whether corpses are looted after a kill.
	Loot bool
}

// DefaultConfig returns sensible defaults for a melee grinding bot.
func DefaultConfig() Config {
	return Config{
		Tick:         250 * time.Millisecond,
		EngageRange:  60,
		MeleeRange:   4,
		MoveSpeed:    7,
		WanderRadius: 12,
		Loot:         true,
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

	pendingLoot wow.GUID
	lootTimer   time.Time

	wanderTarget *player.WorldLocation
	lastWander   time.Time
}

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

	if self != nil {
		if self.HasPos {
			b.syncPosition(self)
		}

		if self.MaxHealth() > 0 && self.Health() == 0 {
			b.log.Warn().Msg("bot is dead; waiting")
			return
		}
	}

	b.updateLoot()

	target := b.validTarget()

	if target == nil {
		if b.target != 0 {
			b.target = 0
			b.swinging = false
		}

		mob := b.client.NearestAttackable(&b.pos, b.cfg.EngageRange)
		if mob == nil {
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
	if distance(b.pos, t.Pos) > b.cfg.MeleeRange {
		b.moveToward(t.Pos)
		return
	}

	b.stopMoving()

	if !b.swinging {
		b.client.AttackSwing(t.GUID)
		b.swinging = true
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

	if step > dist {
		step = dist
	}

	b.pos.X += float32(math.Cos(float64(o))) * step
	b.pos.Y += float32(math.Sin(float64(o))) * step
	b.pos.O = o

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
