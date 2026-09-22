package world

import (
	"sort"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/quest"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

// Client quest log capacity and per-slot field stride, matching AzerothCore's
// MAX_QUEST_LOG_SIZE (25) and MAX_QUEST_OFFSET (5).
const (
	maxQuestLogSlots  = 25
	questLogFieldStep = 5
)

// questLogField returns the update field for a quest-log slot and offset.
// Slot offsets: 0 = quest id, 1 = state, 2..3 = packed objective counters,
// 4 = timer.
func questLogField(slot, offset int) object.UpdateField {
	return object.UpdateField(int(object.PlayerQuestLog1_1) + slot*questLogFieldStep + offset)
}

// addQuestToLog records the quest in a free client quest-log slot and pushes
// the PLAYER_QUEST_LOG_n_* update fields. Mirrors Player::SetQuestSlot.
func (gc *WorldSession) addQuestToLog(q *quest.Quest) {
	if q == nil {
		return
	}

	gc.addQuestIDToLog(q.ID)
}

// addQuestIDToLog records a quest id in a free client quest-log slot.
func (gc *WorldSession) addQuestIDToLog(questID uint32) {
	if gc.player == nil || gc.player.Object == nil || questID == 0 {
		return
	}

	for slot := 0; slot < maxQuestLogSlots; slot++ {
		if gc.questLog[slot] == questID {
			return // already logged
		}
	}

	slot := -1

	for i := 0; i < maxQuestLogSlots; i++ {
		if gc.questLog[i] == 0 {
			slot = i

			break
		}
	}

	if slot < 0 {
		// Quest log full: tell the client to drop it.
		gc.socket.Send(wow.NewPacket(wow.ServerQuestlogFull))

		return
	}

	gc.questLog[slot] = questID

	obj := gc.player.Object
	obj.SetUInt32Value(questLogField(slot, 0), questID) // quest id
	obj.SetUInt32Value(questLogField(slot, 1), 0)       // state
	obj.SetUInt32Value(questLogField(slot, 2), 0)       // objective counters 0-1
	obj.SetUInt32Value(questLogField(slot, 3), 0)       // objective counters 2-3
	obj.SetUInt32Value(questLogField(slot, 4), 0)       // timer
}

// rebuildQuestLog repopulates the client quest log from the player's persisted
// quest state (called on login). Quests are ordered by id for stable slots.
func (gc *WorldSession) rebuildQuestLog() {
	if gc.player == nil {
		return
	}

	status, ok := gc.player.QuestStatus.(map[uint32]*quest.QuestStatusData)
	if !ok || len(status) == 0 {
		return
	}

	ids := make([]uint32, 0, len(status))
	for id := range status {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		s := status[id]
		if s == nil || s.Status == quest.QuestStatusRewarded {
			continue // rewarded quests are not part of the active log
		}

		gc.addQuestIDToLog(id)
		gc.updateQuestLogCounters(id, s.CreatureOrGOCount)
	}
}

// saveQuests persists the character's quest progress to the character document
// with a single partial update (embedded quests, no extra collection).
func (gc *WorldSession) saveQuests() {
	if gc.player == nil {
		return
	}

	server, ok := gc.ws.(*Server)
	if !ok || server.charStore == nil {
		return
	}

	if err := server.charStore.UpdateCharacterQuests(gc.player); err != nil {
		gc.log.Error().Err(err).Msg("failed to persist quest progress")
	}
}

// updateQuestLogCounters writes the packed kill/use objective counters for a
// logged quest. Counters are packed two per uint32 (creature or GO counts).
func (gc *WorldSession) updateQuestLogCounters(questID uint32, creatureOrGO [4]uint16) {
	if gc.player == nil || gc.player.Object == nil {
		return
	}

	for slot := 0; slot < maxQuestLogSlots; slot++ {
		if gc.questLog[slot] != questID {
			continue
		}

		obj := gc.player.Object
		obj.SetUInt32Value(questLogField(slot, 2), uint32(creatureOrGO[0])|uint32(creatureOrGO[1])<<16)
		obj.SetUInt32Value(questLogField(slot, 3), uint32(creatureOrGO[2])|uint32(creatureOrGO[3])<<16)

		return
	}
}

// removeQuestFromLog clears the quest's log slot update fields.
func (gc *WorldSession) removeQuestFromLog(questID uint32) {
	if gc.player == nil || gc.player.Object == nil {
		return
	}

	for slot := 0; slot < maxQuestLogSlots; slot++ {
		if gc.questLog[slot] != questID {
			continue
		}

		gc.questLog[slot] = 0

		obj := gc.player.Object
		for offset := 0; offset < questLogFieldStep; offset++ {
			obj.SetUInt32Value(questLogField(slot, offset), 0)
		}

		return
	}
}

// HandleQuestQuery handles CMSG_QUEST_QUERY: the client asks for the full quest
// definition (e.g. for a quest log entry it does not have cached).
func (gc *WorldSession) HandleQuestQuery(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var questID uint32
	if err := reader.Read(&questID); err != nil {
		return
	}

	qm := gc.getQuestManager()
	if qm == nil {
		return
	}

	qDef := qm.GetQuest(questID)
	if qDef == nil {
		return
	}

	gc.socket.Send(quest.BuildQuestQueryResponse(qDef))

	log.Debug().Uint32("quest", questID).Msg("sent quest query response")
}
