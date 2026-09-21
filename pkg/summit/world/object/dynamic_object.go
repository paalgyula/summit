package object

import "github.com/paalgyula/summit/pkg/wow"

// DynamicObject represents a dynamic object (e.g., spell effects, area of effect).
type DynamicObject struct {
	*Object
}

// NewDynamicObject creates a new DynamicObject with proper type flags.
func NewDynamicObject() *DynamicObject {
	obj := NewObject()
	obj.objectTypeID = wow.TypeIDDynamicoObject
	obj.objectType = wow.TypeMaskDynamicObject

	// DynamicObject uses stationary position (no movement)
	obj.AddUpdateFlags(wow.UpdateFlagStationaryPosition)

	return &DynamicObject{
		Object: obj,
	}
}
