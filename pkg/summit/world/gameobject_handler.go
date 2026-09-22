package world

import "github.com/paalgyula/summit/pkg/wow"

// HandleGameObjectUse handles CMSG_GAMEOBJ_USE (0x0B1): the player right-clicks
// a game object. Mirrors WorldSession::HandleGameObjectUseOpcode.
func (gc *WorldSession) HandleGameObjectUse(data wow.PacketData) {
	gc.gameObjectUse(data, false)
}

// HandleGameObjectReportUse handles CMSG_GAMEOBJ_REPORT_USE (0x481), sent by the
// client after an interaction was reported by the server.
func (gc *WorldSession) HandleGameObjectReportUse(data wow.PacketData) {
	gc.gameObjectUse(data, true)
}

func (gc *WorldSession) gameObjectUse(data wow.PacketData, report bool) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var rawGUID uint64
	if err := reader.Read(&rawGUID); err != nil {
		return
	}

	guid := wow.GUID(rawGUID)
	if guid.High() != wow.GameObjectGUID {
		return
	}

	server, ok := gc.ws.(*Server)
	if !ok || server.gameObjects == nil {
		return
	}

	gobj := server.gameObjects.GetObjectByGUID(guid)
	if gobj == nil {
		return
	}

	gc.log.Debug().
		Uint32("guid", guid.Counter()).
		Uint32("entry", gobj.Entry).
		Bool("report", report).
		Msg("CMSG_GAMEOBJ_USE")

	// Distance check, an event handler that is not in range may not be used.
	if !gobj.IsWithinInteractionDistance(gc.player.Location.X, gc.player.Location.Y, gc.player.Location.Z) {
		gc.log.Debug().
			Uint32("guid", guid.Counter()).
			Msg("game object used out of interaction range")

		return
	}

	ctx := GameObjectUseContext{Server: server, Session: gc, Player: gc.player}
	if err := gobj.Use(ctx); err != nil {
		gc.log.Debug().
			Err(err).
			Uint32("guid", guid.Counter()).
			Msg("game object use rejected")
	}
}
