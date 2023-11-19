package world

import (
	"github.com/paalgyula/summit/pkg/wow"
)

func (gc *WorldSession) HandleRealmSplit(data wow.PacketData) {
	var unknown uint32

	_ = data.Reader().Read(&unknown)

	pkt := wow.NewPacket(wow.ServerRealmSplit)
	_ = pkt.Write(unknown)
	_ = pkt.Write(uint32(0))
	// split states:
	// 0x0 realm normal
	// 0x1 realm split
	// 0x2 realm split pending
	pkt.WriteString("01/01/01")

	gc.socket.Send(pkt)
}
