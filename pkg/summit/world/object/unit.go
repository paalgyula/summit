package object

import "github.com/paalgyula/summit/pkg/wow"

type Unit struct {
	Object *Object

	mask wow.UnitMask

	Speed [wow.MoveTypeMax]float32
}

func NewUnit() *Unit {
	obj := NewObject()

	obj.AddUpdateFlags(
		wow.UpdateFlagHighGuid |
			wow.UpdateFlagLiving |
			wow.UpdateFlagHasPosition,
	)

	obj.objectType |= wow.TypeMaskUnit

	return &Unit{
		Object: obj,
		mask:   wow.UnitMaskNone,
	}
}

func Type() wow.TypeID {
	return wow.TypeIDUnit
}

// GetSpeed returns the speed of the Unit for the given MoveType.
//
// mt: The MoveType to get the speed for.
// float32: The speed of the Unit for the given MoveType.
func (u *Unit) GetSpeed(mt wow.MoveType) float32 {
	return u.Speed[mt]
}
