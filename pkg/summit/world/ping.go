package world

import "github.com/paalgyula/summit/pkg/wow"

// PingHandler handles player ping packets.
func (gc *WorldSession) PingHandler(pkt *wow.Packet) {
	pingData := pkt.Bytes()

	// TODO: check for is the ping interval smaller than 27sec

	pongpkt := wow.NewPacket(wow.ServerPong)
	pongpkt.Write(pingData)

	gc.socket.Send(pongpkt)
}
