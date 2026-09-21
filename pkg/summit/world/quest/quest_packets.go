package quest

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// BuildQuestGiverStatus builds SMSG_QUESTGIVER_STATUS.
// Format: [ObjectGuid npcGUID] [uint8 questStatus]
func BuildQuestGiverStatus(guid wow.GUID, status QuestGiverStatus) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverStatus)
	_ = pkt.Write(guid)
	_ = pkt.WriteOne(int(status))

	return pkt
}

// QuestListItem is a single entry in the quest list sent to the client.
type QuestListItem struct {
	QuestID      uint32
	QuestIcon    uint32
	QuestLevel   int32
	QuestFlags   uint32
	IsRepeatable bool
	Title        string
}

// BuildQuestGiverQuestList builds SMSG_QUESTGIVER_QUEST_LIST.
func BuildQuestGiverQuestList(guid wow.GUID, greeting string, quests []QuestListItem) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverQuestList)
	_ = pkt.Write(guid)
	pkt.WriteString(greeting)
	_ = pkt.WriteUint32(0) // emote delay
	_ = pkt.WriteUint32(0) // emote type
	_ = pkt.WriteOne(len(quests))

	for _, q := range quests {
		_ = pkt.WriteUint32(int(q.QuestID))
		_ = pkt.WriteUint32(int(q.QuestIcon))
		_ = pkt.Write(q.QuestLevel)
		_ = pkt.WriteUint32(int(q.QuestFlags))

		if q.IsRepeatable {
			_ = pkt.WriteOne(1)
		} else {
			_ = pkt.WriteOne(0)
		}

		pkt.WriteString(q.Title)
	}

	return pkt
}

// BuildQuestGiverQuestDetails builds SMSG_QUESTGIVER_QUEST_DETAILS.
func BuildQuestGiverQuestDetails(npcGUID wow.GUID, quest *Quest, activateAccept bool) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverQuestDetails)
	_ = pkt.Write(npcGUID)
	_ = pkt.Write(wow.GUID(0)) // divider GUID
	_ = pkt.WriteUint32(int(quest.ID))

	pkt.WriteString(quest.Title)
	pkt.WriteString(quest.Description)
	pkt.WriteString(quest.Objectives)

	_ = pkt.WriteOne(boolToInt(activateAccept))
	_ = pkt.WriteUint32(int(quest.Flags))
	_ = pkt.WriteUint32(0) // suggested players

	// Choice items
	choiceCount := countChoiceItems(quest)
	_ = pkt.WriteUint32(int(choiceCount))

	for i := 0; i < 6; i++ {
		if quest.RewardChoiceItemId[i] > 0 {
			_ = pkt.WriteUint32(int(quest.RewardChoiceItemId[i]))
			_ = pkt.WriteUint32(int(quest.RewardChoiceItemCount[i]))
			_ = pkt.WriteUint32(0) // displayId
		}
	}

	// Fixed reward items
	rewardCount := countRewardItems(quest)
	_ = pkt.WriteUint32(int(rewardCount))

	for i := 0; i < 4; i++ {
		if quest.RewardItemId[i] > 0 {
			_ = pkt.WriteUint32(int(quest.RewardItemId[i]))
			_ = pkt.WriteUint32(int(quest.RewardItemCount[i]))
			_ = pkt.WriteUint32(0) // displayId
		}
	}

	_ = pkt.WriteUint32(int(quest.RewardMoney))
	_ = pkt.WriteUint32(int(quest.RewardXP))

	// Honor
	_ = pkt.WriteUint32(0)
	_ = pkt.Write(float32(0))

	// Spell reward
	_ = pkt.WriteUint32(0)
	_ = pkt.Write(int32(0))

	// Title
	_ = pkt.WriteUint32(0)
	_ = pkt.WriteUint32(0)
	_ = pkt.WriteUint32(0)

	_ = pkt.WriteUint32(0) // unk

	// Reputation rewards (5 slots)
	for i := 0; i < 5; i++ {
		_ = pkt.WriteUint32(int(quest.RewardFactionId[i]))
	}

	for i := 0; i < 5; i++ {
		_ = pkt.Write(int32(quest.RewardFactionValue[i]))
	}

	for i := 0; i < 5; i++ {
		_ = pkt.Write(int32(0))
	}

	// Emotes
	_ = pkt.WriteUint32(0)

	return pkt
}

// BuildQuestGiverRequestItems builds SMSG_QUESTGIVER_REQUEST_ITEMS.
func BuildQuestGiverRequestItems(npcGUID wow.GUID, quest *Quest, canComplete bool) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverRequestItems)
	_ = pkt.Write(npcGUID)
	_ = pkt.WriteUint32(int(quest.ID))

	pkt.WriteString(quest.Title)
	pkt.WriteString("") // request items text

	_ = pkt.WriteUint32(0) // completion emote

	// Required items
	for i := 0; i < 6; i++ {
		if quest.RequiredItemId[i] > 0 {
			_ = pkt.WriteUint32(int(quest.RequiredItemId[i]))
			_ = pkt.WriteUint32(int(quest.RequiredItemCount[i]))
			_ = pkt.WriteUint32(0) // displayId
		}
	}

	// Money required
	if quest.RewardMoney < 0 {
		_ = pkt.WriteUint32(int(-quest.RewardMoney))
	} else {
		_ = pkt.WriteUint32(0)
	}

	_ = pkt.WriteOne(boolToInt(canComplete))

	return pkt
}

