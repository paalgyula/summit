package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Chat message types (from SMSG_MESSAGECHAT).
const (
	ChatMsgSay        = 0x00
	ChatMsgParty      = 0x01
	ChatMsgRaid       = 0x02
	ChatMsgGuild      = 0x03
	ChatMsgOfficer    = 0x04
	ChatMsgYell       = 0x05
	ChatMsgWhisper    = 0x06
	ChatMsgEmote      = 0x0D
	ChatMsgSystem     = 0x10
)

// HandleMessageChat handles CMSG_MESSAGECHAT from the client.
func (gc *WorldSession) HandleMessageChat(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var chatType uint32
	_ = reader.Read(&chatType)

	var lang int32
	_ = reader.Read(&lang)

	var message string
	_ = reader.ReadString(&message)

	switch chatType {
	case ChatMsgSay:
		gc.handleSay(lang, message)
	case ChatMsgYell:
		gc.handleYell(lang, message)
	case ChatMsgWhisper:
		gc.handleWhisper(reader, lang, message)
	case ChatMsgEmote:
		gc.handleEmote(lang, message)
	default:
		gc.log.Debug().Uint32("type", chatType).Str("msg", message).Msg("unhandled chat type")
	}
}

// handleSay sends a SAY message to nearby players.
func (gc *WorldSession) handleSay(lang int32, message string) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.sendMessageChat(ChatMsgSay, lang, gc.player, message)
		}
	}

	// Echo back to sender
	gc.sendMessageChat(ChatMsgSay, lang, gc.player, message)
}

// handleYell sends a YELL message to all players on the same map.
func (gc *WorldSession) handleYell(lang int32, message string) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.sendMessageChat(ChatMsgYell, lang, gc.player, message)
		}
	}

	// Echo back to sender
	gc.sendMessageChat(ChatMsgYell, lang, gc.player, message)
}

// handleWhisper sends a private message to a specific player.
func (gc *WorldSession) handleWhisper(reader *wow.PacketReader, lang int32, message string) {
	var targetGUID uint64
	_ = reader.Read(&targetGUID)

	target := wow.GUID(targetGUID)

	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Find the target session
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.GUID() == target {
			// Send to target
			other.sendMessageChat(ChatMsgWhisper, lang, gc.player, message)
			// Confirm to sender
			gc.sendMessageChat(ChatMsgWhisper, lang, gc.player, message)

			return
		}
	}

	// Target not found - send error
	gc.log.Debug().Str("target", fmt.Sprintf("0x%x", uint64(target))).Msg("whisper target not found")
}

// handleEmote sends an EMOTE message to nearby players.
func (gc *WorldSession) handleEmote(lang int32, message string) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.sendMessageChat(ChatMsgEmote, lang, gc.player, message)
		}
	}

	// Echo back to sender
	gc.sendMessageChat(ChatMsgEmote, lang, gc.player, message)
}

// sendMessageChat builds and sends SMSG_MESSAGECHAT to the client.
func (gc *WorldSession) sendMessageChat(chatType uint32, lang int32, sender *player.Player, message string) {
	pkt := wow.NewPacket(wow.ServerMessagechat)

	_ = pkt.Write(chatType)
	_ = pkt.Write(lang)
	_ = pkt.Write(sender.GUID())
	_ = pkt.Write(uint32(0)) // chat flags
	_ = pkt.Write(sender.GUID()) // channel name (for channel chat)
	_ = pkt.Write(uint32(0))     // msg prefix (Achievement, BossEmote, etc)
	pkt.WriteString(message)

	gc.socket.Send(pkt)
}
