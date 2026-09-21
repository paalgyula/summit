package world

import "github.com/paalgyula/summit/pkg/wow"

// MaxActionButtons is the size of the 3.3.5a action bar packet.
const MaxActionButtons = 144

// HandleSetActionButton handles CMSG_SET_ACTION_BUTTON (u8 button, u32 packed
// action; 0 clears the slot). Only the persisted range of the bar is kept.
func (gc *WorldSession) HandleSetActionButton(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var (
		button uint8
		packed uint32
	)

	if err := reader.Read(&button); err != nil {
		return
	}

	if err := reader.Read(&packed); err != nil {
		return
	}

	if int(button) >= len(gc.player.Actions) {
		gc.log.Debug().Uint8("button", button).Msg("action button outside the persisted bar")

		return
	}

	gc.player.Actions[button] = packed
}
