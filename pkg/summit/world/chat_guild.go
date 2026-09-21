package world

// handleGuildChat routes guild and officer chat messages to all online
// members of the same guild.
func (gc *WorldSession) handleGuildChat(chatType uint32, lang int32, message string) {
	server := gc.server()
	if server == nil {
		return
	}

	if gc.player.GuildID == 0 {
		return
	}

	pkt := gc.buildChatPacket(chatType, lang, gc.player, message)

	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld && other.player.GuildID == gc.player.GuildID {
			other.Send(pkt)
		}
	}
}
