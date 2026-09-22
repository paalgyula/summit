package quest

// quest_xp.go — Quest XP calculation using QuestXP.dbc.
//
// Reference: AzerothCore
//   - QuestDef.cpp:199 (Quest::XPValue)
//   - PlayerQuest.cpp:1475 (Player::CalculateQuestRewardXP)

import (
	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
)

// QuestXPStore holds the QuestXP.dbc entries loaded at startup.
// Key: quest level (ID in QuestXP.dbc).
var QuestXPStore map[uint32]wotlk.QuestXPEntry

// InitQuestXPStore loads QuestXP.dbc and initializes the store.
func InitQuestXPStore(entries []wotlk.QuestXPEntry) {
	QuestXPStore = make(map[uint32]wotlk.QuestXPEntry, len(entries))

	for _, e := range entries {
		QuestXPStore[e.ID] = e
	}
}

// XPValue computes the quest XP reward for a player of the given level,
// using the QuestXP.dbc data and the quest's difficulty column.
// Source: AzerothCore QuestDef.cpp:199 (Quest::XPValue).
//
//nolint:gomnd
func (q *Quest) XPValue(playerLevel uint8) uint32 {
	questLevel := q.Level
	if questLevel <= 0 {
		questLevel = int32(playerLevel)
	}

	entry, exists := QuestXPStore[uint32(questLevel)]
	if !exists {
		return 0
	}

	// Difficulty factor based on level difference
	diffFactor := 2*(questLevel-int32(playerLevel)) + 20

	if diffFactor < 1 {
		diffFactor = 1
	} else if diffFactor > 10 {
		diffFactor = 10
	}

	xp := uint32(diffFactor) * entry.XPForDifficulty(q.RewardXPDifficulty) / 10

	// Rounding to nice numbers (AzerothCore QuestDef.cpp:219-234)
	switch {
	case xp <= 100:
		xp = 5 * ((xp + 2) / 5)
	case xp <= 500:
		xp = 10 * ((xp + 5) / 10)
	case xp <= 1000:
		xp = 25 * ((xp + 12) / 25)
	default:
		xp = 50 * ((xp + 25) / 50)
	}

	return xp
}

// CalculateQuestRewardXP computes the final XP reward for a quest,
// applying any quest rate multipliers.
// Source: AzerothCore PlayerQuest.cpp:1475 (Player::CalculateQuestRewardXP).
func (q *Quest) CalculateQuestRewardXP(playerLevel uint8) uint32 {
	xp := q.XPValue(playerLevel)

	// TODO: apply RATE_XP_QUEST server rate multiplier
	// TODO: apply SPELL_AURA_MOD_XP_QUEST_PCT aura multipliers

	return xp
}
