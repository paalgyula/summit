package world

import "github.com/paalgyula/summit/pkg/blizzard/world/packets"

func (gc *GameClient) PingHandler() {
	gc.SendPacket(packets.ServerPong, make([]byte, 2))
}
