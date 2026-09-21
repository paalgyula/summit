package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/channel"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// handleChannelChat routes messages to a named channel.
func (gc *WorldSession) handleChannelChat(reader *wow.PacketReader, lang int32, message string) {
	server := gc.server()
	if server == nil {
		return
	}

	var channelName string
	_ = reader.ReadString(&channelName)

	mgr := server.GetChannelManager(0) // neutral
	ch := mgr.Get(channelName)
	if ch == nil {
		return
	}

	if !ch.IsOn(gc.player.GUID()) {
		return
	}

	if err := ch.Say(gc.player.GUID(), message); err != nil {
		return
	}

	pkt := gc.buildChatPacket(ChatMsgChannel, lang, gc.player, message)
	// The packet needs the channel name appended for 3.3.5a client.
	// Build a proper SMSG_MESSAGECHAT with channel name.
	pkt = buildChannelChatPacket(lang, gc.player, channelName, message)

	ch.ForEachPlayer(func(guid wow.GUID, _ *channel.PlayerInfo) {
		if guid == gc.player.GUID() {
			// Echo to sender
			gc.Send(pkt)

			return
		}

		if s := server.SessionByGUID(guid); s != nil {
			s.Send(pkt)
		}
	})
}

// buildChannelChatPacket builds SMSG_MESSAGECHAT for a channel message.
// Channel messages have the channel name after the message text.
func buildChannelChatPacket(lang int32, sender *player.Player, channelName, message string) *wow.Packet {
	pkt := wow.NewPacket(wow.ServerMessagechat)

	_ = pkt.WriteOne(int(ChatMsgChannel))
	_ = pkt.Write(lang)
	_ = pkt.Write(sender.GUID())
	_ = pkt.Write(uint32(0))                    // chat flags
	_ = pkt.Write(sender.GUID())                // target GUID (unused for channels)
	_ = pkt.Write(uint32(len(message) + 1))     // message length incl. terminator
	pkt.WriteString(message)
	_ = pkt.WriteOne(0)                         // chat tag (none)
	_ = pkt.Write(uint32(len(channelName) + 1)) // channel name length incl. terminator
	pkt.WriteString(channelName)

	return pkt
}
