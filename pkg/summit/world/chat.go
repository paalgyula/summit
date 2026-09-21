package world

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Chat message types (ChatMsg in 3.3.5a; CHAT_MSG_ADDON is 0xFFFFFFFF).
const (
	ChatMsgSay                = 0x01
	ChatMsgParty              = 0x02
	ChatMsgRaid               = 0x03
	ChatMsgGuild              = 0x04
	ChatMsgOfficer            = 0x05
	ChatMsgYell               = 0x06
	ChatMsgWhisper            = 0x07
	ChatMsgEmote              = 0x0A
	ChatMsgSystem             = 0x10
	ChatMsgChannel            = 0x0E
	ChatMsgPartyLeader        = 0x0F
	ChatMsgRaidLeader         = 0x11
	ChatMsgRaidWarning        = 0x12
	ChatMsgBattleGround       = 0x13
	ChatMsgBattleGroundLeader = 0x14
	ChatMsgAddon              = 0xFF
)

// Language constants.
const (
	LangUniversal int32 = 0
	LangAddon     int32 = -1
)

// server returns the Server this session belongs to, or nil.
func (gc *WorldSession) server() *Server {
	s, _ := gc.ws.(*Server)

	return s
}

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

	// Validate chat type
	if !IsValidChatType(chatType) {
		gc.log.Debug().Uint32("type", chatType).Msg("invalid chat type")

		return
	}

	isAddon := lang == LangAddon

	// LANG_ADDON is only valid for specific chat types (AC validation)
	if isAddon && !IsValidAddonChatType(chatType) {
		gc.log.Debug().Uint32("type", chatType).Msg("LANG_ADDON with invalid chat type combination")

		return
	}

	// Sanitize non-addon messages
	if !isAddon {
		cleaned, ok := SanitizeChatMessage(message)
		if !ok {
			return
		}

		message = cleaned
	}

	// Message length check (AC: msg.length() > 255)
	if len(message) > chatMaxMsgLen {
		return
	}

	// Flood throttle (skip for system messages, addon, AFK, DND)
	if chatType != ChatMsgSystem && !isAddon && chatType != 0x15 && chatType != 0x16 {
		if !CheckFlood(gc.player, false) {
			return
		}
	}

	// Dead players can't chat (except whisper, addon, AFK, DND)
	if gc.player.IsDead() && IsAliveChatType(chatType) {
		return
	}

	// GM silence aura check (aura 1852) — blocks all chat except whisper
	if HasGMSilenceAura(gc.player) && chatType != ChatMsgWhisper {
		return
	}

	// Say/Yell/Emote level requirement
	if IsAliveChatType(chatType) && !CanPlayerSay(gc.player, gc.server().GetChatConfig().SayLevelReq) {
		return
	}

	// Mute check — if player is muted, they can't send non-addon messages
	if !isAddon && gc.player.IsGhost {
		// Ghost players can only whisper
		if chatType != ChatMsgWhisper {
			return
		}
	}

	// Route to the appropriate handler
	// First, check if this is a GM command (starts with ".")
	if result := gc.parseAndExecuteCommand(message); result != CommandNotCommand {
		return
	}

	switch chatType {
	case ChatMsgSay:
		gc.handleSay(lang, message)
	case ChatMsgYell:
		gc.handleYell(lang, message)
	case ChatMsgWhisper:
		gc.handleWhisper(reader, lang, message)
	case ChatMsgEmote:
		gc.handleEmote(lang, message)

	// Group chat
	case ChatMsgParty, ChatMsgPartyLeader, ChatMsgRaid,
		ChatMsgRaidLeader, ChatMsgRaidWarning,
		ChatMsgBattleGround, ChatMsgBattleGroundLeader:
		if isAddon {
			gc.handleAddonChat(chatType, lang, message)
		} else {
			gc.handleGroupChat(chatType, lang, message)
		}

	// Channel chat
	case ChatMsgChannel:
		if isAddon {
			gc.handleAddonChat(chatType, lang, message)
		} else {
			gc.handleChannelChat(reader, lang, message)
		}

	// Guild chat
	case ChatMsgGuild, ChatMsgOfficer:
		if isAddon {
			gc.handleAddonChat(chatType, lang, message)
		} else {
			gc.handleGuildChat(chatType, lang, message)
		}

	// Addon traffic over whisper
	case ChatMsgAddon:
		gc.handleAddonChat(ChatMsgWhisper, lang, message)

	default:
		gc.log.Debug().Uint32("type", chatType).Str("msg", message).Msg("unhandled chat type")
	}
}

// handleSay sends a SAY message to nearby players.
func (gc *WorldSession) handleSay(lang int32, message string) {
	server := gc.server()
	if server == nil {
		return
	}

	pkt := gc.buildChatPacket(ChatMsgSay, lang, gc.player, message)

	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.Send(pkt)
		}
	}

	// Echo back to sender
	gc.Send(pkt)
}

// handleYell sends a YELL message to all players on the same map.
func (gc *WorldSession) handleYell(lang int32, message string) {
	server := gc.server()
	if server == nil {
		return
	}

	pkt := gc.buildChatPacket(ChatMsgYell, lang, gc.player, message)

	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.Send(pkt)
		}
	}

	// Echo back to sender
	gc.Send(pkt)
}

// handleWhisper sends a private message to a specific player.
func (gc *WorldSession) handleWhisper(reader *wow.PacketReader, lang int32, message string) {
	var targetGUID uint64
	_ = reader.Read(&targetGUID)

	target := wow.GUID(targetGUID)

	server := gc.server()
	if server == nil {
		return
	}

	// Whisper level requirement
	if !CanPlayerWhisper(gc.player, server.GetChatConfig().WhisperLevelReq) {
		return
	}

	other := server.SessionByGUID(target)
	if other == nil || other.player == nil {
		gc.log.Debug().Str("target", fmt.Sprintf("0x%x", uint64(target))).Msg("whisper target not found")

		return
	}

	// Cross-faction check: if disabled, block whispers between Alliance and Horde
	cfg := server.GetChatConfig()
	if !cfg.CrossFactionChat && !SameFaction(gc.player.Race, other.player.Race) {
		return
	}

	pkt := gc.buildChatPacket(ChatMsgWhisper, lang, gc.player, message)
	other.Send(pkt)
	// Confirm to sender
	gc.Send(pkt)
}

// handleEmote sends an EMOTE message to nearby players.
func (gc *WorldSession) handleEmote(lang int32, message string) {
	server := gc.server()
	if server == nil {
		return
	}

	pkt := gc.buildChatPacket(ChatMsgEmote, lang, gc.player, message)

	for _, other := range server.GetOtherSessions(gc) {
		if other.player != nil && other.player.IsInWorld {
			other.Send(pkt)
		}
	}

	// Echo back to sender
	gc.Send(pkt)
}

// buildChatPacket builds and returns SMSG_MESSAGECHAT for a player sender.
func (gc *WorldSession) buildChatPacket(chatType uint32, lang int32, sender *player.Player, message string) *wow.Packet {
	return buildChatPacketStatic(chatType, lang, sender.GUID(), sender.GUID(), message)
}

// sendMessageChat builds and sends SMSG_MESSAGECHAT to the client.
func (gc *WorldSession) sendMessageChat(chatType uint32, lang int32, sender *player.Player, message string) {
	gc.Send(gc.buildChatPacket(chatType, lang, sender, message))
}
