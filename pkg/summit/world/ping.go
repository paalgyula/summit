package world

import "github.com/paalgyula/summit/pkg/wow"

func (gc *GameClient) PingHandler() {
	gc.SendPayload(int(wow.ServerPong), make([]byte, 2))
}
