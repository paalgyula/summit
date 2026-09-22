package quest

import (
	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

// Manager holds all quest templates and quest giver relations in memory.
type Manager struct {
	templates        map[uint32]*Quest
	creatureQuests   map[uint32][]uint32 // creature_entry → quest IDs offered
	creatureInvolved map[uint32][]uint32 // creature_entry → quest IDs completed
}

// NewManager creates a new QuestManager and loads data from the world store.
func NewManager(worldRepo store.WorldRepo) *Manager {
	m := &Manager{
		templates:        make(map[uint32]*Quest),
		creatureQuests:   make(map[uint32][]uint32),
		creatureInvolved: make(map[uint32][]uint32),
	}

	if worldRepo != nil {
		m.loadFromStore(worldRepo)
	}

	return m
}

func (m *Manager) loadFromStore(worldRepo store.WorldRepo) {
	// Load quest templates
	templates, err := worldRepo.GetQuestTemplates()
	if err != nil {
		log.Warn().Err(err).Msg("failed to load quest templates from store")
	} else {
		for id, t := range templates {
			m.templates[id] = storeTemplateToQuest(t)
		}

		log.Info().Int("count", len(templates)).Msg("loaded quest templates from store")
	}

	// Load all creature quest relations in bulk (2 queries instead of 60K)
	rel, err := worldRepo.GetAllCreatureQuestRelations()
	if err != nil {
		log.Warn().Err(err).Msg("failed to load creature quest starter relations")
	} else {
		m.creatureQuests = rel
	}

	invRel, err := worldRepo.GetAllCreatureQuestInvolvedRelations()
	if err != nil {
		log.Warn().Err(err).Msg("failed to load creature quest ender relations")
	} else {
		m.creatureInvolved = invRel
	}

	log.Info().
		Int("quest_givers", len(m.creatureQuests)).
		Int("quest_completers", len(m.creatureInvolved)).
		Msg("loaded creature quest relations")
}

// PlayerInfo carries the player attributes needed to decide whether a quest is
// available to them.
type PlayerInfo struct {
	Level    uint32
	Race     wow.PlayerRace
	Class    wow.PlayerClass
	Rewarded map[uint32]bool
}

// CanTakeQuest reports whether the player may accept the given quest, mirroring
// the relevant parts of AzerothCore's Player::CanTakeQuest / CanAddQuest:
// not already in the log, not already rewarded (unless repeatable), level,
// race, class, previous/next quest in chain.
func (m *Manager) CanTakeQuest(info PlayerInfo, playerQuests map[uint32]*QuestStatusData, quest *Quest) bool {
	if quest == nil {
		return false
	}

	if _, exists := playerQuests[quest.ID]; exists {
		return false
	}

	if !quest.IsRepeatable() && info.Rewarded != nil && info.Rewarded[quest.ID] {
		return false
	}

	if quest.MinLevel > 0 && info.Level < uint32(quest.MinLevel) {
		return false
	}

	if quest.AllowableRaces != 0 && !raceAllowed(quest.AllowableRaces, info.Race) {
		return false
	}

	if quest.RequiredClasses != 0 && !classAllowed(quest.RequiredClasses, info.Class) {
		return false
	}

	switch {
	case quest.PrevQuestId > 0:
		if !questDone(info, playerQuests, uint32(quest.PrevQuestId)) {
			return false
		}
	case quest.PrevQuestId < 0:
		if questDone(info, playerQuests, uint32(-quest.PrevQuestId)) {
			return false
		}
	}

	// If the follow-up quest of this chain was already rewarded, this quest is
	// redundant and should not be offered again.
	if quest.NextQuestId > 0 && info.Rewarded != nil && info.Rewarded[uint32(quest.NextQuestId)] {
		return false
	}

	return true
}

// AvailableQuestsForCreature returns the quests offered by a creature that the
// player can currently take, in relation order.
func (m *Manager) AvailableQuestsForCreature(info PlayerInfo, playerQuests map[uint32]*QuestStatusData, creatureEntry uint32) []*Quest {
	var result []*Quest

	for _, questID := range m.creatureQuests[creatureEntry] {
		quest := m.templates[questID]
		if quest == nil {
			continue
		}

		if m.CanTakeQuest(info, playerQuests, quest) {
			result = append(result, quest)
		}
	}

	return result
}

// raceAllowed reports whether the race bit is set in the packed allowable-race
// mask (bit 0 = Human, ...).
func raceAllowed(mask uint32, race wow.PlayerRace) bool {
	if race == 0 {
		return true
	}

	return mask&(1<<(uint32(race)-1)) != 0
}

// classAllowed reports whether the class bit is set in the packed required-class
// mask (bit 0 = Warrior, ...).
func classAllowed(mask uint32, class wow.PlayerClass) bool {
	if class == 0 {
		return true
	}

	return mask&(1<<(uint32(class)-1)) != 0
}

// questDone reports whether the quest has been rewarded or completed.
func questDone(info PlayerInfo, playerQuests map[uint32]*QuestStatusData, questID uint32) bool {
	if info.Rewarded != nil && info.Rewarded[questID] {
		return true
	}

	if status, ok := playerQuests[questID]; ok && status != nil {
		return status.Status == QuestStatusComplete || status.Status == QuestStatusRewarded
	}

	return false
}

// GetQuest returns a quest template by ID, or nil if not found.
func (m *Manager) GetQuest(id uint32) *Quest {
	return m.templates[id]
}

// GetQuestsForCreature returns all quest IDs offered by a creature entry.
func (m *Manager) GetQuestsForCreature(creatureEntry uint32) []uint32 {
	return m.creatureQuests[creatureEntry]
}

// GetInvolvedQuestsForCreature returns all quest IDs completed by a creature entry.
func (m *Manager) GetInvolvedQuestsForCreature(creatureEntry uint32) []uint32 {
	return m.creatureInvolved[creatureEntry]
}

// IsQuestGiver returns true if the creature entry offers any quests.
func (m *Manager) IsQuestGiver(creatureEntry uint32) bool {
	return len(m.creatureQuests[creatureEntry]) > 0
}

// IsQuestCompleter returns true if the creature entry can complete any quests.
func (m *Manager) IsQuestCompleter(creatureEntry uint32) bool {
	return len(m.creatureInvolved[creatureEntry]) > 0
}

// GetDialogStatus computes the quest giver status icon for a creature relative
// to a player's quest state. Returns the highest-priority status.
func (m *Manager) GetDialogStatus(info PlayerInfo, playerQuests map[uint32]*QuestStatusData, creatureEntry uint32) QuestGiverStatus {
	maxStatus := DialogStatusNone

	// Check quests offered by this NPC
	for _, questID := range m.creatureQuests[creatureEntry] {
		quest := m.templates[questID]
		if quest == nil {
			continue
		}

		status := m.computeOfferStatus(info, playerQuests, quest)
		if status > maxStatus {
			maxStatus = status
		}
	}

	// Check quests completable at this NPC
	for _, questID := range m.creatureInvolved[creatureEntry] {
		quest := m.templates[questID]
		if quest == nil {
			continue
		}

		status := m.computeTurnInStatus(playerQuests, quest)
		if status > maxStatus {
			maxStatus = status
		}
	}

	return maxStatus
}

// computeOfferStatus determines what status icon to show for a quest this NPC offers.
func (m *Manager) computeOfferStatus(info PlayerInfo, playerQuests map[uint32]*QuestStatusData, quest *Quest) QuestGiverStatus {
	if quest == nil {
		return DialogStatusNone
	}

	// Quests already in the log are handled by the turn-in path below.
	if !m.CanTakeQuest(info, playerQuests, quest) {
		return DialogStatusNone
	}

	if quest.Level > 0 && int32(info.Level) < quest.Level {
		return DialogStatusLowLevelAvailable
	}

	return DialogStatusAvailable
}

// computeTurnInStatus determines what status icon to show for a quest this NPC completes.
func (m *Manager) computeTurnInStatus(playerQuests map[uint32]*QuestStatusData, quest *Quest) QuestGiverStatus {
	if quest == nil {
		return DialogStatusNone
	}

	status, exists := playerQuests[quest.ID]
	if !exists {
		return DialogStatusNone
	}

	switch status.Status {
	case QuestStatusComplete:
		return DialogStatusReward
	case QuestStatusIncomplete:
		if quest.CanTurnInWhileIncomplete() {
			return DialogStatusIncomplete
		}

		return DialogStatusNone
	default:
		return DialogStatusNone
	}
}

// storeTemplateToQuest converts a store.QuestTemplate to the runtime Quest struct.
func storeTemplateToQuest(t *store.QuestTemplate) *Quest {
	return &Quest{
		ID:                  t.ID,
		Method:              1,
		ZoneOrSort:          int32(t.QuestSortID),
		MinLevel:            uint32(t.MinLevel),
		Level:               int32(t.QuestLevel),
		Type:                uint32(t.QuestType),
		AllowableRaces:      t.RequiredRaces,
		RequiredClasses:     t.RequiredClasses,
		Flags:               QuestFlags(t.Flags),
		SpecialFlags:        QuestSpecialFlags(t.SpecialFlags),
		TimeAllowed:         t.TimeAllowed,
		StartItem:           t.StartItem,
		RequiredPlayerKills: t.RequiredPlayerKills,
		RewardMoney:         t.RewardMoney,
		RewardXP:            t.RewardXP,
		RewardXPDifficulty:  t.RewardXPDifficulty,

		RequiredItemId:        t.RequiredItemId,
		RequiredItemCount:     t.RequiredItemCount,
		RequiredNpcOrGo:       t.RequiredNpcOrGo,
		RequiredNpcOrGoCount:  t.RequiredNpcOrGoCount,
		RewardItemId:          t.RewardItemId,
		RewardItemCount:       t.RewardItemCount,
		RewardChoiceItemId:    t.RewardChoiceItemId,
		RewardChoiceItemCount: t.RewardChoiceItemCount,
		RewardFactionId:       t.RewardFactionId,
		RewardFactionValue:    t.RewardFactionValue,

		Title:              t.Title,
		Description:        t.Description,
		Objectives:         t.Objectives,
		AreaDescription:    t.AreaDescription,
		QuestCompletionLog: t.QuestCompletionLog,
		ObjectiveText:      t.ObjectiveText,

		PrevQuestId:          t.PrevQuestId,
		NextQuestId:          t.NextQuestId,
		ExclusiveGroup:       t.ExclusiveGroup,
		BreadcrumbForQuestId: t.BreadcrumbForQuestId,
		RequiredSkillId:      t.RequiredSkillId,
		RequiredSkillPoints:  t.RequiredSkillPoints,
	}
}
