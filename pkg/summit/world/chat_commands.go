package world

import (
	"fmt"
	"strings"

	"github.com/paalgyula/summit/pkg/wow"
)

// CommandResult represents the outcome of processing a command.
type CommandResult int

const (
	CommandHandled   CommandResult = 0 // command was processed, don't send as chat
	CommandNotCommand                // message was not a command
	CommandError                    // command failed
)

// parseAndExecuteCommand checks if a message is a command (starts with "." or "/")
// and executes it. Returns CommandHandled if the message was a command,
// or CommandNotCommand if it should be treated as regular chat.
func (gc *WorldSession) parseAndExecuteCommand(message string) CommandResult {
	if len(message) == 0 {
		return CommandNotCommand
	}

	// Only process commands starting with "." (AC convention)
	// "/" commands are client-side slash commands like /wave, /dance
	if message[0] != '.' {
		return CommandNotCommand
	}

	// Parse the command: ".command args"
	parts := strings.SplitN(message[2:], " ", 2) // skip "."
	if len(parts) == 0 {
		return CommandNotCommand
	}

	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	server := gc.server()
	if server == nil {
		return CommandError
	}

	switch cmd {
	case "announce", "ann":
		return gc.cmdAnnounce(args)
	case "mute":
		return gc.cmdMute(server, args)
	case "unmute":
		return gc.cmdUnmute(server, args)
	case "kick":
		return gc.cmdKick(server, args)
	case "whispers":
		return gc.cmdWhispers(args)
	case "who", "whois":
		return gc.cmdWho(server, args)
	case "gps":
		return gc.cmdGPS()
	default:
		gc.log.Debug().Str("cmd", cmd).Msg("unknown GM command")
		gc.sendSystemMessage("Unknown command: " + cmd)

		return CommandHandled
	}
}

// sendSystemMessage sends a SMSG_MESSAGECHAT system message to the player.
func (gc *WorldSession) sendSystemMessage(msg string) {
	pkt := buildChatPacketStatic(ChatMsgSystem, LangUniversal, wow.GUID(0), gc.player.GUID(), msg)
	gc.Send(pkt)
}

// sendServerAnnounce sends a server-wide announcement to all online players.
func (gc *WorldSession) sendServerAnnounce(msg string) {
	server := gc.server()
	if server == nil {
		return
	}

	pkt := buildChatPacketStatic(ChatMsgSystem, LangUniversal, wow.GUID(0), wow.GUID(0), msg)
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld {
			other.Send(pkt)
		}
	}
}

// --- Command implementations ---

// .announce <message> — server-wide announcement
func (gc *WorldSession) cmdAnnounce(args string) CommandResult {
	if args == "" {
		gc.sendSystemMessage("Usage: .announce <message>")

		return CommandHandled
	}

	gc.sendServerAnnounce("[GM] " + args)

	return CommandHandled
}

// .mute <player> — mute a player (placeholder — needs real mute system)
func (gc *WorldSession) cmdMute(server *Server, args string) CommandResult {
	if args == "" {
		gc.sendSystemMessage("Usage: .mute <player>")

		return CommandHandled
	}

	targetName := NormalizePlayerName(args)
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && strings.EqualFold(other.player.Name, targetName) {
			// TODO: set a mute flag on the player and check it in chat validation
			gc.sendSystemMessage(fmt.Sprintf("Muted player %s (placeholder)", targetName))

			return CommandHandled
		}
	}

	gc.sendSystemMessage("Player not found: " + targetName)

	return CommandHandled
}

// .unmute <player> — unmute a player (placeholder)
func (gc *WorldSession) cmdUnmute(server *Server, args string) CommandResult {
	if args == "" {
		gc.sendSystemMessage("Usage: .unmute <player>")

		return CommandHandled
	}

	targetName := NormalizePlayerName(args)
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && strings.EqualFold(other.player.Name, targetName) {
			gc.sendSystemMessage(fmt.Sprintf("Unmuted player %s (placeholder)", targetName))

			return CommandHandled
		}
	}

	gc.sendSystemMessage("Player not found: " + targetName)

	return CommandHandled
}

