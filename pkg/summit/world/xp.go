package world

// xp.go — Experience Engine: kill XP formula, GiveXP, SMSG_LEVELUP_INFO.
//
// Reference: AzerothCore
//   - src/server/game/Entities/Player/Player.cpp  (GiveXP, CheckLevelups)
//   - src/server/game/Formulas/Formulas.h         (BaseGain, XP::Gain)
//   - src/server/game/Entities/Player/PlayerUpdates.cpp (UpdateStats)

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

const (
	// MaxPlayerLevel is the WotLK level cap.
	MaxPlayerLevel uint8 = 80
	// MaxTalentLevel is the level at which talent points start.
	MinTalentLevel uint8 = 10
)

// wotlkXPTable contains the total XP required to reach a given level from 1 to 80.
// Index 0 = XP needed to reach level 2, index 79 = XP to reach 81 (unused sentinel).
// Source: AzerothCore xp_per_level table (Player.cpp, ObjectMgr::LoadPlayerXPForLevel).
//
//nolint:gomnd
var wotlkXPTable = [81]uint32{
	0,       // level 1  → 0 XP to start
	400,     // level 2
	900,     // level 3
	1400,    // level 4
	2100,    // level 5
	2800,    // level 6
	3600,    // level 7
	4500,    // level 8
	5400,    // level 9
	6500,    // level 10
	7600,    // level 11
	8800,    // level 12
	10100,   // level 13
	11400,   // level 14
	12900,   // level 15
	14400,   // level 16
	16000,   // level 17
	17700,   // level 18
	19400,   // level 19
	21300,   // level 20
	23200,   // level 21
	25200,   // level 22
	27300,   // level 23
	29400,   // level 24
	31800,   // level 25
	34200,   // level 26
	36800,   // level 27
	39400,   // level 28
	42100,   // level 29
	44800,   // level 30
	47800,   // level 31
	50800,   // level 32
	54000,   // level 33
	57200,   // level 34
	60700,   // level 35
	64200,   // level 36
	68000,   // level 37
	71800,   // level 38
	75900,   // level 39
	80100,   // level 40
	84700,   // level 41
	89400,   // level 42
	94300,   // level 43
	99400,   // level 44
	104700,  // level 45
	110200,  // level 46
	115900,  // level 47
	121800,  // level 48
	128000,  // level 49
	134400,  // level 50
	141100,  // level 51
	148000,  // level 52
	155200,  // level 53
	162600,  // level 54
	170200,  // level 55
	178000,  // level 56
	186100,  // level 57
	194400,  // level 58
	202900,  // level 59
	211700,  // level 60
	221400,  // level 61
	231400,  // level 62
	241700,  // level 63
	252200,  // level 64
	263100,  // level 65
	274300,  // level 66
	285800,  // level 67
	297600,  // level 68
	309700,  // level 69
	322100,  // level 70
	634900,  // level 71 (TBC → WotLK gap)
	713800,  // level 72
	798600,  // level 73
	889600,  // level 74
	986800,  // level 75
	1090600, // level 76
	1200300, // level 77
	1316300, // level 78
	1438900, // level 79
	1568400, // level 80 (max; sentinel for cap check)
}

// NextLevelXP returns the XP required to advance from `level` to `level+1`.
// Returns 0 if level is already MaxPlayerLevel or out of range.
func NextLevelXP(level uint8) uint32 {
	if level == 0 || level >= MaxPlayerLevel {
		return 0
	}
	return wotlkXPTable[level]
}

// CalculateKillXP returns the base XP a player of `playerLevel` earns for
// killing a mob of `mobLevel`.  The elite multiplier doubles the reward.
//
// Formula mirrors AzerothCore Formulas::XP::Gain / BaseGain:
//   - Grey (mob 5+ below player) → 0 XP
//   - Green/Yellow/Orange/Red    → scaled reward
//
//nolint:gomnd
func CalculateKillXP(playerLevel, mobLevel uint8, elite bool) uint32 {
	if playerLevel >= MaxPlayerLevel {
		return 0
	}

	// Grey threshold: mob is 5 levels below player (below the "challenge" floor).
	greyThreshold := int16(playerLevel) - greyLevelDelta(playerLevel)
	if int16(mobLevel) < greyThreshold {
		return 0
	}

	// Base XP at equal level (AzerothCore: baseGain = mobLevel * 5 + 45).
	baseXP := uint32(mobLevel)*5 + 45

	// Level difference factor: lower mob → less XP, higher → full or slight bonus.
	diff := int16(mobLevel) - int16(playerLevel)
	var factor float32

	switch {
	case diff >= 5:
		factor = 1.20 // red — extra danger bonus
	case diff >= 3:
		factor = 1.10 // orange
	case diff >= 0:
		factor = 1.00 // yellow/equal
	case diff >= -2:
		factor = 0.70 // green
	default:
		factor = 0.40 // low-green (not yet grey)
	}

	xp := float32(baseXP) * factor

	if elite {
		xp *= 2.0
	}

	return uint32(xp)
}

