package wow

// ObjectUpdateType is the kind of an SMSG_UPDATE_OBJECT block.
type ObjectUpdateType uint8

const (
	UpdateTypeValues            = 0
	UpdateTypeMovement          = 1
	UpdateTypeCreateObject      = 2
	UpdateTypeCreateObject2     = 3
	UpdateTypeOutOfRangeObjects = 4
	UpdateTypeNearObjects       = 5
)

// ObjectUpdateFlags select the parts of a create / movement block
// (3.3.5a layout: a uint16 in the packet).
type ObjectUpdateFlags uint16

const (
	UpdateFlagNone               ObjectUpdateFlags = 0x0000
	UpdateFlagSelf               ObjectUpdateFlags = 0x0001
	UpdateFlagTransport          ObjectUpdateFlags = 0x0002
	UpdateFlagHasTarget          ObjectUpdateFlags = 0x0004
	UpdateFlagUnknown            ObjectUpdateFlags = 0x0008
	UpdateFlagLowGUID            ObjectUpdateFlags = 0x0010
	UpdateFlagLiving             ObjectUpdateFlags = 0x0020
	UpdateFlagStationaryPosition ObjectUpdateFlags = 0x0040
	UpdateFlagVehicle            ObjectUpdateFlags = 0x0080
	UpdateFlagPosition           ObjectUpdateFlags = 0x0100
	UpdateFlagRotation           ObjectUpdateFlags = 0x0200
)
