package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleSetSelection handles CMSG_SET_SELECTION (u64 guid, 0 to clear): the
// player's target goes into UNIT_FIELD_TARGET, which the map's value updates
// carry to everyone in view.
func (gc *WorldSession) HandleSetSelection(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var target uint64
	if err := reader.Read(&target); err != nil {
		return
	}

	gc.player.Target = target
	gc.player.Object.SetUInt32Value(object.UnitFieldTarget, uint32(target))
	gc.player.Object.SetUInt32Value(object.UnitFieldTarget+1, uint32(target>>32))
}
