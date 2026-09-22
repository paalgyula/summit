package world

// xp.go — Experience Engine: kill XP formula, GiveXP, SMSG_LOG_XPGAIN,
//          SMSG_EXPLORATION_EXPERIENCE, group XP rate.
//
// Reference: AzerothCore
//   - Formulas.h / Formulas.cpp   (GetGrayLevel, GetZeroDifference, BaseGain, Gain)
//   - Player.cpp                  (GiveXP, GiveLevel, GetXPRestBonus, SendLogXPGain)
//   - PlayerMisc.cpp              (SendExplorationExperience)
//   - KillRewarder.cpp            (group XP splitting — TODO: future)

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

const (
	// MaxPlayerLevel is the WotLK level cap.
	MaxPlayerLevel uint8 = 80
	// MinTalentLevel is the level at which talent points start.
	MinTalentLevel uint8 = 10
)

// ContentLevels determines the base XP tier for a mob based on the map's expansion.
// Source: AzerothCore DBCStores.h (ContentLevels enum).
type ContentLevels uint8

const (
	Content1_60 ContentLevels = 0 // Vanilla: Kalimdor, Eastern Kingdoms
	Content61_70 ContentLevels = 1 // TBC: Outland
	Content71_80 ContentLevels = 2 // WotLK: Northrend
)

// wotlkXPTable contains the total XP required to reach a given level from 1 to 80.
// Index 0 = XP needed to reach level 2, index 79 = XP to reach 81 (unused sentinel).
// Source: AzerothCore xp_per_level table (ObjectMgr::LoadPlayerXPForLevel).
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

// GetGrayLevel returns the level below which a mob gives zero XP.
// Source: AzerothCore Formulas.h:46 (Acore::XP::GetGrayLevel).
//
//nolint:gomnd
func GetGrayLevel(plLevel uint8) uint8 {
	switch {
	case plLevel <= 5:
		return 0
	case plLevel <= 39:
		return plLevel - 5 - plLevel/10
	case plLevel <= 59:
		return plLevel - 1 - plLevel/5
	default:
		return plLevel - 9
	}
}

// GetZeroDifference returns the zero-difference factor used in the XP formula
// for mobs below the player's level.
// Source: AzerothCore Formulas.h:82 (Acore::XP::GetZeroDifference).
//
//nolint:gomnd
func GetZeroDifference(plLevel uint8) uint8 {
	switch {
	case plLevel < 8:
		return 5
	case plLevel < 10:
		return 6
	case plLevel < 12:
		return 7
	case plLevel < 16:
		return 8
	case plLevel < 20:
		return 9
	case plLevel < 30:
		return 11
	case plLevel < 40:
		return 12
	case plLevel < 45:
		return 13
	case plLevel < 50:
		return 14
	case plLevel < 55:
		return 15
	case plLevel < 60:
		return 16
	default:
		return 17
	}
}

// BaseGain returns the base XP for killing a mob of `mobLevel` by a player
// of `plLevel`, before elite multiplier and server rates.
// Source: AzerothCore Formulas.cpp:27 (Acore::XP::BaseGain).
//
//nolint:gomnd
func BaseGain(plLevel, mobLevel uint8, content ContentLevels) uint32 {
	var nBaseExp uint32

	switch content {
	case Content1_60:
		nBaseExp = 45
	case Content61_70:
		nBaseExp = 235
	case Content71_80:
		nBaseExp = 580
	default:
		nBaseExp = 45
	}

	if mobLevel >= plLevel {
		nLevelDiff := mobLevel - plLevel
		if nLevelDiff > 4 {
			nLevelDiff = 4
		}

		return ((uint32(plLevel)*5 + nBaseExp) * (20 + uint32(nLevelDiff)) / 10 + 1) / 2
	}

	// Mob is below player level
	grayLevel := GetGrayLevel(plLevel)
	if mobLevel > grayLevel {
		ZD := GetZeroDifference(plLevel)

		return (uint32(plLevel)*5 + nBaseExp) * (uint32(ZD) + uint32(mobLevel) - uint32(plLevel)) / uint32(ZD)
	}

	// Grey mob — zero XP
	return 0
}

// CalculateKillXP returns the base XP a player of `playerLevel` earns for
// killing a mob of `mobLevel`, including the elite multiplier.
// This is a convenience wrapper around BaseGain for callers that don't need
// the full Gain() pipeline.
func CalculateKillXP(playerLevel, mobLevel uint8, elite bool) uint32 {
	if playerLevel >= MaxPlayerLevel {
		return 0
	}

	gain := BaseGain(playerLevel, mobLevel, Content1_60)

	if elite {
		gain = uint32(float32(gain) * 2.0)
	}

	return gain
}

