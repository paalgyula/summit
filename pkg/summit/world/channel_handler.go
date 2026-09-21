package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/channel"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleJoinChannel handles CMSG_JOIN_CHANNEL.
// Packet: uint32 channelId, uint8 unknown1, uint8 unknown2, string channelName, string password
func (gc *WorldSession) HandleJoinChannel(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var channelID uint32
	_ = reader.Read(&channelID)

	var unknown1 uint8
	_ = reader.Read(&unknown1)

	var unknown2 uint8
	_ = reader.Read(&unknown2)

	var name string
	_ = reader.ReadString(&name)

	var password string
	_ = reader.ReadString(&password)

	// Validate channel name
	if len(name) == 0 || len(name) >= 100 {
		return
	}

	// Reject channel names starting with a digit
	if name[0] >= '0' && name[0] <= '9' {
		return
	}

	server := gc.server()
	if server == nil {
		return
	}

	// Route to the correct faction channel manager
	team := int(gc.player.Race) // simplified: alliance=1, horde=2
	// TODO: proper team detection from Race
	_ = team

	mgr := server.GetChannelManager(0) // neutral for now; fix with real team
	ch := mgr.GetOrCreate(name, channelID)

	if err := ch.Join(gc.player.GUID(), password); err != nil {
		gc.log.Debug().Err(err).Str("channel", name).Msg("join channel failed")

		return
	}

	gc.log.Debug().Str("channel", name).Uint32("id", channelID).Msg("joined channel")

	// Send SMSG_CHANNEL_NOTIFY(0x00 = joined) to the player
	gc.sendChannelNotify(name, 0x00)

	// Announce to other members if channel has announce enabled
	if ch.Announce {
		pkt := buildChatPacketStatic(ChatMsgChannel, LangUniversal, gc.player.GUID(), gc.player.GUID(), name)
		ch.ForEachPlayer(func(guid wow.GUID, _ *channel.PlayerInfo) {
			if guid != gc.player.GUID() {
				if s := server.SessionByGUID(guid); s != nil {
					s.Send(pkt)
				}
			}
		})
	}
}

// HandleLeaveChannel handles CMSG_LEAVE_CHANNEL.
// Packet: uint32 unknown, string channelName
func (gc *WorldSession) HandleLeaveChannel(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var unknown uint32
	_ = reader.Read(&unknown)

	var name string
	_ = reader.ReadString(&name)

	if len(name) == 0 {
		return
	}

	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetChannelManager(0)
	ch := mgr.Get(name)
	if ch != nil {
		ch.Leave(gc.player.GUID())
		mgr.Remove(name)
	}

	gc.sendChannelNotify(name, 0x01) // left
}

// HandleChannelList handles CMSG_CHANNEL_LIST.
// Packet: string channelName
func (gc *WorldSession) HandleChannelList(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var name string
	_ = reader.ReadString(&name)

	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetChannelManager(0)
	ch := mgr.Get(name)
	if ch == nil {
		return
	}

	// Build SMSG_CHANNEL_LIST
	payload := channel.BuildChannelList(ch)
	pkt := wow.NewPacketWithData(wow.ServerChannelList, payload)
	gc.Send(pkt)
}

// HandleChannelPassword handles CMSG_CHANNEL_PASSWORD.
// Packet: string channelName, string password
func (gc *WorldSession) HandleChannelPassword(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var name string
	_ = reader.ReadString(&name)

	var password string
	_ = reader.ReadString(&password)

	// Max password length from AC: MAX_CHANNEL_PASS_STR = 31
	if len(password) > 31 {
		return
	}

	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetChannelManager(0)
	ch := mgr.Get(name)
	if ch == nil {
		return
	}

	if err := ch.SetPassword(gc.player.GUID(), password); err != nil {
		gc.log.Debug().Err(err).Str("channel", name).Msg("set channel password failed")
	}
}

