package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/quest"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

// getNPCByGUID looks up an NPC from the world's spawn manager by GUID.
func (gc *WorldSession) getNPCByGUID(guid wow.GUID) *NPC {
	server, ok := gc.ws.(*Server)
	if !ok || server.spawns == nil {
		return nil
	}

	return server.spawns.GetNPC(uint64(guid.Counter()))
}

// getQuestManager returns the quest manager from the server.
func (gc *WorldSession) getQuestManager() *quest.Manager {
	server, ok := gc.ws.(*Server)
	if !ok {
		return nil
	}

	return server.questMgr
}

// getPlayerQuests returns the player's quest status map, initializing it if needed.
func (gc *WorldSession) getPlayerQuests() map[uint32]*quest.QuestStatusData {
	if gc.player == nil {
		return nil
	}

	if gc.player.QuestStatus == nil {
		gc.player.QuestStatus = make(map[uint32]*quest.QuestStatusData)
	}

	m, ok := gc.player.QuestStatus.(map[uint32]*quest.QuestStatusData)
	if !ok {
		m = make(map[uint32]*quest.QuestStatusData)
		gc.player.QuestStatus = m
	}

	return m
}

// setPlayerQuests stores the quest status map back to the player.
func (gc *WorldSession) setPlayerQuests(m map[uint32]*quest.QuestStatusData) {
	if gc.player != nil {
		gc.player.QuestStatus = m
	}
}

// HandleQuestgiverHello handles CMSG_QUESTGIVER_HELLO — right-click on an NPC.
func (gc *WorldSession) HandleQuestgiverHello(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var guid uint64

	_ = reader.Read(&guid)

	npc := gc.getNPCByGUID(wow.GUID(guid))
	if npc == nil {
		return
	}

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	// Check if NPC is a quest giver or quest completer
	if !qm.IsQuestGiver(npc.EntryID) && !qm.IsQuestCompleter(npc.EntryID) {
		greeting := fmt.Sprintf("Greetings, %s. How can I help you?", gc.player.Name)
		pkt := quest.BuildQuestGiverQuestList(wow.GUID(guid), greeting, nil)
		gc.socket.Send(pkt)

		return
	}

	// Build quest list
	playerQuests := gc.getPlayerQuests()

	// Get quests offered by this NPC
	offeredQuests := qm.GetQuestsForCreature(npc.EntryID)
	// Get quests completable at this NPC
	involvedQuests := qm.GetInvolvedQuestsForCreature(npc.EntryID)

	// Collect all relevant quests
	var questList []quest.QuestListItem

	// First: quests that can be turned in (involved/completable)
	for _, questID := range involvedQuests {
		status, exists := playerQuests[questID]
		if !exists {
			continue
		}

		if status.Status == quest.QuestStatusComplete || status.Status == quest.QuestStatusIncomplete {
			qDef := qm.GetQuest(questID)
			if qDef == nil {
				continue
			}

			questList = append(questList, quest.QuestListItem{
				QuestID:      questID,
				QuestIcon:    questIconForStatus(status.Status),
				QuestLevel:   qDef.Level,
				QuestFlags:   uint32(qDef.Flags),
				IsRepeatable: qDef.IsRepeatable(),
				Title:        qDef.Title,
			})
		}
	}

	// Then: quests available to accept
	for _, questID := range offeredQuests {
		// Skip if already in quest log
		if _, exists := playerQuests[questID]; exists {
			continue
		}

		qDef := qm.GetQuest(questID)
		if qDef == nil {
			continue
		}

		// TODO: check level, race, class requirements
		questList = append(questList, quest.QuestListItem{
			QuestID:      questID,
			QuestIcon:    0, // exclamation mark (!)
			QuestLevel:   qDef.Level,
			QuestFlags:   uint32(qDef.Flags),
			IsRepeatable: qDef.IsRepeatable(),
			Title:        qDef.Title,
		})
	}

	if len(questList) == 0 {
		greeting := fmt.Sprintf("Greetings, %s. I have no tasks for you right now.", gc.player.Name)
		pkt := quest.BuildQuestGiverQuestList(wow.GUID(guid), greeting, nil)
		gc.socket.Send(pkt)

		return
	}

	// Send quest list
	greeting := fmt.Sprintf("Greetings, %s. I have some tasks that might interest you.", gc.player.Name)
	pkt := quest.BuildQuestGiverQuestList(wow.GUID(guid), greeting, questList)
	gc.socket.Send(pkt)
}