// XpInGroupRate returns the group XP rate multiplier based on the number
// of alive members in the group.
// Source: AzerothCore Formulas.h:119 (Acore::XP::xp_in_group_rate).
//
//nolint:gomnd
func XpInGroupRate(count uint32, isRaid bool) float32 {
	if isRaid {
		// FIXME: Must apply decrease modifiers depending on raid size.
		return 1.0
	}

	switch {
	case count <= 2:
		return 1.0
	case count == 3:
		return 1.166
	case count == 4:
		return 1.3
	default:
		return 1.4
	}
}

// GiveXP awards XP to the player, applies rest bonus, sends SMSG_LOG_XPGAIN,
// and triggers level-up loop.
// Source: AzerothCore Player.cpp:2404 (Player::GiveXP).
func (gc *WorldSession) GiveXP(xp uint32, victim wow.GUID, groupRate float32) {
	if gc.player == nil || xp < 1 {
		return
	}

	p := gc.player

	if p.Level >= MaxPlayerLevel {
		return
	}

	// Check XP gain toggle
	if p.PlayerFlags&wow.PlayerFlagsNoXpGain != 0 {
		return
	}

	// Half XP if partial play time (CAIS)
	if p.PlayerFlags&wow.PlayerFlagsPartialPlayTime != 0 {
		if xp > 1 {
			xp /= 2
		}
	}

	// Calculate rest bonus (RaF not implemented yet — rest only)
	bonusXP := uint32(0)
	if victim != 0 { // only for kill XP, not quest/exploration
		bonusXP = GetXPRestBonus(p, xp)
	}

	// Send SMSG_LOG_XPGAIN
	gc.sendLogXpgain(xp, victim, bonusXP, false, groupRate)

	// Level-up loop
	newXP := p.XP + xp + bonusXP

	for p.Level < MaxPlayerLevel {
		needed := NextLevelXP(p.Level)
		if needed == 0 || newXP < needed {
			break
		}

		newXP -= needed
		p.Level++

		gc.applyLevelUp(p)
	}

	p.XP = newXP

	// Update update fields
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
// new level. Uses a simple linear approximation; real data would come from
// the gtOCT* DBC tables and playerClassLevelStats.
//
//nolint:gomnd
func recalcLevelUpStats(p *player.Player) (hp, mana, str, agi, sta, int_, spi uint32) {
	lv := uint32(p.Level)

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

// sendLogXpgain sends SMSG_LOG_XPGAIN to the client.
// Source: AzerothCore Player.cpp:2385 (Player::SendLogXPGain).
//
// Packet layout:
//
//	ObjectGuid  victim GUID      (8 bytes)
//	uint32      givenXP          (total XP + bonus)
//	uint8       type             (0 = kill, 1 = non-kill)
//	[if kill]:
//	  uint32    baseXP           (XP without bonus)
//	  float     groupRate        (group bonus rate, 1.0 = none)
//	uint8       recruitAFriend   (1 if RaF bonus included)
func (gc *WorldSession) sendLogXpgain(baseXP uint32, victim wow.GUID, bonusXP uint32, recruitAFriend bool, groupRate float32) {
	pkt := wow.NewPacket(wow.ServerLogXpgain)

	// Victim GUID (0 for non-kill XP)
	_ = pkt.Write(uint64(victim))

	// Total XP given (base + bonus)
	_ = pkt.Write(baseXP + bonusXP)

	// Type: 0 = kill XP, 1 = non-kill XP
	if victim != 0 {
		_ = pkt.Write(uint8(0))
		_ = pkt.Write(baseXP) // XP without bonus
		_ = pkt.Write(groupRate)
	} else {
		_ = pkt.Write(uint8(1))
	}

	// Recruit-A-Friend flag
	if recruitAFriend {
		_ = pkt.Write(uint8(1))
	} else {
		_ = pkt.Write(uint8(0))
	}

	gc.Send(pkt)
}

// SendExplorationExperience sends SMSG_EXPLORATION_EXPERIENCE when a player
// discovers a new area.
// Source: AzerothCore PlayerMisc.cpp:161 (Player::SendExplorationExperience).
func (gc *WorldSession) SendExplorationExperience(areaID, xp uint32) {
	pkt := wow.NewPacket(wow.ServerExplorationExperience)

	_ = pkt.Write(areaID)
	_ = pkt.Write(xp)

	gc.Send(pkt)
}
