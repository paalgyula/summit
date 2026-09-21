package object_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObject_InitValues_SetsChangesMaskCount(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(128)

	mask := obj.ChangesMask()
	assert.NotNil(t, mask)
	assert.Equal(t, uint32(128), mask.Count())
}

func TestObject_SetUInt32Value_TracksChanges(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	// Initially no changes
	assert.False(t, obj.HasChanges())

	// Set a value — should mark bit 10 as changed
	obj.SetUInt32Value(object.UpdateField(10), 42)
	assert.True(t, obj.HasChanges())
	assert.True(t, obj.ChangesMask().GetBit(10))

	// Value 11 should NOT be marked
	assert.False(t, obj.ChangesMask().GetBit(11))
}

func TestObject_SetUInt32Value_SameValue_NoChange(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	obj.SetUInt32Value(object.UpdateField(10), 42)
	obj.ClearChanges()

	// Setting the same value again should NOT set the change bit
	obj.SetUInt32Value(object.UpdateField(10), 42)
	assert.False(t, obj.HasChanges())
}

func TestObject_SetFloatValue_TracksChanges(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	obj.SetFloatValue(object.UpdateField(5), 3.14)
	assert.True(t, obj.HasChanges())
	assert.True(t, obj.ChangesMask().GetBit(5))
}

func TestObject_SetInt32Value_TracksChanges(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	obj.SetInt32Value(object.UpdateField(20), -100)
	assert.True(t, obj.HasChanges())
	assert.True(t, obj.ChangesMask().GetBit(20))
}

func TestObject_SetByteValue_TracksChanges(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	obj.SetByteValue(object.UpdateField(8), 2, 0xFF)
	assert.True(t, obj.HasChanges())
	assert.True(t, obj.ChangesMask().GetBit(8))
}

func TestObject_ClearChanges(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	obj.SetUInt32Value(object.UpdateField(10), 42)
	assert.True(t, obj.HasChanges())

	obj.ClearChanges()
	assert.False(t, obj.HasChanges())
	assert.False(t, obj.ChangesMask().GetBit(10))
}

func TestObject_MultipleFieldChanges(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(128)

	obj.SetUInt32Value(object.UpdateField(0), 100)
	obj.SetUInt32Value(object.UpdateField(32), 200)
	obj.SetUInt32Value(object.UpdateField(63), 300)

	assert.True(t, obj.ChangesMask().GetBit(0))
	assert.True(t, obj.ChangesMask().GetBit(32))
	assert.True(t, obj.ChangesMask().GetBit(63))
	assert.False(t, obj.ChangesMask().GetBit(1))
	assert.False(t, obj.ChangesMask().GetBit(31))
}

func TestObject_FieldNotifyFlags(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	// Dynamic fields are always notified, like AzerothCore's Object::Object()
	base := uint16(object.UFFlagDynamic)
	assert.Equal(t, base, obj.FieldNotifyFlags())

	obj.SetFieldNotifyFlag(0x01)
	assert.Equal(t, base|0x01, obj.FieldNotifyFlags())

	obj.SetFieldNotifyFlag(0x04)
	assert.Equal(t, base|0x05, obj.FieldNotifyFlags())

	obj.RemoveFieldNotifyFlag(0x01)
	assert.Equal(t, base|0x04, obj.FieldNotifyFlags())
}

func TestUpdateMask_SetAndGetBit(t *testing.T) {
	mask := &object.UpdateMask{}
	mask.SetCount(96) // 3 blocks

	mask.SetBit(0)
	mask.SetBit(31)
	mask.SetBit(32)
	mask.SetBit(95)

	assert.True(t, mask.GetBit(0))
	assert.True(t, mask.GetBit(31))
	assert.True(t, mask.GetBit(32))
	assert.True(t, mask.GetBit(95))
	assert.False(t, mask.GetBit(1))
	assert.False(t, mask.GetBit(33))
}

func TestUpdateMask_GetUpdateBlockCount(t *testing.T) {
	mask := &object.UpdateMask{}
	mask.SetCount(128) // 4 blocks

	// No bits set — block count should be 0
	assert.Equal(t, uint32(0), mask.GetUpdateBlockCount())

	// Set bit in block 0
	mask.SetBit(5)
	assert.Equal(t, uint32(1), mask.GetUpdateBlockCount())

	// Set bit in block 2 (skipping block 1)
	mask.SetBit(70)
	assert.Equal(t, uint32(3), mask.GetUpdateBlockCount())

	// Set bit in block 3
	mask.SetBit(100)
	assert.Equal(t, uint32(4), mask.GetUpdateBlockCount())
}

func TestUpdateMask_Clear(t *testing.T) {
	mask := &object.UpdateMask{}
	mask.SetCount(64)

	mask.SetBit(10)
	mask.SetBit(30)
	require.True(t, mask.GetBit(10))

	mask.Clear()
	assert.False(t, mask.GetBit(10))
	assert.False(t, mask.GetBit(30))
}

func TestUpdateMask_BlockCount(t *testing.T) {
	mask := &object.UpdateMask{}
	mask.SetCount(96) // should be 3 blocks (96/32)

	assert.Equal(t, uint32(3), mask.BlockCount())
	assert.Equal(t, uint32(96), mask.Count())
}

func TestObject_BuildFullUpdateMask(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(64)

	mask := obj.BuildFullUpdateMask()
	require.NotNil(t, mask)

	for i := uint32(0); i < 64; i++ {
		assert.True(t, mask.GetBit(i), "bit %d should be set", i)
	}
}

func TestObject_BuildValuesUpdateBlock(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(8)

	obj.SetUInt32Value(object.UpdateField(0), 0xDEADBEEF)
	obj.SetUInt32Value(object.UpdateField(3), 42)

	mask := obj.BuildFullUpdateMask()
	block := obj.BuildValuesUpdateBlock(mask, nil)

	// block should contain:1 byte blockCount + 4 bytes mask (1 block) + 8*4 bytes values
	assert.Equal(t, 1+4+8*4, len(block))
	// First byte is blockCount
	assert.Equal(t, byte(1), block[0])
}

func TestObject_SetValues_OutOfRange(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(16)

	// Should not panic on out-of-range index
	obj.SetUInt32Value(object.UpdateField(20), 999)
	obj.SetFloatValue(object.UpdateField(-1), 1.0)

	// Should still have no changes (out-of-range writes are ignored)
	assert.False(t, obj.HasChanges())
}
