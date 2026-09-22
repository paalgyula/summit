package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

// Stand states (UNIT_FIELD_BYTES_1 byte 0) the text emotes switch to.
const (
	StandStateStand uint8 = 0
	StandStateSit   uint8 = 1
	StandStateSleep uint8 = 3
	StandStateKneel uint8 = 8
)

// Emote ids with EmoteTypeStandState map to a stand state instead of an animation.
const (
	EmoteStateSleep = 12
	EmoteStateSit   = 13
	EmoteStateStand = 26
	EmoteStateKneel = 68
)

// HandleTextEmote handles CMSG_TEXT_EMOTE (u32 text emote, u32 emote number,
// u64 target): plays the emote's animation for everyone and broadcasts the
// SMSG_TEXT_EMOTE line, mirroring WorldSession::HandleTextEmoteOpcode.
func (gc *WorldSession) HandleTextEmote(data wow.PacketData) {
	if gc.player == nil || !gc.player.IsInWorld {
		return
	}

	reader := wow.NewPacketReader(data)

	var (
		textEmote uint32
		emoteNum  uint32
		target    uint64
	)

	_ = reader.Read(&textEmote)
	_ = reader.Read(&emoteNum)
	_ = reader.Read(&target)

	emoteID, ok := TextEmotes[textEmote]
	if !ok {
		gc.log.Debug().Uint32("textEmote", textEmote).Msg("unknown text emote")

		return
	}

	if emoteID != 0 {
		gc.playEmote(emoteID)
	}

	targetName := ""

	if server, ok := gc.ws.(*Server); ok && target != 0 {
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && uint64(other.player.GUID()) == target {
				targetName = other.player.Name

				break
			}
		}
	}

	pkt := wow.NewPacket(wow.ServerTextEmote)
	_ = pkt.Write(gc.player.GUID())
	_ = pkt.Write(textEmote)
	_ = pkt.Write(emoteNum)
	_ = pkt.Write(uint32(len(targetName) + 1))
	pkt.WriteString(targetName)

	gc.broadcastInWorld(pkt, true)
}

// HandleEmote handles CMSG_EMOTE (u32 emote id): a bare animation request.
func (gc *WorldSession) HandleEmote(data wow.PacketData) {
	if gc.player == nil || !gc.player.IsInWorld {
		return
	}

	reader := wow.NewPacketReader(data)

	var emoteID uint32

	_ = reader.Read(&emoteID)

	if _, ok := Emotes[emoteID]; ok {
		gc.playEmote(emoteID)
	}
}

// playEmote applies an emote the way the C++ core does: one-shots go out as
// SMSG_EMOTE, looping states become UNIT_NPC_EMOTESTATE and sit / sleep /
// kneel become the stand state. State changes reach the clients through the
// map's value updates.
func (gc *WorldSession) playEmote(emoteID uint32) {
	info, ok := Emotes[emoteID]
	if !ok {
		return
	}

	switch info.Type {
	case EmoteTypeStandState:
		switch emoteID {
		case EmoteStateSit:
			gc.setStandState(StandStateSit)
		case EmoteStateSleep:
			gc.setStandState(StandStateSleep)
		case EmoteStateKneel:
			gc.setStandState(StandStateKneel)
		default:
			gc.setStandState(StandStateStand)
		}
	case EmoteTypeState:
		gc.setEmoteState(emoteID)
	default:
		pkt := wow.NewPacket(wow.ServerEmote)
		_ = pkt.Write(emoteID)
		_ = pkt.Write(gc.player.GUID())

		gc.broadcastInWorld(pkt, true)
	}
}

func (gc *WorldSession) setEmoteState(emoteID uint32) {
	gc.player.Object.SetUInt32Value(object.UnitNpcEmotestate, emoteID)
}

func (gc *WorldSession) setStandState(state uint8) {
	// Sitting down (or standing up) ends a looping emote such as a dance
	gc.setEmoteState(0)
	gc.player.Object.SetByteValue(object.UnitFieldBytes_1, 0, state)
}

// clearEmotes ends a looping emote or a sit / sleep / kneel when the player moves.
func (gc *WorldSession) clearEmotes() {
	if gc.player.Object.GetUInt32Value(object.UnitNpcEmotestate) != 0 {
		gc.setEmoteState(0)
	}

	if gc.player.Object.GetUInt32Value(object.UnitFieldBytes_1)&0xff != uint32(StandStateStand) {
		gc.setStandState(StandStateStand)
	}
}

// broadcastInWorld sends a packet to every in-world session, optionally including this one.
func (gc *WorldSession) broadcastInWorld(pkt *wow.Packet, includeSelf bool) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	for _, other := range server.GetOnlineSessions() {
		if other.player == nil || !other.player.IsInWorld {
			continue
		}

		if other == gc && !includeSelf {
			continue
		}

		other.socket.Send(pkt)
	}
}

// Sheath states of UNIT_FIELD_BYTES_2 byte 0 (SheathState).
const (
	SheathStateUnarmed uint8 = 0
	SheathStateMelee   uint8 = 1
	SheathStateRanged  uint8 = 2
)

// HandleSetSheathed handles CMSG_SETSHEATHED: the client draws or sheathes
// its weapons (it sends melee when auto-attack starts); everyone in view
// gets the new state through the value update.
func (gc *WorldSession) HandleSetSheathed(data wow.PacketData) {
	if gc.player == nil || !gc.player.IsInWorld {
		return
	}

	reader := wow.NewPacketReader(data)

	var state uint32
	if err := reader.Read(&state); err != nil || state > uint32(SheathStateRanged) {
		return
	}

	gc.player.Object.SetByteValue(object.UnitFieldBytes_2, 0, uint8(state))
}
