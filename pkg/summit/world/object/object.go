package object

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// 1973 in MoP? Seems 1326 in wotlk.
const dataLength int = int(wow.NumMsgTypes)

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
	valuesCount  int

	updateFlags wow.ObjectUpdateFlags
}

func NewObject() *Object {
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

func (o *Object) Guid() wow.GUID {
	return o.guid
}
