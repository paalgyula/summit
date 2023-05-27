package world

import "github.com/paalgyula/summit/pkg/summit/world/packets"

func (gc *GameClient) PingHandler() {
	gc.SendPayload(packets.ServerPong.Int(), make([]byte, 2))
}
