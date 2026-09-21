package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/lfg"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleLfgJoin handles CMSG_LFG_JOIN.
func (gc *WorldSession) HandleLfgJoin(data wow.PacketData) {
	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetLfgManager()
	h := lfg.NewHandler(mgr, gc)
	h.HandleJoin(data)
}

// HandleLfgLeave handles CMSG_LFG_LEAVE.
func (gc *WorldSession) HandleLfgLeave(data wow.PacketData) {
	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetLfgManager()
	h := lfg.NewHandler(mgr, gc)
	h.HandleLeave(data)
}

// HandleLfgSetRoles handles CMSG_LFG_SET_ROLES.
func (gc *WorldSession) HandleLfgSetRoles(data wow.PacketData) {
	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetLfgManager()
	h := lfg.NewHandler(mgr, gc)
	h.HandleSetRoles(data)
}

// HandleLfgGetStatus handles CMSG_LFG_GET_STATUS.
func (gc *WorldSession) HandleLfgGetStatus(data wow.PacketData) {
	server := gc.server()
	if server == nil {
		return
	}

	mgr := server.GetLfgManager()
	h := lfg.NewHandler(mgr, gc)
	h.HandleGetStatus(data)
}