// .kick <player> — kick a player from the server
func (gc *WorldSession) cmdKick(server *Server, args string) CommandResult {
	if args == "" {
		gc.sendSystemMessage("Usage: .kick <player>")

		return CommandHandled
	}

	targetName := NormalizePlayerName(args)
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && strings.EqualFold(other.player.Name, targetName) {
			gc.sendSystemMessage(fmt.Sprintf("Kicked player %s", targetName))
			other.Close()

			return CommandHandled
		}
	}

	gc.sendSystemMessage("Player not found: " + targetName)

	return CommandHandled
}

// .whispers on|off — toggle whisper acceptance
func (gc *WorldSession) cmdWhispers(args string) CommandResult {
	switch strings.ToLower(strings.TrimSpace(args)) {
	case "on", "1", "true":
		// TODO: set acceptWhispers flag on player
		gc.sendSystemMessage("Whispers enabled (placeholder)")
	case "off", "0", "false":
		// TODO: set acceptWhispers flag on player
		gc.sendSystemMessage("Whispers disabled (placeholder)")
	default:
		gc.sendSystemMessage("Usage: .whispers on|off")
	}

	return CommandHandled
}

// .who <player> — show info about a player
func (gc *WorldSession) cmdWho(server *Server, args string) CommandResult {
	if args == "" {
		gc.sendSystemMessage("Usage: .who <player>")

		return CommandHandled
	}

	targetName := NormalizePlayerName(args)
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && strings.EqualFold(other.player.Name, targetName) {
			p := other.player
			gc.sendSystemMessage(fmt.Sprintf(
				"%s | Lv.%d %s %s | Map:%d Loc:(%.1f, %.1f, %.1f) | HP:%d/%d",
				p.Name, p.Level, raceName(p.Race), className(p.Class),
				p.Location.Map, p.Location.X, p.Location.Y, p.Location.Z,
				p.Health, p.MaxHealth,
			))

			return CommandHandled
		}
	}

	gc.sendSystemMessage("Player not found: " + targetName)

	return CommandHandled
}

// .gps — show your current position
func (gc *WorldSession) cmdGPS() CommandResult {
	p := gc.player
	gc.sendSystemMessage(fmt.Sprintf(
		"Map:%d Zone:%d Loc:(%.1f, %.1f, %.1f, %.1f)",
		p.Location.Map, p.Location.Zone,
		p.Location.X, p.Location.Y, p.Location.Z, p.Location.O,
	))

	return CommandHandled
}

// raceName returns the human-readable name for a player race.
func raceName(race wow.PlayerRace) string {
	switch race {
	case wow.RaceHuman:
		return "Human"
	case wow.RaceOrc:
		return "Orc"
	case wow.RaceDwarf:
		return "Dwarf"
	case wow.RaceNightElf:
		return "Night Elf"
	case wow.RaceUndead:
		return "Undead"
	case wow.RaceTauren:
		return "Tauren"
	case wow.RaceGnome:
		return "Gnome"
	case wow.RaceTroll:
		return "Troll"
	case wow.RaceGoblin:
		return "Goblin"
	case wow.RaceBloodElf:
		return "Blood Elf"
	case wow.RaceDraenei:
		return "Draenei"
	default:
		return "Unknown"
	}
}

// className returns the human-readable name for a player class.
func className(class wow.PlayerClass) string {
	switch class {
	case wow.ClassWarior:
		return "Warrior"
	case wow.ClassPaladin:
		return "Paladin"
	case wow.ClassHunter:
		return "Hunter"
	case wow.ClassRogue:
		return "Rogue"
	case wow.ClassPriest:
		return "Priest"
	case wow.ClassShaman:
		return "Shaman"
	case wow.ClassMage:
		return "Mage"
	case wow.ClassWarlock:
		return "Warlock"
	case wow.ClassDruid:
		return "Druid"
	case wow.ClassDeathKnight:
		return "Death Knight"
	default:
		return "Unknown"
	}
}
