package object

import (
	"encoding/binary"
	"math"

	"github.com/paalgyula/summit/pkg/wow"
)

// ObjectUpdater is the interface that objects use to register themselves
// for the next update cycle. Maps implement this to collect dirty objects.
type ObjectUpdater interface {
	AddUpdateObject(obj *Object)
	RemoveUpdateObject(obj *Object)
}

type Object struct {
	guid wow.GUID

	UpdateData []wow.Packet
	UpdateMask *UpdateMask

	movementFlags wow.MovementFlag

	objectTypeID wow.TypeID
	objectType   wow.TypeMask

	isInWorld bool
	isUpdated bool

	// updater is set when the object is added to a map. Setters use it
	// to auto-queue the object for the next value-update cycle.
	updater ObjectUpdater

	// values holds the update field values as uint32 words.
	// Index corresponds to the UpdateField constants (ObjectFieldGuid, UnitFieldHealth, etc.).
	values []uint32

	// changesMask tracks which fields have been modified since the last ClearChanges.
	changesMask UpdateMask

	// fieldNotifyFlags are OR'd into visibleFlag for every field that has
	// the corresponding bit set in its per-field flags. This forces certain
	// fields to always be included in updates (matching AzerothCore's
	// _fieldNotifyFlags).
	fieldNotifyFlags uint16

	updateFlags wow.ObjectUpdateFlags

	// visibilityOverrideType controls the visibility distance tier for this object.
	// Matches AzerothCore's _visibilityDistanceOverrideType.
	visibilityOverrideType VisibilityDistanceType
}

func NewObject() *Object {
	//nolint:exhaustruct
	return &Object{
		objectTypeID: wow.TypeIDObject,
		objectType:   wow.TypeMaskObject,

		isInWorld: false,
		isUpdated: false,

		// Dynamic fields (NPC flags, dynamic flags) always go out, like AzerothCore's _fieldNotifyFlags
		fieldNotifyFlags: uint16(UFFlagDynamic),
	}
}

// InitValues allocates the values array with the given count and
// initialises the changes mask. Must be called before SetUInt32Value/GetUInt32Value.
func (o *Object) InitValues(count int) {
	o.values = make([]uint32, count)
	o.changesMask.SetCount(uint32(count))
}

// SetUpdater sets the ObjectUpdater for this object (called when added to a map).
func (o *Object) SetUpdater(u ObjectUpdater) {
	o.updater = u
}

// AddToObjectUpdateIfNeeded queues this object for the next update cycle
// if it hasn't been queued yet. Called from every field setter.
func (o *Object) AddToObjectUpdateIfNeeded() {
	if o.updater != nil && !o.isUpdated {
		o.isUpdated = true
		o.updater.AddUpdateObject(o)
	}
}

// RemoveFromObjectUpdate removes this object from the update queue.
func (o *Object) RemoveFromObjectUpdate() {
	if o.updater != nil && o.isUpdated {
		o.isUpdated = false
		o.updater.RemoveUpdateObject(o)
	}
}

// MarkUpdateSent resets the queued flag after the map drained the object from
// its update set itself, so the next change queues it again.
func (o *Object) MarkUpdateSent() {
	o.isUpdated = false
}

// ClearUpdateMask clears the changes mask and removes the object from the
// update queue. Called after building updates for all visible players.
func (o *Object) ClearUpdateMask() {
	o.changesMask.Clear()
	o.RemoveFromObjectUpdate()
}

// ValuesCount returns the length of the values array.
func (o *Object) ValuesCount() int {
	return len(o.values)
}

