package world

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

// --- Ghost State Tests ---

func TestPlayerDieSetsGhostState(t *testing.T) {
	p := createTestPlayer()
	p.Health = 100
	p.MaxHealth = 100

	p.SetHealth(0)

	assert.True(t, p.IsGhost, "player should be ghost after death")
	assert.Equal(t, uint32(0), p.Health, "health should be 0")
	assert.Equal(t, uint32(0x10), p.CharFlags&0x10, "CHAR_FLAG_GHOST should be set")
	assert.True(t, p.PlayerFlags&wow.PlayerFlagsGhost != 0, "PLAYER_FLAGS_GHOST should be set")
}

func TestPlayerResurrectClearsGhostState(t *testing.T) {
	p := createTestPlayer()
	p.Health = 100
	p.MaxHealth = 100
	p.SetHealth(0) // die

	p.Resurrect(0.5, 0.5)

	assert.False(t, p.IsGhost, "player should not be ghost after resurrect")
	assert.Greater(t, p.Health, uint32(0), "health should be > 0 after resurrect")
	assert.Equal(t, uint32(0), p.CharFlags&0x10, "CHAR_FLAG_GHOST should be cleared")
	assert.Equal(t, wow.PlayerFlag(0), p.PlayerFlags&wow.PlayerFlagsGhost, "PLAYER_FLAGS_GHOST should be cleared")
}

func TestPlayerResurrectHealthPercentage(t *testing.T) {
	p := createTestPlayer()
	p.Health = 100
	p.MaxHealth = 100
	p.SetHealth(0) // die

	p.Resurrect(0.5, 0.5)

	assert.Equal(t, uint32(50), p.Health, "health should be 50% of max")
}

func TestPlayerResurrectMinHealth(t *testing.T) {
	p := createTestPlayer()
	p.Health = 10
	p.MaxHealth = 10
	p.SetHealth(0) // die

	p.Resurrect(0.01, 0.5) // 1% of 10 = 0.1 → should clamp to 1

	assert.Equal(t, uint32(1), p.Health, "health should be at least 1")
}

// --- Corpse Reclaim Delay Tests ---

func TestGetCorpseReclaimDelayFirstDeath(t *testing.T) {
	p := createTestPlayer()
	p.DeathExpireTime = 0 // no previous death

	delay := p.GetCorpseReclaimDelay(false)

	assert.Equal(t, uint32(30), delay, "first death should have 30s delay")
}

func TestGetCorpseReclaimDelaySecondDeath(t *testing.T) {
	p := createTestPlayer()
	// After first death, UpdateCorpseReclaimDelay sets DeathExpireTime = now + 300
	// For second death, count should be 1, so we need DeathExpireTime such that
	// (DeathExpireTime - 1 - now) / 300 = 1 → DeathExpireTime = now + 450
	p.DeathExpireTime = time.Now().Unix() + 450

	delay := p.GetCorpseReclaimDelay(true) // PvP uses escalating delay

	assert.Equal(t, uint32(60), delay, "second death should have 60s delay")
}

func TestGetCorpseReclaimDelayThirdDeath(t *testing.T) {
	p := createTestPlayer()
	// After two deaths, count should be 2, so DeathExpireTime = now + 901
	p.DeathExpireTime = time.Now().Unix() + 901

	delay := p.GetCorpseReclaimDelay(true) // PvP uses escalating delay

	assert.Equal(t, uint32(120), delay, "third death should have 120s delay")
}

func TestGetCorpseReclaimDelayExpired(t *testing.T) {
	p := createTestPlayer()
	// Death expired (more than 5 min ago)
	p.DeathExpireTime = time.Now().Unix() - 600

	delay := p.GetCorpseReclaimDelay(false)

	assert.Equal(t, uint32(30), delay, "expired death should reset to 30s delay")
}

func TestUpdateCorpseReclaimDelay(t *testing.T) {
	p := createTestPlayer()

	// First death
	p.UpdateCorpseReclaimDelay()
	assert.Greater(t, p.DeathExpireTime, int64(0), "DeathExpireTime should be set after first death")

	// Second death within 5 min
	p.UpdateCorpseReclaimDelay()
	// DeathExpireTime should be extended
	assert.Greater(t, p.DeathExpireTime, time.Now().Unix()+300, "DeathExpireTime should be extended")
}

// --- SMSG_PRE_RESURRECT Packet Tests ---

func TestBuildPreResurrectPacket(t *testing.T) {
	p := createTestPlayer()
	p.Health = 100
	p.MaxHealth = 100

	pkt := BuildPreResurrectPacket(p)

	assert.NotNil(t, pkt, "packet should not be nil")
	// Opcode should be ServerPreResurrect (0x494)
	assert.Equal(t, wow.ServerPreResurrect, pkt.Opcode(), "opcode should be SMSG_PRE_RESURRECT")
}

