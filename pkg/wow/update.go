package wow

type ObjectUpdateType uint8

const (
	UpdateTypeValues            = 0
	UpdateTypeMovement          = 1
	UpdateTypeCreateObject      = 2
	UpdateTypeCreateObject2     = 3
	UpdateTypeOutOfRangeObjects = 4
	UpdateTypeNearObjects       = 5
)

type ObjectUpdateFlags uint8

const (
	UpdateFlagNone               = 0x00
	UpdateFlagSelf               = 0x01
	UpdateFlagTransport          = 0x02
	UpdateFlagHasAttackingTarget = 0x04
	UpdateFlagLowGuid            = 0x08
	UpdateFlagHighGuid           = 0x10
	UpdateFlagLiving             = 0x20
	UpdateFlagHasPosition        = 0x40
)
