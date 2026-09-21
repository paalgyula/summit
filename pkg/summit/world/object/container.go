package object

import "github.com/paalgyula/summit/pkg/wow"

// Container represents a bag/container item.
type Container struct {
	*Object
}

// NewContainer creates a new Container with proper type flags.
func NewContainer() *Container {
	obj := NewObject()
	obj.objectTypeID = wow.TypeIDContainer
	obj.objectType = wow.TypeMaskContainer

	return &Container{
		Object: obj,
	}
}