// HandleGossipSelectOption handles CMSG_GOSSIP_SELECT_OPTION.
func (gc *WorldSession) HandleGossipSelectOption(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var guid uint64
	var menuID uint32
	var optionID uint32

	_ = reader.Read(&guid)
	_ = reader.Read(&menuID)
	_ = reader.Read(&optionID)

	gc.sendGossipComplete(wow.GUID(guid))
}

// HandleQuestgiverQueryQuest handles CMSG_QUESTGIVER_QUERY_QUEST — view quest details.
func (gc *WorldSession) HandleQuestgiverQueryQuest(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var npcGUID uint64
	var questID uint32

	_ = reader.Read(&npcGUID)
	_ = reader.Read(&questID)

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	qDef := qm.GetQuest(questID)
	if qDef == nil {
		return
	}

	// Auto-accept if the quest has that flag
	if qDef.IsAutoAccept() {
		gc.handleAcceptQuest(wow.GUID(npcGUID), questID)

		return
	}

	// Auto-complete quests skip to request items
	if qDef.IsAutoComplete() {
		canComplete := qm.CanCompleteQuest(gc.getPlayerQuests(), questID)
		pkt := quest.BuildQuestGiverRequestItems(wow.GUID(npcGUID), qDef, canComplete)
		gc.socket.Send(pkt)

		return
	}

	// Show quest details
	pkt := quest.BuildQuestGiverQuestDetails(wow.GUID(npcGUID), qDef, true)
	gc.socket.Send(pkt)
}

// HandleQuestgiverAcceptQuest handles CMSG_QUESTGIVER_ACCEPT_QUEST.
func (gc *WorldSession) HandleQuestgiverAcceptQuest(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var npcGUID uint64
	var questID uint32

	_ = reader.Read(&npcGUID)
	_ = reader.Read(&questID)

	gc.handleAcceptQuest(wow.GUID(npcGUID), questID)
}

func (gc *WorldSession) handleAcceptQuest(npcGUID wow.GUID, questID uint32) {
	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	qDef := qm.GetQuest(questID)
	if qDef == nil {
		return
	}

	playerQuests := gc.getPlayerQuests()

	// Try to add the quest
	if !qm.AddQuest(playerQuests, questID) {
		// Failed — send invalid
		pkt := quest.BuildQuestGiverQuestInvalid(0) // 0 = OK actually failed
		gc.socket.Send(pkt)

		return
	}

	// Send close gossip
	gc.sendGossipComplete(npcGUID)

	// TODO: cast source spell if applicable
	// TODO: send party share if QUEST_FLAGS_PARTY_ACCEPT

	log.Debug().
		Uint32("quest", questID).
		Str("title", qDef.Title).
		Msg("quest accepted")
}

// HandleQuestgiverCompleteQuest handles CMSG_QUESTGIVER_COMPLETE_QUEST — turn-in.
func (gc *WorldSession) HandleQuestgiverCompleteQuest(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var npcGUID uint64
	var questID uint32

	_ = reader.Read(&npcGUID)
	_ = reader.Read(&questID)

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	qDef := qm.GetQuest(questID)
	if qDef == nil {
		return
	}

	playerQuests := gc.getPlayerQuests()
	status, exists := playerQuests[questID]
	if !exists {
		return
	}

	// Check if quest has required items
	hasRequiredItems := false
	for i := 0; i < 6; i++ {
		if qDef.RequiredItemId[i] > 0 {
			hasRequiredItems = true

			break
		}
	}

	canComplete := status.Status == quest.QuestStatusComplete || qDef.CanTurnInWhileIncomplete()

	if hasRequiredItems {
		// Send request items
		pkt := quest.BuildQuestGiverRequestItems(wow.GUID(npcGUID), qDef, canComplete)
		gc.socket.Send(pkt)
	} else {
		// Send offer reward
		pkt := quest.BuildQuestGiverOfferReward(wow.GUID(npcGUID), qDef, qDef.IsAutoComplete())
		gc.socket.Send(pkt)
	}
}

