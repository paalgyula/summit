package lfg

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// Handler processes LFG-related opcodes.
type Handler struct {
	mgr     *Manager
	session SessionSender
}

// SessionSender is the interface a WorldSession must implement to send packets.
type SessionSender interface {
	Send(pkt *wow.Packet)
}

// NewHandler creates an LFG opcode handler.
func NewHandler(mgr *Manager, session SessionSender) *Handler {
	return &Handler{
		mgr:     mgr,
		session: session,
	}
}

// HandleJoin processes CMSG_LFG_JOIN.
func (h *Handler) HandleJoin(data []byte) {
	reader := wow.NewPacketReader(data)

	var roles uint32
	_ = reader.Read(&roles)

	var slotCount uint32
	_ = reader.Read(&slotCount)

	var dungeonID uint32
	_ = reader.Read(&dungeonID)

	var comment string
	_ = reader.ReadString(&comment)

	// For now, simplified: single dungeon selection
	_ = slotCount

	if err := h.mgr.JoinQueue(0, 0, roles, dungeonID); err != nil {
		h.sendJoinResult(JoinResultLFGJoinFailed, 0)

		return
	}

	h.sendJoinResult(JoinResultOk, 0)
	h.sendQueueStatus(QueueStatusData{
		DungeonID: dungeonID,
	})
}

// HandleLeave processes CMSG_LFG_LEAVE.
func (h *Handler) HandleLeave(_ []byte) {
	_ = h.mgr.LeaveQueue(0)
}

// HandleSetRoles processes CMSG_LFG_SET_ROLES.
func (h *Handler) HandleSetRoles(data []byte) {
	reader := wow.NewPacketReader(data)

	var roles uint32
	_ = reader.Read(&roles)

	_ = h.mgr.SetRoles(0, roles)
}

// HandleProposalResponse processes CMSG_LFG_PROPOSAL_RESPONSE.
func (h *Handler) HandleProposalResponse(data []byte) {
	reader := wow.NewPacketReader(data)

	var accept uint32
	_ = reader.Read(&accept)

	_ = h.mgr.ProposalResponse(0, accept != 0)
}

// HandleGetStatus processes CMSG_LFG_GET_STATUS.
func (h *Handler) HandleGetStatus(_ []byte) {
	pd := h.mgr.GetPlayerData(0)
	if pd == nil {
		return
	}

	// Send current state update
	h.sendUpdatePlayer(pd)
}

// sendJoinResult sends SMSG_LFG_JOIN_RESULT.
func (h *Handler) sendJoinResult(result, state uint32) {
	pkt := wow.NewPacket(wow.ServerLfgJoinResult)

	_ = pkt.Write(result)
	_ = pkt.Write(state)

	h.session.Send(pkt)
}

// sendQueueStatus sends SMSG_LFG_QUEUE_STATUS.
func (h *Handler) sendQueueStatus(qs QueueStatusData) {
	pkt := wow.NewPacket(wow.ServerLfgQueueStatus)

	_ = pkt.Write(qs.DungeonID)
	_ = pkt.Write(qs.WaitTimeAvg)
	_ = pkt.Write(qs.WaitTime)
	_ = pkt.Write(qs.WaitTimeTank)
	_ = pkt.Write(qs.WaitTimeHealer)
	_ = pkt.Write(qs.WaitTimeDps)
	_ = pkt.WriteOne(int(qs.TanksNeeded))
	_ = pkt.WriteOne(int(qs.HealersNeeded))
	_ = pkt.WriteOne(int(qs.DpsNeeded))
	_ = pkt.Write(qs.QueuedTime)

	h.session.Send(pkt)
}

// sendUpdatePlayer sends SMSG_LFG_UPDATE_PLAYER or SMSG_LFG_UPDATE_PARTY.
func (h *Handler) sendUpdatePlayer(pd *PlayerData) {
	pkt := wow.NewPacket(wow.ServerLfgUpdatePlayer)

	_ = pkt.WriteOne(1) // in queue
	_ = pkt.Write(uint32(pd.State))
	_ = pkt.Write(pd.Roles)
	_ = pkt.Write(pd.SelectedDungeon)

	h.session.Send(pkt)
}

// sendProposalUpdate sends SMSG_LFG_PROPOSAL_UPDATE.
func (h *Handler) sendProposalUpdate(prop *ProposalData) {
	pkt := wow.NewPacket(wow.ServerLfgProposalUpdate)

	_ = pkt.WriteOne(int(StateProposal))
	_ = pkt.Write(prop.DungeonID)
	_ = pkt.Write(uint32(len(prop.Players)))

	for _, guid := range prop.Players {
		_ = pkt.Write(uint64(guid))
		accepted := prop.Accepted[guid]
		_ = pkt.WriteOne(0) // role (simplified)
		if accepted {
			_ = pkt.WriteOne(1) // responded
		} else {
			_ = pkt.WriteOne(0) // not responded
		}
		_ = pkt.WriteOne(0) // result (pending)
	}

	h.session.Send(pkt)
}

// SendLfgDisabled sends SMSG_LFG_DISABLED when LFG is not available.
func (h *Handler) SendLfgDisabled() {
	pkt := wow.NewPacket(wow.ServerLfgDisabled)
	h.session.Send(pkt)
}
