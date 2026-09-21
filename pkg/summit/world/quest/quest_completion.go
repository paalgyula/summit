package quest

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// CanCompleteQuest checks if all objectives for a quest are met.
func (m *Manager) CanCompleteQuest(playerQuests map[uint32]*QuestStatusData, questID uint32) bool {
	quest := m.templates[questID]
	if quest == nil {
		return false
	}

	status, exists := playerQuests[questID]
	if !exists || status.Status != QuestStatusIncomplete {
		return false
	}

	// Check kill/collection objectives
	for i := 0; i < 4; i++ {
		if quest.RequiredNpcOrGoCount[i] > 0 && status.CreatureOrGOCount[i] < quest.RequiredNpcOrGoCount[i] {
			return false
		}
	}

	// Check item objectives
	for i := 0; i < 6; i++ {
		if quest.RequiredItemCount[i] > 0 && status.ItemCount[i] < quest.RequiredItemCount[i] {
			return false
		}
	}

	// Check exploration
	if quest.HasSpecialFlag(QuestSpecialExplorationOrEvent) && !status.Explored {
		return false
	}

	return true
}

// OnCreatureKilled processes kill credit for all active quests on a player.
func (m *Manager) OnCreatureKilled(playerQuests map[uint32]*QuestStatusData, creatureEntry uint32) []uint32 {
	var completedQuests []uint32

	for questID, status := range playerQuests {
		if status.Status != QuestStatusIncomplete {
			continue
		}

		quest := m.templates[questID]
		if quest == nil {
			continue
		}

		updated := false

		for i := 0; i < 4; i++ {
			if quest.RequiredNpcOrGo[i] > 0 &&
				uint32(quest.RequiredNpcOrGo[i]) == creatureEntry &&
				status.CreatureOrGOCount[i] < quest.RequiredNpcOrGoCount[i] {
				status.CreatureOrGOCount[i]++
				updated = true
			}
		}

		if updated && m.CanCompleteQuest(playerQuests, questID) {
			status.Status = QuestStatusComplete
			completedQuests = append(completedQuests, questID)
		}
	}

	return completedQuests
}

// OnItemAdded processes item credit for all active quests on a player.
func (m *Manager) OnItemAdded(playerQuests map[uint32]*QuestStatusData, itemEntry uint32, count uint32) []uint32 {
	var completedQuests []uint32

	for questID, status := range playerQuests {
		if status.Status != QuestStatusIncomplete {
			continue
		}

		quest := m.templates[questID]
		if quest == nil {
			continue
		}

		updated := false

		for i := 0; i < 6; i++ {
			if quest.RequiredItemId[i] == itemEntry && status.ItemCount[i] < quest.RequiredItemCount[i] {
				newCount := status.ItemCount[i] + uint16(count)
				if newCount > quest.RequiredItemCount[i] {
					newCount = quest.RequiredItemCount[i]
				}

				status.ItemCount[i] = newCount
				updated = true
			}
		}

		if updated && m.CanCompleteQuest(playerQuests, questID) {
			status.Status = QuestStatusComplete
			completedQuests = append(completedQuests, questID)
		}
	}

	return completedQuests
}

// OnExploration processes exploration credit for a quest.
func (m *Manager) OnExploration(playerQuests map[uint32]*QuestStatusData, questID uint32) bool {
	status, exists := playerQuests[questID]
	if !exists || status.Status != QuestStatusIncomplete {
		return false
	}

	quest := m.templates[questID]
	if quest == nil || !quest.HasSpecialFlag(QuestSpecialExplorationOrEvent) {
		return false
	}

	status.Explored = true

	if m.CanCompleteQuest(playerQuests, questID) {
		status.Status = QuestStatusComplete

		return true
	}

	return false
}

