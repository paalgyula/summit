package world

import (
	"strings"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

const (
	chatFloodLimit  = 10
	chatFloodWindow = 6 * time.Second
	chatMaxMsgLen   = 255
	chatAddonMaxLen = 255
	gmSilenceAuraID = 1852
)

// IsNasty returns true if the byte is a control character that should be
// rejected in chat messages. Tabs are allowed.
func IsNasty(c byte) bool {
	return c != '\t' && c <= 0x1F
}

// SanitizeChatMessage validates and cleans a chat message.
// It trims at the first newline, rejects empty or control-character messages,
// and collapses multiple spaces. Returns the cleaned message and true if valid.
func SanitizeChatMessage(msg string) (string, bool) {
	// Trim at first newline / carriage return
	if idx := strings.IndexAny(msg, "\n\r"); idx == 0 {
		return "", false
	} else if idx > 0 {
		msg = msg[:idx]
	}

	if msg == "" {
		return "", false
	}

	// Reject nasty control characters
	for i := 0; i < len(msg); i++ {
		if IsNasty(msg[i]) {
			return "", false
		}
	}

	// Collapse multiple spaces into one
	msg = strings.Join(strings.Fields(msg), " ")

	// Reject messages containing "|0" (AC chat hack prevention)
	if strings.Contains(msg, "|0") {
		return "", false
	}

	return msg, true
}

// CheckFlood returns true if the player is allowed to send another chat message.
// It implements a simple sliding-window flood throttle.
func CheckFlood(p *player.Player, isAddon bool) bool {
	now := time.Now()

	if now.After(p.ChatFloodResetAt) {
		p.ChatFloodCount = 0
		p.ChatFloodResetAt = now.Add(chatFloodWindow)
	}

	p.ChatFloodCount++

	return p.ChatFloodCount <= chatFloodLimit
}

// ValidateAddonLanguage checks if the language flag indicates addon traffic.
func ValidateAddonLanguage(lang int32) bool {
	return lang == LangAddon
}

// IsValidChatType returns true if the chat type is one we handle.
func IsValidChatType(chatType uint32) bool {
	switch chatType {
	case ChatMsgSay, ChatMsgParty, ChatMsgRaid, ChatMsgGuild,
		ChatMsgOfficer, ChatMsgYell, ChatMsgWhisper, ChatMsgEmote,
		ChatMsgSystem, ChatMsgChannel, ChatMsgRaidLeader,
		ChatMsgRaidWarning, ChatMsgBattleGround, ChatMsgBattleGroundLeader,
		ChatMsgPartyLeader, ChatMsgAddon:
		return true
	default:
		return false
	}
}

// IsValidAddonChatType checks if LANG_ADDON is valid for this chat type.
// In AC, LANG_ADDON is only valid for party, raid, guild, battleground, and whisper.
func IsValidAddonChatType(chatType uint32) bool {
	switch chatType {
	case ChatMsgParty, ChatMsgPartyLeader,
		ChatMsgRaid, ChatMsgRaidLeader, ChatMsgRaidWarning,
		ChatMsgGuild, ChatMsgOfficer,
		ChatMsgBattleGround, ChatMsgBattleGroundLeader,
		ChatMsgWhisper:
		return true
	default:
		return false
	}
}

// IsAliveChatType returns true if dead players cannot use this chat type.
func IsAliveChatType(chatType uint32) bool {
	switch chatType {
	case ChatMsgSay, ChatMsgYell, ChatMsgEmote:
		return true
	default:
		return false
	}
}

// CanPlayerSay returns true if the player meets the level requirement for say/yell/emote.
// The level requirement comes from the server's ChatConfig.
func CanPlayerSay(p *player.Player, sayLevelReq int) bool {
	return int(p.Level) >= sayLevelReq
}

// CanPlayerWhisper returns true if the player meets the level requirement for whispering.
// The level requirement comes from the server's ChatConfig.
func CanPlayerWhisper(p *player.Player, whisperLevelReq int) bool {
	return int(p.Level) >= whisperLevelReq
}

// HasGMSilenceAura returns true if the player has the GM silence aura (1852).
func HasGMSilenceAura(p *player.Player) bool {
	// TODO: implement real aura check when aura system is complete
	// For now, always return false
	return false
}
