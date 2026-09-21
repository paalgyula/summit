package world

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// handleGroupChat routes party, raid, raid leader, raid warning,
// battleground, battleground leader, and party leader messages.
func (gc *WorldSession) handleGroupChat(chatType uint32, lang int32, message string) {
	server := gc.server()
	if server == nil {
		return
	}

	group := server.GetGroup(gc.player.GroupID)
	if group == nil {
		return
	}

	// Raid warnings can only be sent by the leader or an assistant.
	if chatType == ChatMsgRaidWarning {
		if !group.IsLeader(gc.player.GUID()) {
			return
		}
	}

	// Party/Raid leader messages can only be sent by the leader.
	if chatType == ChatMsgPartyLeader || chatType == ChatMsgRaidLeader {
		if !group.IsLeader(gc.player.GUID()) {
			return
		}
	}

	pkt := gc.buildChatPacket(chatType, lang, gc.player, message)

	for _, guid := range group.Members {
		if s := server.SessionByGUID(guid); s != nil {
			s.Send(pkt)
		}
	}
}

// handleAddonChat routes LANG_ADDON messages to the appropriate group/guild/whisper target.
func (gc *WorldSession) handleAddonChat(chatType uint32, lang int32, message string) {
	server := gc.server()
	if server == nil {
		return
	}

	switch chatType {
	case ChatMsgParty, ChatMsgPartyLeader:
		gc.handleGroupChat(chatType, lang, message)
	case ChatMsgRaid, ChatMsgRaidLeader, ChatMsgRaidWarning:
		gc.handleGroupChat(chatType, lang, message)
	case ChatMsgGuild, ChatMsgOfficer:
		gc.handleGuildChat(chatType, lang, message)
	case ChatMsgBattleGround, ChatMsgBattleGroundLeader:
		gc.handleGroupChat(chatType, lang, message)
	case ChatMsgWhisper:
		// addon whispers use the same path — already routed by caller
	}
}

// getGroupSessionGUIDs returns the member GUIDs of the group.
func getGroupMemberGUIDs(group *Group) []wow.GUID {
	guildIDs := make([]wow.GUID, 0, group.MemberCount())

	for _, guid := range group.Members {
		guildIDs = append(guildIDs, guid)
	}

	return guildIDs
}

// buildChatPacketStatic builds an SMSG_MESSAGECHAT packet without a session.
func buildChatPacketStatic(chatType uint32, lang int32, senderGUID wow.GUID, targetGUID wow.GUID, message string) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerMessagechat)

	_ = pkt.WriteOne(int(chatType))
	_ = pkt.Write(lang)
	_ = pkt.Write(senderGUID)
	_ = pkt.Write(uint32(0))                // chat flags
	_ = pkt.Write(targetGUID)               // target GUID
	_ = pkt.Write(uint32(len(message) + 1)) // message length incl. terminator
	pkt.WriteString(message)
	_ = pkt.WriteOne(0) // chat tag (none)

	return pkt
}
