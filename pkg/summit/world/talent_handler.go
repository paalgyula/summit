package world

// talent_handler.go — CMSG_LEARN_TALENT / SMSG_TALENTS_INFO handler.
//
// Reference: AzerothCore
//   - src/server/game/Entities/Player/Player.cpp  (LearnTalent)
//   - src/server/game/Handlers/TalentHandler.cpp  (HandleLearnTalentOpcode)
//
// WotLK talent system overview:
//   - Each player gets 1 talent point per level starting at level 10.
//   - CMSG_LEARN_TALENT sends (talentID uint32, rank uint32).
//   - On success the server decrements PlayerCharacterPoints1 by 1 and sends
//     SMSG_TALENTS_INFO with the full talent snapshot.
//   - Full DBC-driven talent validation (TalentTab, prerequisites, max ranks)
//     is deferred to a future issue; this handler validates the point budget
//     and echos a compressed talent block back to the client.

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleLearnTalent processes CMSG_LEARN_TALENT.
//
// Packet layout:
//
//	uint32  talentID
//	uint32  requestedRank (0-based)
func (gc *WorldSession) HandleLearnTalent(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	p := gc.player
	r := wow.NewPacketReader(data)

	var talentID, requestedRank uint32

	_ = r.Read(&talentID)
	_ = r.Read(&requestedRank)

	gc.log.Debug().
		Uint32("talentID", talentID).
		Uint32("rank", requestedRank).
		Str("player", p.Name).
		Msg("CMSG_LEARN_TALENT received")

	// Check available talent points
	availablePoints := p.Object.GetUInt32Value(object.PlayerCharacterPoints1)
	if availablePoints == 0 {
		gc.log.Warn().
			Str("player", p.Name).
			Msg("no talent points available — ignoring CMSG_LEARN_TALENT")

		return
	}

	// Basic range guard (WotLK: 3 talent trees per class, up to 71 talents each,
	// max rank 5; actual IDs are in Talent.dbc — we accept without DBC for now).
	if talentID == 0 {
		return
	}

	// Spend one talent point
	p.Object.SetUInt32Value(object.PlayerCharacterPoints1, availablePoints-1)

	gc.log.Info().
		Uint32("talentID", talentID).
		Uint32("rank", requestedRank).
		Uint32("pointsLeft", availablePoints-1).
		Str("player", p.Name).
		Msg("talent learned")

	// Send full talent info snapshot back to the client
	gc.sendTalentsInfo()
}

// sendTalentsInfo sends SMSG_TALENTS_INFO to the client.
//
// WotLK packet layout:
//
//	uint8   talentGroup (0 = primary, 1 = secondary; dual-spec)
//	uint8   talentGroupCount
//	for each group:
//	  uint32  count
//	  for each talent:
//	    uint32  talentID
//	    uint8   rank
//	uint8   glyphCount
//	for each glyph:
//	  uint8   slotIndex
//	  uint32  spellID
//
// We send a minimal response: active group 0, 1 group, 0 talents, 0 glyphs.
// The client will not crash; actual talent rank tracking is deferred.
func (gc *WorldSession) sendTalentsInfo() {
	pkt := wow.NewPacket(wow.ServerTalentsInfo)
	_ = pkt.Write(uint8(0)) // active talent group index (primary)
	_ = pkt.Write(uint8(1)) // number of talent groups (1 = no dual spec yet)

	// Group 0: talent count = 0 (client will show points but empty tree)
	_ = pkt.Write(uint32(0)) // talent entry count

	// Glyph count = 0
	_ = pkt.Write(uint8(0))

	gc.Send(pkt)
}
