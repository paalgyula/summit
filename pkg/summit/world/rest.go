package world

// rest.go — Rest XP System: accumulation, consumption, persistence.
//
// Reference: AzerothCore
//   - Player.cpp:SetRestBonus, GetXPRestBonus
//   - PlayerUpdates.cpp: in-game rest tick
//   - PlayerStorage.cpp: offline rest accumulation

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Rest state visual constants for PLAYER_BYTES_2 byte 3.
// Source: AzerothCore Player.h RestStateVisual enum.
const (
	restStateRested        = 0x01 // resting in inn
	restStateNotRafLinked = 0x02 // normal (not rested)
	restStateRafLinked    = 0x06 // Recruit-A-Friend linked
)

// Rest tick rate constants.
// Source: AzerothCore PlayerUpdates.cpp, PlayerStorage.cpp.
const (
	// restBubbleOnline is the rest accumulation rate while resting in-game.
	// Full rest reached in ~13.3 days of /played time in inn.
	restBubbleOnline float64 = 0.125

	// restBubbleOfflineWild is the offline rest rate when NOT in a tavern/city.
	restBubbleOfflineWild float64 = 0.031

	// restBubbleOfflineTavern is the offline rest rate when logged out in tavern/city.
	restBubbleOfflineTavern float64 = 0.125

	// restMaxBonusMultiplier is the maximum rest bonus as a multiple of nextLevelXP.
	// Max rest = nextLevelXP * 1.5 / 2 = 0.75 * nextLevelXP (client doubles for display).
	restMaxBonusMultiplier float64 = 1.5

	// restThreshold is the minimum rest bonus to show rested visual state.
	restThreshold float64 = 10.0
)

// SetRestBonus sets the player's rest bonus, capping at the maximum allowed
// and updating the visual rest state on the client.
// Source: AzerothCore Player.cpp:SetRestBonus (line 10369).
func SetRestBonus(p *player.Player, bonus float64) {
	if p.Level >= MaxPlayerLevel {
		bonus = 0
	}

	if bonus < 0 {
		bonus = 0
	}

	// Max rest bonus = nextLevelXP * multiplier / 2
	// The client doubles the value for display, so we divide by 2 here.
	nextXP := NextLevelXP(p.Level)
	maxBonus := float64(nextXP) * restMaxBonusMultiplier / 2.0

	if bonus > maxBonus {
		bonus = maxBonus
	}

	p.RestBonus = bonus

	// Update PLAYER_REST_STATE_EXPERIENCE (the rest bonus value)
	p.Object.SetUInt32Value(object.PlayerRestStateExperience, uint32(bonus))

	// Update rest state visual in PLAYER_BYTES_2 byte 3
	setRestStateVisual(p)
}

// GetRestBonus returns the player's current rest bonus.
func GetRestBonus(p *player.Player) float64 {
	return p.RestBonus
}

// GetXPRestBonus consumes rest bonus up to the given XP amount and returns
// the bonus XP to add. This is called during GiveXP when a player kills a mob.
// Source: AzerothCore Player.cpp:GetXPRestBonus (line 9096).
func GetXPRestBonus(p *player.Player, xp uint32) uint32 {
	restedBonus := uint32(p.RestBonus)

	if restedBonus > xp {
		restedBonus = xp
	}

	SetRestBonus(p, p.RestBonus-float64(restedBonus))

	return restedBonus
}

// UpdateRestBonus ticks the rest bonus accumulation for a resting player.
// Called periodically while the player is online and has PLAYER_FLAGS_RESTING set.
// Source: AzerothCore PlayerUpdates.cpp (line 243-261).
func UpdateRestBonus(p *player.Player, now int64) {
	if !p.IsResting || p.RestTime <= 0 {
		return
	}

	timeDiff := now - p.RestTime
	if timeDiff < 10 {
		return
	}

	p.RestTime = now

	// Accumulation rate: nextLevelXP / 72000 per second × bubble rate
	nextXP := float64(NextLevelXP(p.Level))
	extraPerSec := (nextXP / 72000.0) * restBubbleOnline

	SetRestBonus(p, p.RestBonus+float64(timeDiff)*extraPerSec)
}

// CalculateOfflineRestBonus accumulates rest bonus for time spent offline.
// The rate depends on whether the player was resting at logout.
// Source: AzerothCore PlayerStorage.cpp (line 5517-5531).
func CalculateOfflineRestBonus(p *player.Player, now int64) {
	if p.LogoutTime <= 0 || now <= p.LogoutTime {
		return
	}

	timeDiff := now - p.LogoutTime

	// Select bubble rate based on rest state at logout
	bubble := restBubbleOfflineWild
	if p.IsLogoutResting {
		bubble = restBubbleOfflineTavern
	}

	// Client doubles the value, so divide by 2
	nextXP := float64(NextLevelXP(p.Level))
	SetRestBonus(p, p.RestBonus+float64(timeDiff)*(nextXP/144000.0)*bubble)
}

// SetRestFlag marks the player as resting (in inn/city).
// Source: AzerothCore Player.cpp:SetRestFlag (line 16501).
func SetRestFlag(p *player.Player) {
	if !p.IsResting {
		p.IsResting = true
		p.RestTime = currentUnixTime()
		p.PlayerFlags |= wow.PlayerFlagsResting
	}
}

// RemoveRestFlag clears the resting state.
// Source: AzerothCore Player.cpp:RemoveRestFlag (line 16516).
func RemoveRestFlag(p *player.Player) {
	if p.IsResting {
		p.IsResting = false
		p.RestTime = 0
		p.PlayerFlags &^= wow.PlayerFlagsResting
	}
}

// setRestStateVisual updates the rest state byte in PLAYER_BYTES_2.
// Source: AzerothCore Player.cpp:SetRestBonus (line 10389-10397).
func setRestStateVisual(p *player.Player) {
	var restState uint32

	switch {
	case p.RestBonus > restThreshold:
		restState = restStateRested
	case p.RestBonus <= 1:
		restState = restStateNotRafLinked
	default:
		restState = restStateNotRafLinked
	}

	// Read current PLAYER_BYTES_2, replace byte 3 (bits 24-31)
	current := p.Object.GetUInt32Value(object.PlayerBytes_2)
	current &^= 0xFF000000           // clear byte 3
	current |= restState << 24       // set new rest state

	p.Object.SetUInt32Value(object.PlayerBytes_2, current)
}

// currentUnixTime returns the current unix timestamp in seconds.
func currentUnixTime() int64 {
	return time.Now().Unix()
}
