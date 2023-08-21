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

type ObjectUpdateFlags uint16

const (
	UpdateFlagNone               ObjectUpdateFlags = 0x0000
	UpdateFlagSelf               ObjectUpdateFlags = 0x0001
	UpdateFlagTransport          ObjectUpdateFlags = 0x0002
	UpdateFlagHasTarget          ObjectUpdateFlags = 0x0004
	UpdateFlagUnknown            ObjectUpdateFlags = 0x0008
	UpdateFlagLowGuid            ObjectUpdateFlags = 0x0010
	UpdateFlagLiving             ObjectUpdateFlags = 0x0020
	UpdateFlagStationaryPosition ObjectUpdateFlags = 0x0040
	UpdateFlagVehicle            ObjectUpdateFlags = 0x0080
	UpdateFlagPosition           ObjectUpdateFlags = 0x0100
	UpdateFlagRotation           ObjectUpdateFlags = 0x0200
)

type ObjectManager struct{}

func (*ObjectManager) CreateUpdatePacketFor(*player.Player) {
	p := wow.NewPacket(wow.ServerUpdateObject)

	var updateFlag ObjectUpdateFlags = UpdateFlagNone
	updateFlag |= UpdateFlagSelf

	p.Write(updateFlag) // Creating self

	p.Write(uint32(0)) // Has transport
}
