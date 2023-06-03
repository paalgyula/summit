package wow

type PowerType int32

const (
	PowerTypeMana      PowerType = 0
	PowerTypeRage      PowerType = 1
	PowerTypeFocus     PowerType = 2
	PowerTypeEnergy    PowerType = 3
	PowerTypeHappiness PowerType = 4
	PowerTypeHealth    PowerType = -2 // 0xFFFFFFFE as a signed value
)

const MaxPowerTypes = 5

type UnitMask uint32

const (
	UnitMaskNone                UnitMask = 0x00000000
	UnitMaskSummon              UnitMask = 0x00000001
	UnitMaskMinion              UnitMask = 0x00000002
	UnitMaskGuardian            UnitMask = 0x00000004
	UnitMaskTotem               UnitMask = 0x00000008
	UnitMaskPet                 UnitMask = 0x00000010
	UnitMaskPuppet              UnitMask = 0x00000040
	UnitMaskHunterPet           UnitMask = 0x00000080
	UnitMaskControlableGuardian UnitMask = 0x00000100
)
