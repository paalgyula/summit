package quest

import (
	"github.com/paalgyula/summit/pkg/store"
	"github.com/rs/zerolog/log"
)

// Manager holds all quest templates and quest giver relations in memory.
type Manager struct {
	templates         map[uint32]*Quest
	creatureQuests    map[uint32][]uint32 // creature_entry → quest IDs offered
	creatureInvolved  map[uint32][]uint32 // creature_entry → quest IDs completed
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
func (m *Manager) GetDialogStatus(playerQuests map[uint32]*QuestStatusData, creatureEntry uint32) QuestGiverStatus {
	maxStatus := DialogStatusNone

	// Check quests offered by this NPC
	for _, questID := range m.creatureQuests[creatureEntry] {
		quest := m.templates[questID]
		if quest == nil {
			continue
		}

		status := m.computeOfferStatus(playerQuests, quest)
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
func (m *Manager) computeOfferStatus(playerQuests map[uint32]*QuestStatusData, quest *Quest) QuestGiverStatus {
	if quest == nil {
		return DialogStatusNone
	}

	status, exists := playerQuests[quest.ID]

	switch {
	case exists && status.Status == QuestStatusComplete:
		return DialogStatusReward
	case exists && status.Status == QuestStatusIncomplete:
		return DialogStatusIncomplete
	case !exists:
		// Quest not in log — available
		if quest.IsRepeatable() {
			return DialogStatusAvailable
		}

		return DialogStatusAvailable
	default:
		return DialogStatusNone
	}
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
		ID:             t.ID,
		Method:         1,
		ZoneOrSort:     int32(t.QuestSortID),
		MinLevel:       uint32(t.MinLevel),
		Level:          int32(t.QuestLevel),
		Type:           uint32(t.QuestType),
		AllowableRaces: t.RequiredRaces,
		Flags:          QuestFlags(t.Flags),
		SpecialFlags:   QuestSpecialFlags(t.SpecialFlags),
		TimeAllowed:    t.TimeAllowed,
		StartItem:      t.StartItem,
		RequiredPlayerKills: t.RequiredPlayerKills,
		RewardMoney:    t.RewardMoney,
		RewardXP:       t.RewardXP,

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
