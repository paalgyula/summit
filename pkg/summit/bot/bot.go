// Package bot implements a playable AI client for the Summit world server.
//
// The bot enters the world through pkg/summit/client, perceives nearby
// creatures from SMSG_UPDATE_OBJECT, picks a level-appropriate hostile target,
// walks to it, auto-attacks and casts spells, loots the corpse, rests and
// resurrects, then repeats. It never gets stuck on mobs it cannot kill.
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

// statusInterval is how often a summary line is logged.
const statusInterval = 30 * time.Second

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

	// LevelMaxOffset ignores creatures more than this many levels above the bot.
	LevelMaxOffset int
	// SkipElite ignores elite/rare/boss creatures.
	SkipElite bool
	// MaxHealthFactor ignores creatures whose max health exceeds
	// this multiple of the bot's own max health (0 disables the check).
	MaxHealthFactor float32
	// GiveUpAfter disengages a target that has taken no damage for this long.
	GiveUpAfter time.Duration
	// BlacklistFor is how long an ignored creature stays ignored.
	BlacklistFor time.Duration

	// RestHealthPct makes the bot rest below this health fraction.
	RestHealthPct float32
	// FleeHealthPct makes the bot disengage and retreat below this fraction.
	FleeHealthPct float32
}

// DefaultConfig returns sensible defaults for a melee/caster grinding bot.
func DefaultConfig() Config {
	return Config{
		Tick:            250 * time.Millisecond,
		EngageRange:     60,
		MeleeRange:      4,
		MoveSpeed:       7,
		WanderRadius:    12,
		LeashRadius:     80,
		Loot:            true,
		GroundFollow:    true,
		AutoCast:        true,
		SpellRange:      25,
		GCD:             1500 * time.Millisecond,
		LevelMaxOffset:  2,
		SkipElite:       true,
		MaxHealthFactor: 5,
		GiveUpAfter:     20 * time.Second,
		BlacklistFor:    2 * time.Minute,
		RestHealthPct:   0.6,
		FleeHealthPct:   0.15,
	}
}

// Stats summarises what the bot has done.
type Stats struct {
	Kills  int
	Deaths int
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

	kills  int
	deaths int

	lastCast time.Time

	blacklist map[wow.GUID]time.Time
	engagedAt time.Time

	pendingLoot wow.GUID
	lootTimer   time.Time

	resting    bool
	restLogged bool

	wanderTarget *player.WorldLocation
	lastWander   time.Time

	lastStatus time.Time

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
		client:    wc,
		cfg:       cfg,
		log:       log.With().Str("service", "bot").Logger(),
		blacklist: make(map[wow.GUID]time.Time),
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

// Stats returns the bot's counters.
func (b *Bot) Stats() Stats { return Stats{Kills: b.kills, Deaths: b.deaths} }

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

	b.handleLoot()
	b.updateLoot()
	b.drainCombatErrors()
	b.maybeStatus(self)

	// Rest until recovered before pulling again.
	if b.needsRest(self) {
		b.rest()

		return
	}

	b.wake()

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

		mob := b.findTarget(self)
		if mob == nil {
			b.wander()

			return
		}

		b.engage(mob)

		return
	}

	b.fight(target)
}

// eligible applies the target filters.
func (b *Bot) eligible(mob, self *client.Entity) bool {
	if !mob.Attackable() || b.isBlacklisted(mob.GUID) {
		return false
	}

	if self != nil && self.Level() > 0 && mob.Level() > 0 {
		if int(mob.Level()) > int(self.Level())+b.cfg.LevelMaxOffset {
			return false
		}
	}

	if b.cfg.SkipElite && mob.IsElite() {
		return false
	}

	if b.cfg.MaxHealthFactor > 0 && self != nil && self.MaxHealth() > 0 {
		limit := float32(self.MaxHealth()) * b.cfg.MaxHealthFactor
		if float32(mob.MaxHealth()) > limit {
			return false
		}
	}

	return true
}

