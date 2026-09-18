package object

import (
	"encoding/binary"
	"math"

	"github.com/paalgyula/summit/pkg/wow"
)

type Object struct {
	guid wow.GUID

	UpdateData []wow.Packet
	UpdateMask *UpdateMask

	movementFlags wow.MovementFlag

	objectTypeID wow.TypeID
	objectType   wow.TypeMask

	isInWorld bool
	isUpdated bool

	// values holds the update field values as uint32 words.
	// Index corresponds to the UpdateField constants (ObjectFieldGuid, UnitFieldHealth, etc.).
	values []uint32

	updateFlags wow.ObjectUpdateFlags
}

func NewObject() *Object {
	//nolint:exhaustruct
	return &Object{
		objectTypeID: wow.TypeIDObject,
		objectType:   wow.TypeMaskObject,

		isInWorld: false,
		isUpdated: false,
	}
}

// InitValues allocates the values array with the given count.
// Must be called before SetUInt32Value/GetUInt32Value.
func (o *Object) InitValues(count int) {
	o.values = make([]uint32, count)
}

// ValuesCount returns the length of the values array.
func (o *Object) ValuesCount() int {
	return len(o.values)
}

// SetUInt32Value sets a uint32 update field value.
func (o *Object) SetUInt32Value(field UpdateField, val uint32) {
	idx := int(field)
	if idx >= 0 && idx < len(o.values) {
		o.values[idx] = val
	}
}

// GetUInt32Value returns a uint32 update field value.
func (o *Object) GetUInt32Value(field UpdateField) uint32 {
	idx := int(field)
	if idx >= 0 && idx < len(o.values) {
		return o.values[idx]
	}

	return 0
}

// SetFloatValue sets a float32 update field value (stored as uint32 bits).
func (o *Object) SetFloatValue(field UpdateField, val float32) {
	o.SetUInt32Value(field, math.Float32bits(val))
}

// GetFloatValue returns a float32 update field value.
func (o *Object) GetFloatValue(field UpdateField) float32 {
	return math.Float32frombits(o.GetUInt32Value(field))
}

// SetInt32Value sets a signed int32 update field value.
func (o *Object) SetInt32Value(field UpdateField, val int32) {
	o.SetUInt32Value(field, uint32(val))
}

// GetInt32Value returns a signed int32 update field value.
func (o *Object) GetInt32Value(field UpdateField) int32 {
	return int32(o.GetUInt32Value(field))
}

// SetByteValue sets a single byte within a uint32 update field.
// byteIdx is 0-3 (little-endian byte order).
func (o *Object) SetByteValue(field UpdateField, byteIdx uint8, val uint8) {
	idx := int(field)
	if idx < 0 || idx >= len(o.values) {
		return
	}

	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, o.values[idx])
	b[byteIdx] = val
	o.values[idx] = binary.LittleEndian.Uint32(b)
}

// BuildValuesUpdateBlock writes the update mask and values block for SMSG_UPDATE_OBJECT.
// It writes only the fields that differ from the target's current values.
// mask is the UpdateMask indicating which fields are set.
func (o *Object) BuildValuesUpdateBlock(mask *UpdateMask, target *Object) []byte {
	blockCount := mask.GetUpdateBlockCount()

	// UpdateMask (4 bytes per block)
	maskBytes := make([]byte, blockCount*4)
	copy(maskBytes, mask.Mask()[:blockCount*4])

	// Values block (4 bytes per set bit)
	var valuesBytes []byte
	for i := uint32(0); i < blockCount*32; i++ {
		if mask.GetBit(i) && int(i) < len(o.values) {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, o.values[i])
			valuesBytes = append(valuesBytes, b...)
		}
	}

	result := make([]byte, 0, len(maskBytes)+len(valuesBytes))
	result = append(result, maskBytes...)
	result = append(result, valuesBytes...)

	return result
}

// BuildFullUpdateMask creates an UpdateMask with all bits set (for initial create).
func (o *Object) BuildFullUpdateMask() *UpdateMask {
	mask := &UpdateMask{}
	mask.SetCount(uint32(len(o.values)))

	for i := uint32(0); i < uint32(len(o.values)); i++ {
		mask.SetBit(i)
	}

	return mask
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
