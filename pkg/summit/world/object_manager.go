package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

type ObjectUpdateType int

const (
	ObjectUpdateTypeValues            ObjectUpdateType = 0
	ObjectUpdateTypeMovement          ObjectUpdateType = 1
	ObjectUpdateTypeCreateObject      ObjectUpdateType = 2
	ObjectUpdateTypeCreateObject2     ObjectUpdateType = 3
	ObjectUpdateTypeOutOfRangeObjects ObjectUpdateType = 4
	ObjectUpdateTypeNearObjects       ObjectUpdateType = 5
)

type ObjectManager struct{}

func (*ObjectManager) CreateUpdatePacketFor(*player.Player) {
	_ = wow.NewPacket(wow.ServerUpdateObject)
}