// OnKillCreditGO processes GameObject kill credit for a quest.
func (m *Manager) OnKillCreditGO(playerQuests map[uint32]*QuestStatusData, goEntry uint32) []uint32 {
	var completedQuests []uint32

	for questID, status := range playerQuests {
		if status.Status != QuestStatusIncomplete {
			continue
		}

		quest := m.templates[questID]
		if quest == nil {
			continue
		}

		updated := false

		for i := 0; i < 4; i++ {
			if quest.RequiredNpcOrGo[i] < 0 &&
				uint32(-quest.RequiredNpcOrGo[i]) == goEntry &&
				status.CreatureOrGOCount[i] < quest.RequiredNpcOrGoCount[i] {
				status.CreatureOrGOCount[i]++
				updated = true
			}
		}

		if updated && m.CanCompleteQuest(playerQuests, questID) {
			status.Status = QuestStatusComplete
			completedQuests = append(completedQuests, questID)
		}
	}

	return completedQuests
}

// OnPlayerKill processes PvP kill credit for a quest.
func (m *Manager) OnPlayerKill(playerQuests map[uint32]*QuestStatusData) []uint32 {
	var completedQuests []uint32

	for questID, status := range playerQuests {
		if status.Status != QuestStatusIncomplete {
			continue
		}

		quest := m.templates[questID]
		if quest == nil || quest.RequiredPlayerKills == 0 {
			continue
		}

		if status.PlayerCount < uint16(quest.RequiredPlayerKills) {
			status.PlayerCount++
		}

		if m.CanCompleteQuest(playerQuests, questID) {
			status.Status = QuestStatusComplete
			completedQuests = append(completedQuests, questID)
		}
	}

	return completedQuests
}

// AddQuest initializes quest tracking for a newly accepted quest.
func (m *Manager) AddQuest(playerQuests map[uint32]*QuestStatusData, questID uint32) bool {
	quest := m.templates[questID]
	if quest == nil {
		return false
	}

	if _, exists := playerQuests[questID]; exists {
		return false // already have this quest
	}

	status := &QuestStatusData{
		Status: QuestStatusIncomplete,
	}

	// Auto-complete quests
	if quest.IsAutoComplete() {
		status.Status = QuestStatusComplete
	}

	playerQuests[questID] = status

	return true
}

// RemoveQuest abandons a quest.
func (m *Manager) RemoveQuest(playerQuests map[uint32]*QuestStatusData, questID uint32) bool {
	if _, exists := playerQuests[questID]; !exists {
		return false
	}

	delete(playerQuests, questID)

	return true
}

// RewardQuest grants the rewards for a completed quest.
// Returns (xp, money, items given, error).
type RewardResult struct {
	XP              uint32
	Money           uint32
	RewardItems     [4]uint32
	RewardItemCount [4]uint16
}

func (m *Manager) RewardQuest(playerQuests map[uint32]*QuestStatusData, questID uint32, chosenRewardIndex int) *RewardResult {
	quest := m.templates[questID]
	if quest == nil {
		return nil
	}

	status, exists := playerQuests[questID]
	if !exists || status.Status != QuestStatusComplete {
		return nil
	}

	result := &RewardResult{
		XP:    quest.RewardXP,
		Money: uint32(quest.RewardMoney),
	}

	// Choice reward
	if chosenRewardIndex >= 0 && chosenRewardIndex < 6 {
		if quest.RewardChoiceItemId[chosenRewardIndex] > 0 {
			// Handled by caller (item granting)
		}
	}

	// Fixed rewards
	for i := 0; i < 4; i++ {
		if quest.RewardItemId[i] > 0 {
			result.RewardItems[i] = quest.RewardItemId[i]
			result.RewardItemCount[i] = quest.RewardItemCount[i]
		}
	}

	// Mark as rewarded
	status.Status = QuestStatusRewarded

	return result
}

// ComputeXPReward calculates XP for a quest based on player level.
// This is a simplified version; the real calculation uses QuestXP.dbc.
func ComputeXPReward(questLevel int32, playerLevel uint8) uint32 {
	if questLevel <= 0 {
		return 0
	}

	// Very rough XP formula matching WotLK scaling
	level := int32(playerLevel)
	if level <= 0 {
		level = 1
	}

	diff := questLevel - level
	if diff < -5 {
		return 0 // too low level
	}

	base := uint32(questLevel * questLevel * 10)

	return base
}

// GetPlayerRace returns the player's race from a GUID.
// This is a placeholder — in a real implementation, you'd look up the player.
func GetPlayerRace(guid wow.GUID) wow.PlayerRace {
	return wow.RaceHuman // default
}