// greyLevelDelta returns how many levels below the player makes a mob "grey".
// Source: AzerothCore Player.cpp GetGrayLevel().
//
//nolint:gomnd
func greyLevelDelta(level uint8) int16 {
	switch {
	case level <= 5:
		return 0
	case level <= 39:
		return int16((level / 10) + 3)
	case level <= 59:
		return int16((level / 5) + 1)
	default:
		return int16(level / 5)
	}
}

// GiveXP awards `amount` XP to the player and triggers level-up if
// `XP >= NextLevelXP`.  Called by handleCreatureDeath and quest reward code.
//
// Ref: AzerothCore Player::GiveXP, Player::CheckLevelups.
func (gc *WorldSession) GiveXP(amount uint32) {
	if gc.player == nil || amount == 0 {
		return
	}

	p := gc.player

	if p.Level >= MaxPlayerLevel {
		return
	}

	// Accumulate XP
	p.XP += amount
	p.Object.SetUInt32Value(object.PlayerXp, p.XP)

	// Broadcast XP gain log (optional — some servers omit this; AzerothCore sends it)
	// We skip the XP_GAINED server notification for now; the XP field update is enough.

	// Level-up loop (in case of huge XP grants)
	for p.Level < MaxPlayerLevel {
		needed := NextLevelXP(p.Level)
		if needed == 0 || p.XP < needed {
			break
		}

		p.XP -= needed
		p.Level++

		gc.applyLevelUp(p)
	}

	// Persist final XP and NextLevelXP update fields
	p.Object.SetUInt32Value(object.PlayerXp, p.XP)
	p.Object.SetUInt32Value(object.PlayerNextLevelXp, NextLevelXP(p.Level))
}

// applyLevelUp handles all side-effects of a single level gain.
func (gc *WorldSession) applyLevelUp(p *player.Player) {
	// Update UNIT_FIELD_LEVEL
	p.Object.SetUInt32Value(object.UnitFieldLevel, uint32(p.Level))
	p.Object.SetUInt32Value(object.PlayerNextLevelXp, NextLevelXP(p.Level))

	// Recalculate stats for new level
	hpGain, manaGain, strGain, agiGain, staGain, intGain, spiGain := recalcLevelUpStats(p)

	// Fully restore Health and primary Power on level-up (Blizzard behaviour)
	p.MaxHealth += hpGain
	p.Health = p.MaxHealth
	p.Object.SetUInt32Value(object.UnitFieldMaxhealth, p.MaxHealth)
	p.Object.SetUInt32Value(object.UnitFieldHealth, p.Health)

	primaryPower := p.GetPrimaryPowerType()
	if int(primaryPower) < wow.MaxPowerTypes {
		p.MaxPower[primaryPower] += manaGain
		p.Power[primaryPower] = p.MaxPower[primaryPower]

		p.Object.SetUInt32Value(
			object.UpdateField(int(object.UnitFieldMaxpower1)+int(primaryPower)),
			p.MaxPower[primaryPower],
		)
		p.Object.SetUInt32Value(
			object.UpdateField(int(object.UnitFieldPower1)+int(primaryPower)),
			p.Power[primaryPower],
		)
	}

	// Grant a talent point every level >= MinTalentLevel
	if p.Level >= MinTalentLevel {
		current := p.Object.GetUInt32Value(object.PlayerCharacterPoints1)
		p.Object.SetUInt32Value(object.PlayerCharacterPoints1, current+1)
	}

	// Send SMSG_LEVELUP_INFO
	gc.sendLevelUpInfo(p.Level, hpGain, manaGain, strGain, agiGain, staGain, intGain, spiGain)

	gc.log.Info().
		Uint8("level", p.Level).
		Str("player", p.Name).
		Msg("player levelled up")
}

// recalcLevelUpStats returns (hp, mana, str, agi, sta, int, spi) gains for the
// new level.  Uses a simple linear approximation; real data would come from
// the gtOCT* DBC tables.
//
//nolint:gomnd
func recalcLevelUpStats(p *player.Player) (hp, mana, str, agi, sta, int_, spi uint32) {
	lv := uint32(p.Level)

	// Stamina drives HP; Intellect drives Mana (rough WotLK approximations).
	staGain := uint32(1)
	intGain := uint32(1)

	switch p.Class {
	case wow.ClassWarior, wow.ClassPaladin, wow.ClassDeathKnight:
		staGain = 2
		hp = lv * 2
	case wow.ClassPriest, wow.ClassMage, wow.ClassWarlock:
		intGain = 2
		mana = lv * 2
		hp = lv
	case wow.ClassHunter, wow.ClassRogue:
		hp = lv + lv/2
		mana = lv
	case wow.ClassDruid, wow.ClassShaman:
		hp = lv
		mana = lv
	default:
		hp = lv
	}

	if hp == 0 {
		hp = lv
	}
	if mana == 0 {
		mana = lv
	}

	return hp, mana, 1, 1, staGain, intGain, 1
}