// HandleChannelSetOwner handles CMSG_CHANNEL_SET_OWNER.
// Packet: string channelName, string targetName
func (gc *WorldSession) HandleChannelSetOwner(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var name string
	_ = reader.ReadString(&name)

	var targetName string
	_ = reader.ReadString(&targetName)

	_ = targetName // TODO: resolve player by name and transfer ownership

	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetChannelManager(0)
	ch := mgr.Get(name)
	if ch == nil {
		return
	}

	_ = ch // TODO: implement SetOwner by name
}

// HandleChannelOwner handles CMSG_CHANNEL_OWNER.
// Packet: string channelName
func (gc *WorldSession) HandleChannelOwner(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var name string
	_ = reader.ReadString(&name)

	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetChannelManager(0)
	ch := mgr.Get(name)
	if ch == nil {
		return
	}

	// TODO: Send SMSG_CHANNEL_OWNER with the owner's GUID
	_ = ch
}

// HandleChannelModerator handles CMSG_CHANNEL_MODERATOR.
// Packet: string channelName, string targetName
func (gc *WorldSession) HandleChannelModerator(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var name string
	_ = reader.ReadString(&name)

	var targetName string
	_ = reader.ReadString(&targetName)

	_ = name
	_ = targetName
	// TODO: resolve target player by name, then ch.SetModerator(caller, target, true)
}

// HandleChannelUnmoderator handles CMSG_CHANNEL_UNMODERATOR.
func (gc *WorldSession) HandleChannelUnmoderator(data wow.PacketData) {
	// Same structure as HandleChannelModerator but sets moderator to false
	gc.HandleChannelModerator(data)
}

// HandleChannelMute handles CMSG_CHANNEL_MUTE.
// Packet: string channelName, string targetName
func (gc *WorldSession) HandleChannelMute(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var name string
	_ = reader.ReadString(&name)

	var targetName string
	_ = reader.ReadString(&targetName)

	_ = name
	_ = targetName
	// TODO: resolve target player by name, then ch.SetMute(caller, target, true)
}

// HandleChannelUnmute handles CMSG_CHANNEL_UNMUTE.
func (gc *WorldSession) HandleChannelUnmute(data wow.PacketData) {
	// Same structure as HandleChannelMute but sets mute to false
	gc.HandleChannelMute(data)
}

// HandleChannelKick handles CMSG_CHANNEL_KICK.
// Packet: string channelName, string targetName
func (gc *WorldSession) HandleChannelKick(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var name string
	_ = reader.ReadString(&name)

	var targetName string
	_ = reader.ReadString(&targetName)

	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetChannelManager(0)
	ch := mgr.Get(name)
	if ch == nil {
		return
	}

	// TODO: resolve targetName to GUID, then:
	// ch.Kick(gc.player.GUID(), targetGUID)
	_ = ch
	_ = targetName
}

// HandleChannelBan handles CMSG_CHANNEL_BAN.
// Packet: string channelName, string targetName
func (gc *WorldSession) HandleChannelBan(data wow.PacketData) {
	gc.HandleChannelKick(data) // same packet structure, different action
}

// HandleChannelUnban handles CMSG_CHANNEL_UNBAN.
// Packet: string channelName, string targetName
func (gc *WorldSession) HandleChannelUnban(data wow.PacketData) {
	gc.HandleChannelKick(data) // same packet structure
}

// sendChannelNotify sends SMSG_CHANNEL_NOTIFY to the client.
// notifyType: 0x00=joined, 0x01=left, 0x02=you_kicked, 0x03=you_banned,
//
//	0x04=player_joined, 0x05=player_left, 0x06=player_kicked,
//	0x07=player_banned, 0x08=moderator, 0x09=unmoderator,
//	0x0A=muted, 0x0B=unmuted, 0x0C=owner, 0x0D=mode_change
func (gc *WorldSession) sendChannelNotify(name string, notifyType uint8) {
	pkt := wow.NewPacket(wow.ServerChannelNotify)

	_ = pkt.WriteOne(int(notifyType))
	pkt.WriteString(name)
	_ = pkt.Write(uint64(0)) // target GUID (0 for self-notifications)

	gc.Send(pkt)
}
