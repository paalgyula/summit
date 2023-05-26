package world

import "github.com/paalgyula/summit/pkg/summit/world/packets"

func (gc *GameClient) PingHandler() {
	gc.SendPacket(packets.ServerPong, make([]byte, 2))
}