// BuildQuestGiverOfferReward builds SMSG_QUESTGIVER_OFFER_REWARD.
func BuildQuestGiverOfferReward(npcGUID wow.GUID, quest *Quest, autoFinish bool) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverOfferReward)
	_ = pkt.Write(npcGUID)
	_ = pkt.WriteUint32(int(quest.ID))

	pkt.WriteString(quest.Title)
	pkt.WriteString("") // reward text

	_ = pkt.WriteOne(boolToInt(autoFinish))
	_ = pkt.WriteUint32(int(quest.Flags))
	_ = pkt.WriteUint32(0) // suggested players

	// Emotes
	_ = pkt.WriteUint32(0)

	// Choice items
	choiceCount := countChoiceItems(quest)
	_ = pkt.WriteUint32(int(choiceCount))

	for i := 0; i < 6; i++ {
		if quest.RewardChoiceItemId[i] > 0 {
			_ = pkt.WriteUint32(int(quest.RewardChoiceItemId[i]))
			_ = pkt.WriteUint32(int(quest.RewardChoiceItemCount[i]))
			_ = pkt.WriteUint32(0)
		}
	}

	// Fixed reward items
	rewardCount := countRewardItems(quest)
	_ = pkt.WriteUint32(int(rewardCount))

	for i := 0; i < 4; i++ {
		if quest.RewardItemId[i] > 0 {
			_ = pkt.WriteUint32(int(quest.RewardItemId[i]))
			_ = pkt.WriteUint32(int(quest.RewardItemCount[i]))
			_ = pkt.WriteUint32(0)
		}
	}

	_ = pkt.WriteUint32(int(quest.RewardMoney))
	_ = pkt.WriteUint32(int(quest.RewardXP))

	// Honor
	_ = pkt.WriteUint32(0)
	_ = pkt.Write(float32(0))

	// Spell reward
	_ = pkt.WriteUint32(0)
	_ = pkt.Write(int32(0))

	// Title
	_ = pkt.WriteUint32(0)
	_ = pkt.WriteUint32(0)
	_ = pkt.WriteUint32(0)

	_ = pkt.WriteUint32(0)

	// Reputation
	for i := 0; i < 5; i++ {
		_ = pkt.WriteUint32(int(quest.RewardFactionId[i]))
	}

	for i := 0; i < 5; i++ {
		_ = pkt.Write(int32(quest.RewardFactionValue[i]))
	}

	for i := 0; i < 5; i++ {
		_ = pkt.Write(int32(0))
	}

	return pkt
}

// BuildQuestGiverQuestInvalid builds SMSG_QUESTGIVER_QUEST_INVALID.
func BuildQuestGiverQuestInvalid(failureReason uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverQuestInvalid)
	_ = pkt.WriteUint32(int(failureReason))

	return pkt
}

// BuildQuestGiverQuestComplete builds SMSG_QUESTGIVER_QUEST_COMPLETE.
func BuildQuestGiverQuestComplete(questID, xp, money, honor, talents uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverQuestComplete)
	_ = pkt.WriteUint32(int(questID))
	_ = pkt.WriteUint32(int(xp))
	_ = pkt.WriteUint32(int(money))
	_ = pkt.WriteUint32(int(honor))
	_ = pkt.WriteUint32(int(talents))
	_ = pkt.WriteUint32(0) // arena points

	return pkt
}

// BuildQuestGiverQuestFailed builds SMSG_QUESTGIVER_QUEST_FAILED.
func BuildQuestGiverQuestFailed(questID uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestgiverQuestFailed)
	_ = pkt.WriteUint32(int(questID))

	return pkt
}

// BuildQuestUpdateAddKill builds SMSG_QUESTUPDATE_ADD_KILL.
func BuildQuestUpdateAddKill(questID, creatureEntry, current, required uint32, guid wow.GUID) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestupdateAddKill)
	_ = pkt.WriteUint32(int(questID))
	_ = pkt.WriteUint32(int(creatureEntry))
	_ = pkt.WriteUint32(int(current))
	_ = pkt.WriteUint32(int(required))
	_ = pkt.Write(guid)

	return pkt
}

// BuildQuestUpdateAddItem builds SMSG_QUESTUPDATE_ADD_ITEM.
func BuildQuestUpdateAddItem(questID, itemEntry, count uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestupdateAddItem)
	_ = pkt.WriteUint32(int(questID))
	_ = pkt.WriteUint32(int(itemEntry))
	_ = pkt.WriteUint32(int(count))

	return pkt
}

// BuildQuestUpdateComplete builds SMSG_QUESTUPDATE_COMPLETE.
func BuildQuestUpdateComplete(questID uint32) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerQuestupdateComplete)
	_ = pkt.WriteUint32(int(questID))

	return pkt
}

// --- helpers ---

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

func countChoiceItems(q *Quest) uint32 {
	count := uint32(0)
	for i := 0; i < 6; i++ {
		if q.RewardChoiceItemId[i] > 0 {
			count++
		}
	}

	return count
}

func countRewardItems(q *Quest) uint32 {
	count := uint32(0)
	for i := 0; i < 4; i++ {
		if q.RewardItemId[i] > 0 {
			count++
		}
	}

	return count
}
