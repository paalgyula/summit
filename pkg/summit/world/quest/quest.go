package quest

import "github.com/paalgyula/summit/pkg/wow"

// Quest status values matching the client's expectations.
type QuestStatus uint32

const (
	QuestStatusNone        QuestStatus = 0
	QuestStatusUnavailable QuestStatus = 1
	QuestStatusComplete    QuestStatus = 2
	QuestStatusIncomplete  QuestStatus = 3
	QuestStatusRewarded    QuestStatus = 4
	QuestStatusFailed      QuestStatus = 5
)

// Quest flags from QuestDef.h.
type QuestFlags uint32

const (
	QuestFlagsStayAlive     QuestFlags = 0x00000001
	QuestFlagsPartyAccept   QuestFlags = 0x00000002
	QuestFlagsExploration   QuestFlags = 0x00000004
	QuestFlagsSharable      QuestFlags = 0x00000008
	QuestFlagsDaily         QuestFlags = 0x00001000
	QuestFlagsWeekly        QuestFlags = 0x00008000
	QuestFlagsAutoComplete  QuestFlags = 0x00010000
	QuestFlagsAutoAccept    QuestFlags = 0x00080000
)

// Quest special flags (quest_template_addon.SpecialFlags).
type QuestSpecialFlags uint32

const (
	QuestSpecialRepeatable         QuestSpecialFlags = 0x01
	QuestSpecialExplorationOrEvent QuestSpecialFlags = 0x02
	QuestSpecialAutoAccept         QuestSpecialFlags = 0x04
	QuestSpecialDFQuest            QuestSpecialFlags = 0x08
	QuestSpecialMonthly            QuestSpecialFlags = 0x10
	QuestSpecialCast               QuestSpecialFlags = 0x20
)

// QuestGiverStatus controls the floating icon above quest giver NPCs.
type QuestGiverStatus uint32

const (
	DialogStatusNone                 QuestGiverStatus = 0
	DialogStatusUnavailable          QuestGiverStatus = 1
	DialogStatusLowLevelAvailable    QuestGiverStatus = 2
	DialogStatusLowLevelRewardRep    QuestGiverStatus = 3
	DialogStatusLowLevelAvailableRep QuestGiverStatus = 4
	DialogStatusIncomplete           QuestGiverStatus = 5
	DialogStatusRewardRep            QuestGiverStatus = 6
	DialogStatusAvailable            QuestGiverStatus = 8
	DialogStatusReward               QuestGiverStatus = 10
)

// NPC flags (UnitNPCFlags).
const (
	NpcFlagNone         uint32 = 0x00000000
	NpcFlagGossip       uint32 = 0x00000001
	NpcFlagQuestGiver   uint32 = 0x00000002
	NpcFlagTrainer      uint32 = 0x00000010
	NpcFlagVendor       uint32 = 0x00000080
	NpcFlagRepair       uint32 = 0x00001000
	NpcFlagFlightMaster uint32 = 0x00002000
	NpcFlagInnkeeper    uint32 = 0x00010000
	NpcFlagBanker       uint32 = 0x00020000
	NpcFlagAuctioneer   uint32 = 0x00200000
)

// Quest is the in-memory representation of a quest template.
type Quest struct {
	ID             uint32
	Method         uint32
	ZoneOrSort     int32
	MinLevel       uint32
	Level          int32
	Type           uint32
	AllowableRaces uint32
	Flags          QuestFlags
	SpecialFlags   QuestSpecialFlags
	TimeAllowed    uint32
	StartItem      uint32
	RequiredPlayerKills uint32
	RewardMoney    int32
	RewardXP       uint32

	RequiredItemId     [6]uint32
	RequiredItemCount  [6]uint16
	RequiredNpcOrGo    [4]int32 // >0 = creature entry, <0 = GO entry (abs)
	RequiredNpcOrGoCount [4]uint16

	RewardChoiceItemId    [6]uint32
	RewardChoiceItemCount [6]uint16
	RewardItemId          [4]uint32
	RewardItemCount       [4]uint16

	RewardFactionId    [5]uint32
	RewardFactionValue [5]int32

	Title              string
	Description        string
	Objectives         string
	AreaDescription    string
	QuestCompletionLog string
	ObjectiveText      [4]string

	// Addon fields (quest_template_addon)
	PrevQuestId          int32
	NextQuestId          int32
	ExclusiveGroup       int32
	BreadcrumbForQuestId uint32
	RequiredSkillId      uint16
	RequiredSkillPoints  uint16
}

// HasFlag returns true if the quest has the given flag.
func (q *Quest) HasFlag(f QuestFlags) bool { return q.Flags&f != 0 }

// HasSpecialFlag returns true if the quest has the given special flag.
func (q *Quest) HasSpecialFlag(f QuestSpecialFlags) bool { return q.SpecialFlags&f != 0 }

// IsAutoComplete returns true if the quest completes automatically.
func (q *Quest) IsAutoComplete() bool { return q.HasFlag(QuestFlagsAutoComplete) }

// IsAutoAccept returns true if the quest is accepted automatically.
func (q *Quest) IsAutoAccept() bool {
	return q.HasFlag(QuestFlagsAutoAccept) || q.HasSpecialFlag(QuestSpecialAutoAccept)
}

// IsRepeatable returns true if the quest can be repeated.
func (q *Quest) IsRepeatable() bool { return q.HasSpecialFlag(QuestSpecialRepeatable) }

// CanTurnInWhileIncomplete returns true if the quest can be turned in even
// when objectives are not fully met (e.g. escort quests).
func (q *Quest) CanTurnInWhileIncomplete() bool { return q.HasFlag(QuestFlagsExploration) }

// QuestStatusData tracks per-quest progress on a player.
type QuestStatusData struct {
	Status            QuestStatus
	Timer             uint32
	ItemCount         [6]uint16
	CreatureOrGOCount [4]uint16
	PlayerCount       uint16
	Explored          bool
}

// Completed returns true if all kill/collection objectives are met.
func (q *QuestStatusData) Completed(qDef *Quest) bool {
	for i := 0; i < 4; i++ {
		if qDef.RequiredNpcOrGoCount[i] > 0 && q.CreatureOrGOCount[i] < qDef.RequiredNpcOrGoCount[i] {
			return false
		}
	}

	for i := 0; i < 6; i++ {
		if qDef.RequiredItemCount[i] > 0 && q.ItemCount[i] < qDef.RequiredItemCount[i] {
			return false
		}
	}

	return true
}

// IsQuestGiverNPC returns true if the NPC has the quest giver flag.
func IsQuestGiverNPC(npcFlags uint32) bool {
	return npcFlags&NpcFlagQuestGiver != 0
}

// GetDisplayIDForRace returns a default display ID for a race/gender combo.
// This is a simplified version; the real data comes from CreatureModelInfo.dbc.
func GetDisplayIDForRace(race wow.PlayerRace, gender wow.PlayerGender) uint32 {
	// Human male/female defaults
	if race == wow.RaceHuman {
		if gender == wow.GenderMale {
			return 49
		}

		return 50
	}

	return 49 // fallback
}
