package wow

type TypeMask uint16

const (
	TypeMaskObject        TypeMask = 0x0001
	TypeMaskItem          TypeMask = 0x0002
	TypeMaskContainer     TypeMask = TypeMaskItem | 0x0004
	TypeMaskUnit          TypeMask = 0x0008
	TypeMaskPlayer        TypeMask = 0x0010 | TypeMaskUnit
	TypeMaskGameObject    TypeMask = 0x0020
	TypeMaskDynamicObject TypeMask = 0x0040
	TypeMaskCorpse        TypeMask = 0x0080
	TypeMaskSeer          TypeMask = TypeMaskUnit | TypeMaskDynamicObject
)