// findTarget returns the nearest eligible creature in range and inside the leash.
func (b *Bot) findTarget(self *client.Entity) *client.Entity {
	var (
		best     *client.Entity
		bestDist = b.cfg.EngageRange
	)

	for _, e := range b.client.Entities() {
		if !b.eligible(e, self) || distance(e.Pos, b.home) > b.cfg.LeashRadius {
			continue
		}

		if d := distance(b.pos, e.Pos); d < bestDist {
			bestDist = d
			best = e
		}
	}

	return best
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
	b.engagedAt = time.Now()

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
	// Flee when badly hurt.
	if self := b.client.SelfEntity(); self != nil && self.MaxHealth() > 0 && self.HealthPct() < b.cfg.FleeHealthPct {
		b.flee(t)

		return
	}

	// Give up on a target we cannot hurt.
	if t.DamageDealtBySelf == 0 && time.Since(b.engagedAt) > b.cfg.GiveUpAfter {
		b.log.Warn().Str("name", b.nameOf(t)).Msg("target is not taking damage, blacklisting")
		b.blacklistEntity(t.GUID)
		b.disengage()

		return
	}

	// Keep the target inside the leash.
	if distance(t.Pos, b.home) > b.cfg.LeashRadius {
		b.log.Debug().Str("name", b.nameOf(t)).Msg("target beyond leash, disengaging")
		b.disengage()

		return
	}

	dist := distance(b.pos, t.Pos)

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

// flee disengages from a target and retreats home.
func (b *Bot) flee(t *client.Entity) {
	b.log.Warn().Str("name", b.nameOf(t)).Msg("low health, retreating")
	b.blacklistEntity(t.GUID)
	b.disengage()
	b.moveToward(b.home)
}

func (b *Bot) disengage() {
	b.stopAttack()
	b.target = 0
	b.swinging = false
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

// needsRest reports whether the bot should stop fighting and regenerate.
func (b *Bot) needsRest(self *client.Entity) bool {
	if self == nil || self.MaxHealth() == 0 {
		return false
	}

	return self.HealthPct() < b.cfg.RestHealthPct
}

func (b *Bot) rest() {
	b.stopAttack()
	b.stopMoving()

	if !b.resting {
		b.resting = true
		b.restLogged = false
		b.client.StandState(1) // sit
	}

	if !b.restLogged {
		b.restLogged = true
		b.log.Info().Msg("resting until recovered")
	}
}

func (b *Bot) wake() {
	if b.resting {
		b.resting = false
		b.client.StandState(0) // stand
		b.log.Info().Msg("recovered, resuming")
	}
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
		b.deaths++
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
	b.lootTimer = time.Now().Add(2 * time.Second)
}

// handleLoot drains opened loot windows, takes everything and releases.
func (b *Bot) handleLoot() {
	for {
		select {
		case loot := <-b.client.LootOpened():
			b.takeLoot(loot)
		default:
			return
		}
	}
}

func (b *Bot) takeLoot(loot client.LootInfo) {
	if loot.Gold > 0 {
		b.client.LootMoney()
	}

	for _, item := range loot.Items {
		b.client.AutostoreLootItem(item.Slot)
	}

	b.client.LootRelease()

	b.log.Info().
		Uint32("gold", loot.Gold).
		Int("items", len(loot.Items)).
		Msg("looted")

	b.pendingLoot = 0
}

// updateLoot releases a loot window the server never answered.
func (b *Bot) updateLoot() {
	if b.pendingLoot == 0 {
		return
	}

	if time.Now().After(b.lootTimer) {
		b.client.LootRelease()
		b.pendingLoot = 0
	}
}

// drainCombatErrors reacts to the server's attack rejections.
func (b *Bot) drainCombatErrors() {
	for {
		select {
		case reason := <-b.client.CombatErrors():
			b.log.Debug().Str("reason", reason).Msg("combat error")

			// A dead or unattackable target ends the engagement; the rest
			// (range/facing) are handled by continuing to approach.
			if b.target != 0 && (reason == "target is dead" || reason == "cannot attack that target") {
				b.stopAttack()
				b.target = 0
				b.swinging = false
			}
		default:
			return
		}
	}
}

func (b *Bot) maybeStatus(self *client.Entity) {
	if time.Since(b.lastStatus) < statusInterval {
		return
	}

	b.lastStatus = time.Now()

	health := "?"
	if self != nil {
		health = fmt.Sprintf("%d/%d", self.Health(), self.MaxHealth())
	}

	b.log.Info().
		Int("kills", b.kills).
		Int("deaths", b.deaths).
		Str("health", health).
		Bool("hasTarget", b.target != 0).
		Msg("status")
}

func (b *Bot) isBlacklisted(guid wow.GUID) bool {
	until, ok := b.blacklist[guid]
	if !ok {
		return false
	}

	if time.Now().After(until) {
		delete(b.blacklist, guid)

		return false
	}

	return true
}

func (b *Bot) blacklistEntity(guid wow.GUID) {
	if b.cfg.BlacklistFor <= 0 {
		return
	}

	b.blacklist[guid] = time.Now().Add(b.cfg.BlacklistFor)
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