// HandleQuestgiverRequestReward handles CMSG_QUESTGIVER_REQUEST_REWARD.
func (gc *WorldSession) HandleQuestgiverRequestReward(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var npcGUID uint64
	var questID uint32

	_ = reader.Read(&npcGUID)
	_ = reader.Read(&questID)

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	qDef := qm.GetQuest(questID)
	if qDef == nil {
		return
	}

	// Send offer reward
	pkt := quest.BuildQuestGiverOfferReward(wow.GUID(npcGUID), qDef, false)
	gc.socket.Send(pkt)
}

// HandleQuestgiverChooseReward handles CMSG_QUESTGIVER_CHOOSE_REWARD.
func (gc *WorldSession) HandleQuestgiverChooseReward(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var npcGUID uint64
	var questID uint32
	var chosenItem uint32

	_ = reader.Read(&npcGUID)
	_ = reader.Read(&questID)
	_ = reader.Read(&chosenItem)

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	playerQuests := gc.getPlayerQuests()
	result := qm.RewardQuest(playerQuests, questID, int(chosenItem))
	if result == nil {
		return
	}

	// Grant XP
	if result.XP > 0 {
		gc.player.XP += result.XP
		// TODO: check level up
	}

	// Grant money
	if result.Money > 0 {
		gc.player.Money += result.Money
	}

	// Grant fixed reward items
	for i := 0; i < 4; i++ {
		if result.RewardItems[i] > 0 {
			// TODO: add item to inventory
			log.Debug().
				Uint32("item", result.RewardItems[i]).
				Uint16("count", result.RewardItemCount[i]).
				Msg("quest reward item given")
		}
	}

	// Send completion
	pkt := quest.BuildQuestGiverQuestComplete(questID, result.XP, result.Money, 0, 0)
	gc.socket.Send(pkt)

	// Close gossip
	gc.sendGossipComplete(wow.GUID(npcGUID))

	log.Debug().
		Uint32("quest", questID).
		Uint32("xp", result.XP).
		Uint32("money", result.Money).
		Msg("quest rewarded")
}

// HandleQuestgiverCancel handles CMSG_QUESTGIVER_CANCEL.
func (gc *WorldSession) HandleQuestgiverCancel(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var npcGUID uint64
	_ = reader.Read(&npcGUID)

	// Close the gossip window
	gc.sendGossipComplete(wow.GUID(npcGUID))
}

// HandleQuestgiverStatusQuery handles CMSG_QUESTGIVER_STATUS_QUERY.
func (gc *WorldSession) HandleQuestgiverStatusQuery(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var guid uint64
	_ = reader.Read(&guid)

	npc := gc.getNPCByGUID(wow.GUID(guid))
	if npc == nil {
		return
	}

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	playerQuests := gc.getPlayerQuests()
	status := qm.GetDialogStatus(playerQuests, npc.EntryID)

	pkt := quest.BuildQuestGiverStatus(wow.GUID(guid), status)
	gc.socket.Send(pkt)
}

// HandleQuestLogRemoveQuest handles CMSG_QUESTLOG_REMOVE_QUEST — abandon quest.
func (gc *WorldSession) HandleQuestLogRemoveQuest(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var questID uint32
	_ = reader.Read(&questID)

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	playerQuests := gc.getPlayerQuests()
	qm.RemoveQuest(playerQuests, questID)

	log.Debug().Uint32("quest", questID).Msg("quest abandoned")
}

// sendGossipComplete sends SMSG_GOSSIP_COMPLETE to close the gossip window.
func (gc *WorldSession) sendGossipComplete(guid wow.GUID) {
	pkt := wow.NewPacket(wow.ServerGossipComplete)
	gc.socket.Send(pkt)
}

// questIconForStatus returns the quest icon for a given status.
func questIconForStatus(status quest.QuestStatus) uint32 {
	switch status {
	case quest.QuestStatusComplete:
		return 1 // yellow ? (turn-in)
	case quest.QuestStatusIncomplete:
		return 1 // yellow ? (in progress)
	default:
		return 0 // exclamation mark (!)
	}
}
