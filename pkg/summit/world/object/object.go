package object

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// 1973 in MoP? Seems 1326 in wotlk.
// const dataLength int = int(wow.NumMsgTypes)

type Object struct {
	guid wow.GUID

	UpdateData []wow.Packet
	UpdateMask *UpdateMask

	movementFlags wow.MovementFlag

	objectTypeID wow.TypeID
	objectType   wow.TypeMask

	isInWorld bool
	isUpdated bool

	uint32Values int
	// valuesCount  int

	updateFlags wow.ObjectUpdateFlags
}

func NewObject() *Object {
	//nolint:exhaustruct
	return &Object{
		objectTypeID: wow.TypeIDObject,
		objectType:   wow.TypeMaskObject,

		isInWorld:    false,
		isUpdated:    false,
		uint32Values: 0,
	}
}

func (o *Object) AddUpdateFlags(flags ...wow.ObjectUpdateFlags) {
	for _, ouf := range flags {
		o.updateFlags |= ouf
	}
}

func (o *Object) UpdateFlags() wow.ObjectUpdateFlags {
	return o.updateFlags
}

func (o *Object) MovementFlags() wow.MovementFlag {
	return o.movementFlags
}

func (o *Object) GameObjectType() wow.GameObjectType {
	// return wow.GameObjectTypeObject
	return wow.GameObjectTypeGeneric
}

func (o *Object) GUID() wow.GUID {
	return o.guid
}

// SetGUID sets the object's GUID.
func (o *Object) SetGUID(g wow.GUID) {
	o.guid = g
}

// SetObjectType sets the object type mask.
func (o *Object) SetObjectType(mask wow.TypeMask) {
	o.objectType = mask
}

// ObjectType returns the object type mask.
func (o *Object) ObjectType() wow.TypeMask {
	return o.objectType
}
