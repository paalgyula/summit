package world

import (
	"github.com/paalgyula/summit/pkg/wow"
)

func (gc *GameClient) HandleRealmSplit(data wow.PacketData) {
	var unknown uint32
	data.Reader().Read(&unknown)

	w := wow.NewPacket(wow.ServerRealmSplit)
	w.Write(unknown)
	w.Write(uint32(0))
	// split states:
	// 0x0 realm normal
	// 0x1 realm split
	// 0x2 realm split pending
	w.WriteString("01/01/01")

	gc.SendPayload(int(wow.ServerRealmSplit), w.Bytes())
}
