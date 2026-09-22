package object

import "github.com/paalgyula/summit/pkg/wow"

type GameObject struct {
	*Object
}

func NewGameObject() *GameObject {
	obj := NewObject()
	obj.objectTypeID = wow.TypeIDGameObject
	obj.objectType = wow.TypeMaskGameObject

	// GameObject uses stationary position + position + low GUID + rotation
	// (mirrors GameObject::GameObject's m_updateFlag).
	obj.AddUpdateFlags(
		wow.UpdateFlagLowGUID |
			wow.UpdateFlagStationaryPosition |
			wow.UpdateFlagPosition |
			wow.UpdateFlagRotation,
	)

	return &GameObject{
		Object: obj,
	}
}