// SetUInt32Value sets a uint32 update field value and marks the field as changed.
func (o *Object) SetUInt32Value(field UpdateField, val uint32) {
	idx := int(field)
	if idx >= 0 && idx < len(o.values) {
		if o.values[idx] != val {
			o.values[idx] = val
			o.changesMask.SetBit(uint32(idx))
			o.AddToObjectUpdateIfNeeded()
		}
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

// SetFloatValue sets a float32 update field value (stored as uint32 bits) and marks changed.
func (o *Object) SetFloatValue(field UpdateField, val float32) {
	newBits := math.Float32bits(val)
	idx := int(field)
	if idx >= 0 && idx < len(o.values) {
		if o.values[idx] != newBits {
			o.values[idx] = newBits
			o.changesMask.SetBit(uint32(idx))
			o.AddToObjectUpdateIfNeeded()
		}
	}
}

// GetFloatValue returns a float32 update field value.
func (o *Object) GetFloatValue(field UpdateField) float32 {
	return math.Float32frombits(o.GetUInt32Value(field))
}

// SetInt32Value sets a signed int32 update field value and marks changed.
func (o *Object) SetInt32Value(field UpdateField, val int32) {
	o.SetUInt32Value(field, uint32(val))
}

// GetInt32Value returns a signed int32 update field value.
func (o *Object) GetInt32Value(field UpdateField) int32 {
	return int32(o.GetUInt32Value(field))
}

// SetFlag sets bits in a uint32 update field and marks changed.
func (o *Object) SetFlag(field UpdateField, flags uint32) {
	idx := int(field)
	if idx < 0 || idx >= len(o.values) {
		return
	}

	oldVal := o.values[idx]
	newVal := oldVal | flags

	if oldVal != newVal {
		o.values[idx] = newVal
		o.changesMask.SetBit(uint32(idx))
		o.AddToObjectUpdateIfNeeded()
	}
}

// RemoveFlag clears bits in a uint32 update field and marks changed.
func (o *Object) RemoveFlag(field UpdateField, flags uint32) {
	idx := int(field)
	if idx < 0 || idx >= len(o.values) {
		return
	}

	oldVal := o.values[idx]
	newVal := oldVal & ^flags

	if oldVal != newVal {
		o.values[idx] = newVal
		o.changesMask.SetBit(uint32(idx))
		o.AddToObjectUpdateIfNeeded()
	}
}

// ToggleFlag toggles bits in a uint32 update field and marks changed.
func (o *Object) ToggleFlag(field UpdateField, flags uint32) {
	idx := int(field)
	if idx < 0 || idx >= len(o.values) {
		return
	}

	o.values[idx] ^= flags
	o.changesMask.SetBit(uint32(idx))
	o.AddToObjectUpdateIfNeeded()
}

// HasFlag returns true if the given bits are set in the update field.
func (o *Object) HasFlag(field UpdateField, flags uint32) bool {
	return o.GetUInt32Value(field)&flags != 0
}

// SetByteValue sets a single byte within a uint32 update field and marks changed.
// byteIdx is 0-3 (little-endian byte order).
func (o *Object) SetByteValue(field UpdateField, byteIdx uint8, val uint8) {
	idx := int(field)
	if idx < 0 || idx >= len(o.values) {
		return
	}

	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, o.values[idx])
	b[byteIdx] = val
	newVal := binary.LittleEndian.Uint32(b)

	if o.values[idx] != newVal {
		o.values[idx] = newVal
		o.changesMask.SetBit(uint32(idx))
		o.AddToObjectUpdateIfNeeded()
	}
}

// BuildValuesUpdateBlock writes the update mask and values block for SMSG_UPDATE_OBJECT.
// It writes: uint8 blockCount + mask bytes + values bytes.
// mask is the UpdateMask indicating which fields are set.
func (o *Object) BuildValuesUpdateBlock(mask *UpdateMask, target *Object) []byte {
	blockCount := mask.GetUpdateBlockCount()

	// Total:1 byte blockCount + blockCount*4 mask bytes + variable values bytes
	result := make([]byte, 0, 1+blockCount*4+blockCount*32)

	// blockCount as uint8 (matches AzerothCore's *data << uint8(updateMask.GetBlockCount()))
	result = append(result, byte(blockCount))

	// UpdateMask (4 bytes per block, little-endian)
	for i := uint32(0); i < blockCount; i++ {
		val := uint32(0)
		for b := uint32(0); b < 32; b++ {
			idx := i*32 + b
			if mask.GetBit(idx) {
				val |= 1 << b
			}
		}

		result = append(result,
			byte(val), byte(val>>8), byte(val>>16), byte(val>>24),
		)
	}

	// Values block (4 bytes per set bit)
	for i := uint32(0); i < blockCount*32; i++ {
		if mask.GetBit(i) && int(i) < len(o.values) {
			v := o.values[i]
			result = append(result,
				byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
			)
		}
	}

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

// SetObjectTypeID sets the object type ID.
func (o *Object) SetObjectTypeID(tid wow.TypeID) {
	o.objectTypeID = tid
}

// ObjectType returns the object type mask.
func (o *Object) ObjectType() wow.TypeMask {
	return o.objectType
}

// ObjectTypeID returns the object type ID.
func (o *Object) ObjectTypeID() wow.TypeID {
	return o.objectTypeID
}

// GetVisibilityOverrideType returns the visibility distance override type.
// Matches AzerothCore's WorldObject::GetVisibilityOverrideType.
func (o *Object) GetVisibilityOverrideType() VisibilityDistanceType {
	return o.visibilityOverrideType
}

// SetVisibilityOverrideType sets the visibility distance override type.
// Matches AzerothCore's WorldObject::SetVisibilityDistanceOverride.
func (o *Object) SetVisibilityOverrideType(vd VisibilityDistanceType) {
	o.visibilityOverrideType = vd
}

// IsFarVisible returns true if this object uses the far-visible grid container.
// Matches AzerothCore's WorldObject::IsFarVisible.
func (o *Object) IsFarVisible() bool {
	return o.visibilityOverrideType.IsFarVisible()
}

// ChangesMask returns the change-tracking mask.
func (o *Object) ChangesMask() *UpdateMask {
	return &o.changesMask
}

// HasChanges returns true if any field has been modified since the last ClearChanges.
func (o *Object) HasChanges() bool {
	for _, b := range o.changesMask.Mask() {
		if b != 0 {
			return true
		}
	}

	return false
}

// ClearChanges resets all change bits.
func (o *Object) ClearChanges() {
	o.changesMask.Clear()
}

// FieldNotifyFlags returns the field-notify flags.
func (o *Object) FieldNotifyFlags() uint16 {
	return o.fieldNotifyFlags
}

// SetFieldNotifyFlag sets one or more field-notify flags.
func (o *Object) SetFieldNotifyFlag(flag uint16) {
	o.fieldNotifyFlags |= flag
}

// RemoveFieldNotifyFlag removes one or more field-notify flags.
func (o *Object) RemoveFieldNotifyFlag(flag uint16) {
	o.fieldNotifyFlags &^= flag
}

// GetUpdateFieldDataForTesting is a test helper that exposes the visibility
// resolution logic without requiring a full Player. It mirrors AzerothCore's
// Object::GetUpdateFieldData.
func GetUpdateFieldDataForTesting(obj, target *Object) uint32 {
	visibleFlag := uint32(UFFlagPublic)

	if obj == target {
		visibleFlag |= UFFlagPrivate
	}

	return visibleFlag
}

// BuildFilteredUpdateMask creates an UpdateMask with only the fields that are
// visible to the target viewer. If isSelf is true, PRIVATE fields are included.
// This is used for CREATE_OBJECT where all visible fields must be sent.
func (o *Object) BuildFilteredUpdateMask(target *Object, isSelf bool) *UpdateMask {
	visibleFlag := uint32(UFFlagPublic)
	if isSelf {
		visibleFlag |= UFFlagPrivate

		// For items isSelf means the target owns it (Object::GetUpdateFieldData)
		if o.objectTypeID == wow.TypeIDItem || o.objectTypeID == wow.TypeIDContainer {
			visibleFlag |= UFFlagOwner | UFFlagItemOwner
		}
	}

	mask := &UpdateMask{}
	mask.SetCount(uint32(len(o.values)))

	for i := uint32(0); i < uint32(len(o.values)); i++ {
		f := o.fieldFlagAt(i)
		if f&visibleFlag != 0 || (o.fieldNotifyFlags != 0 && f&uint32(o.fieldNotifyFlags) != 0) {
			mask.SetBit(i)
		}
	}

	return mask
}

// BuildIncrementalUpdateMask creates an UpdateMask that combines the changesMask
// with visibility filtering. Only changed fields that are visible to the target
// are included. Fields where fieldNotifyFlags match are always included.
// Used for UPDATETYPE_VALUES (incremental updates).
func (o *Object) BuildIncrementalUpdateMask(target *Object) *UpdateMask {
	var visibleFlag uint32 = UFFlagPublic
	if target == o {
		visibleFlag |= UFFlagPrivate
	}

	mask := &UpdateMask{}
	mask.SetCount(uint32(len(o.values)))

	for i := uint32(0); i < uint32(len(o.values)); i++ {
		f := o.fieldFlagAt(i)

		// Include if field has a notify flag set (always send these)
		if o.fieldNotifyFlags != 0 && f&uint32(o.fieldNotifyFlags) != 0 {
			mask.SetBit(i)
			continue
		}

		// Include if changed AND visible to target
		if o.changesMask.GetBit(i) && f&visibleFlag != 0 {
			mask.SetBit(i)
		}
	}

	return mask
}

// fieldFlagAt returns the visibility flags of field i: the shared object
// header fields come from ObjectFieldFlags, the rest from the type's table
// (which is indexed relative to ObjectEnd).
func (o *Object) fieldFlagAt(i uint32) uint32 {
	if i < uint32(ObjectEnd) {
		return ObjectFieldFlags[i]
	}

	fieldFlags := o.getFieldFlags()
	fi := int(i) - int(ObjectEnd)

	if fi < len(fieldFlags) {
		return fieldFlags[fi]
	}

	return 0
}

// getFieldFlags returns the per-field visibility flags array for this object's type.
func (o *Object) getFieldFlags() []uint32 {
	switch o.objectTypeID {
	case wow.TypeIDUnit, wow.TypeIDPlayer:
		return UnitUpdateFieldFlags[:]

	case wow.TypeIDItem, wow.TypeIDContainer:
		return ItemUpdateFieldFlags[:]

	case wow.TypeIDGameObject:
		return GameObjectFieldFlags[:]

	case wow.TypeIDDynamicoObject:
		return DynamicObjectFieldFlags[:]

	case wow.TypeIDCorpse:
		return CorpseUpdateFieldFlags[:]

	default:
		return ObjectFieldFlags[:]
	}
}