// --- SMSG_DEATH_RELEASE_LOC Packet Tests ---

func TestBuildDeathReleaseLocPacket(t *testing.T) {
	pkt := BuildDeathReleaseLocPacket(0, -8949.95, -132.66, 83.53)

	assert.NotNil(t, pkt, "packet should not be nil")
	assert.Equal(t, wow.ServerDeathReleaseLoc, pkt.Opcode(), "opcode should be SMSG_DEATH_RELEASE_LOC")
}

func TestBuildDeathReleaseLocClearPacket(t *testing.T) {
	pkt := BuildDeathReleaseLocClearPacket()

	assert.NotNil(t, pkt, "packet should not be nil")
	assert.Equal(t, wow.ServerDeathReleaseLoc, pkt.Opcode(), "opcode should be SMSG_DEATH_RELEASE_LOC")
}

// --- SMSG_DURABILITY_DAMAGE_DEATH Packet Tests ---

func TestBuildDurabilityDamageDeathPacket(t *testing.T) {
	pkt := BuildDurabilityDamageDeathPacket()

	assert.NotNil(t, pkt, "packet should not be nil")
	assert.Equal(t, wow.ServerDurabilityDamageDeath, pkt.Opcode(), "opcode should be SMSG_DURABILITY_DAMAGE_DEATH")
}

// --- Dead State Transition Tests ---

func TestKillSetsHealthToZero(t *testing.T) {
	attacker := createTestPlayer()
	victim := createTestPlayer()
	victim.MaxHealth = 100
	victim.Health = 50

	damageInfo := &CalcDamageInfo{
		Attacker:   attacker,
		Target:     victim,
		Damages:    [2]CalcDamage{{Damage: 100}},
		HitOutcome: MeleeHitNormal,
	}

	Kill(attacker, victim, damageInfo)

	assert.Equal(t, uint32(0), victim.Health, "health should be 0 after kill")
	assert.True(t, victim.IsGhost, "should be ghost after kill")
}

func TestKillPlayerSetsGhostFlags(t *testing.T) {
	attacker := createTestPlayer()
	victim := createTestPlayer()
	victim.MaxHealth = 100
	victim.Health = 50

	damageInfo := &CalcDamageInfo{
		Attacker:   attacker,
		Target:     victim,
		Damages:    [2]CalcDamage{{Damage: 100}},
		HitOutcome: MeleeHitNormal,
	}

	Kill(attacker, victim, damageInfo)

	assert.NotEqual(t, uint32(0), victim.CharFlags&0x10, "CHAR_FLAG_GHOST should be set")
}

func TestKillPlayerStopsCombat(t *testing.T) {
	attacker := createTestPlayer()
	victim := createTestPlayer()
	victim.MaxHealth = 100
	victim.Health = 50
	victim.AttackState = AttackStateSwinging
	victim.AttackTarget = 12345

	damageInfo := &CalcDamageInfo{
		Attacker:   attacker,
		Target:     victim,
		Damages:    [2]CalcDamage{{Damage: 100}},
		HitOutcome: MeleeHitNormal,
	}

	Kill(attacker, victim, damageInfo)

	assert.Equal(t, 0, victim.AttackState, "attack state should be idle")
	assert.Equal(t, uint64(0), victim.AttackTarget, "attack target should be cleared")
}

func TestKillAlreadyDeadUnit(t *testing.T) {
	attacker := createTestPlayer()
	victim := createTestPlayer()
	victim.Health = 0
	victim.IsGhost = true

	damageInfo := &CalcDamageInfo{
		Attacker:   attacker,
		Target:     victim,
		Damages:    [2]CalcDamage{{Damage: 100}},
		HitOutcome: MeleeHitNormal,
	}

	Kill(attacker, victim, damageInfo)

	// Should not change anything
	assert.True(t, victim.IsGhost, "should still be ghost")
}

// --- Helper Functions ---

func createTestPlayer() *player.Player {
	p := player.NewPlayer()
	p.ID = 1
	p.Name = "TestPlayer"
	p.Level = 10
	p.Health = 100
	p.MaxHealth = 100
	p.IsGhost = false
	p.DeathExpireTime = 0
	p.Location = player.WorldLocation{X: 0, Y: 0, Z: 0, Map: 0}
	p.CorpseLocation = player.WorldLocation{X: 0, Y: 0, Z: 0, Map: 0}

	// Initialize object for update fields
	p.Init()

	return p
}
