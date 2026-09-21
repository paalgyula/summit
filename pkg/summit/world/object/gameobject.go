package object

import "github.com/paalgyula/summit/pkg/wow"

type GameObject struct {
	*Object
}

func NewGameObject() *GameObject {
	obj := NewObject()
	obj.objectTypeID = wow.TypeIDGameObject
	obj.objectType = wow.TypeMaskGameObject

	// GameObject uses stationary position + low GUID + rotation
	obj.AddUpdateFlags(
		wow.UpdateFlagLowGUID |
			wow.UpdateFlagStationaryPosition |
			wow.UpdateFlagRotation,
	)

	return &GameObject{
		Object: obj,
	}
}
