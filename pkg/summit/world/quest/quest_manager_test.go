package quest

import (
	"testing"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

func testManager() *Manager {
	m := &Manager{
		templates:        make(map[uint32]*Quest),
		creatureQuests:   make(map[uint32][]uint32),
		creatureInvolved: make(map[uint32][]uint32),
	}

	m.templates[100] = &Quest{ID: 100, MinLevel: 10, Level: 12, AllowableRaces: 0}
	m.templates[101] = &Quest{ID: 101, MinLevel: 1, Level: 3, AllowableRaces: 1}  // Humans only (bit 0)
	m.templates[102] = &Quest{ID: 102, MinLevel: 1, Level: 3, RequiredClasses: 1} // Warrior only (bit 0)
	m.templates[103] = &Quest{ID: 103, MinLevel: 1, Level: 5, PrevQuestId: 100}
	m.templates[104] = &Quest{ID: 104, MinLevel: 1, Level: 5, SpecialFlags: QuestSpecialRepeatable}
	m.templates[105] = &Quest{ID: 105, MinLevel: 1, Level: 5, NextQuestId: 106}
	m.templates[106] = &Quest{ID: 106, MinLevel: 1, Level: 6}

	m.creatureQuests[900] = []uint32{100, 101, 102, 103, 104, 105, 106}

	return m
}

func TestCanTakeQuest_Level(t *testing.T) {
	m := testManager()
	info := PlayerInfo{Level: 5, Race: wow.RaceHuman, Class: wow.ClassWarior}

	// MinLevel 10, player level 5 → not available.
	assert.False(t, m.CanTakeQuest(info, map[uint32]*QuestStatusData{}, m.templates[100]))

	info.Level = 10
	assert.True(t, m.CanTakeQuest(info, map[uint32]*QuestStatusData{}, m.templates[100]))
}

func TestCanTakeQuest_Race(t *testing.T) {
	m := testManager()
	human := PlayerInfo{Level: 80, Race: wow.RaceHuman, Class: wow.ClassWarior}
	orc := PlayerInfo{Level: 80, Race: wow.RaceOrc, Class: wow.ClassWarior}

	assert.True(t, m.CanTakeQuest(human, map[uint32]*QuestStatusData{}, m.templates[101]))
	assert.False(t, m.CanTakeQuest(orc, map[uint32]*QuestStatusData{}, m.templates[101]))
}

func TestCanTakeQuest_Class(t *testing.T) {
	m := testManager()
	warrior := PlayerInfo{Level: 80, Race: wow.RaceHuman, Class: wow.ClassWarior}
	mage := PlayerInfo{Level: 80, Race: wow.RaceHuman, Class: wow.ClassMage}

	assert.True(t, m.CanTakeQuest(warrior, map[uint32]*QuestStatusData{}, m.templates[102]))
	assert.False(t, m.CanTakeQuest(mage, map[uint32]*QuestStatusData{}, m.templates[102]))
}

func TestCanTakeQuest_Rewarded(t *testing.T) {
	m := testManager()
	info := PlayerInfo{
		Level:    80,
		Race:     wow.RaceHuman,
		Class:    wow.ClassWarior,
		Rewarded: map[uint32]bool{100: true, 104: true},
	}

	// Non-repeatable rewarded quest is gone.
	assert.False(t, m.CanTakeQuest(info, map[uint32]*QuestStatusData{}, m.templates[100]))

	// Repeatable rewarded quest stays available.
	assert.True(t, m.CanTakeQuest(info, map[uint32]*QuestStatusData{}, m.templates[104]))
}

func TestCanTakeQuest_PrevQuest(t *testing.T) {
	m := testManager()
	info := PlayerInfo{Level: 80, Race: wow.RaceHuman, Class: wow.ClassWarior}

	// PrevQuestId 100 not done → locked.
	assert.False(t, m.CanTakeQuest(info, map[uint32]*QuestStatusData{}, m.templates[103]))

	info.Rewarded = map[uint32]bool{100: true}
	assert.True(t, m.CanTakeQuest(info, map[uint32]*QuestStatusData{}, m.templates[103]))
}

func TestCanTakeQuest_AlreadyInLog(t *testing.T) {
	m := testManager()
	info := PlayerInfo{Level: 80, Race: wow.RaceHuman, Class: wow.ClassWarior}
	inLog := map[uint32]*QuestStatusData{100: {Status: QuestStatusIncomplete}}

	assert.False(t, m.CanTakeQuest(info, inLog, m.templates[100]))
}

func TestCanTakeQuest_NextQuestRewarded(t *testing.T) {
	m := testManager()
	info := PlayerInfo{
		Level:    80,
		Race:     wow.RaceHuman,
		Class:    wow.ClassWarior,
		Rewarded: map[uint32]bool{106: true},
	}

	assert.False(t, m.CanTakeQuest(info, map[uint32]*QuestStatusData{}, m.templates[105]))
}

func TestAvailableQuestsForCreature(t *testing.T) {
	m := testManager()
	// Level 1 human warrior: only the low-level, race/class-compatible quests.
	info := PlayerInfo{Level: 1, Race: wow.RaceHuman, Class: wow.ClassWarior}

	available := m.AvailableQuestsForCreature(info, map[uint32]*QuestStatusData{}, 900)

	ids := make(map[uint32]bool)
	for _, q := range available {
		ids[q.ID] = true
	}

	assert.True(t, ids[101], "human quest should be available")
	assert.True(t, ids[102], "warrior quest should be available")
	assert.True(t, ids[105], "unblocked chain quest should be available")
	assert.False(t, ids[100], "level-10 quest should be filtered out")
	assert.False(t, ids[103], "chain-locked quest should be filtered out")
}
