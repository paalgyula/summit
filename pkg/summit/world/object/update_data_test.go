package object_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateData_BuildPacket_Empty(t *testing.T) {
	ud := object.NewUpdateData()
	pkt := ud.BuildPacket()

	// Empty: blockCount=0
	r := pkt.Reader()
	var blockCount uint32
	_ = r.Read(&blockCount)
	assert.Equal(t, uint32(0), blockCount)
}

func TestUpdateData_BuildPacket_SingleBlock(t *testing.T) {
	ud := object.NewUpdateData()

	// Fake update block: just a few bytes
	block := []byte{0x01, 0x02, 0x03, 0x04}
	ud.AddUpdateBlock(block)

	pkt := ud.BuildPacket()
	r := pkt.Reader()

	var blockCount uint32
	_ = r.Read(&blockCount)
	assert.Equal(t, uint32(1), blockCount)

	data, _ := r.ReadNBytes(4)
	assert.Equal(t, block, data)
}

func TestUpdateData_BuildPacket_MultipleBlocks(t *testing.T) {
	ud := object.NewUpdateData()

	ud.AddUpdateBlock([]byte{0xAA, 0xBB})
	ud.AddUpdateBlock([]byte{0xCC, 0xDD, 0xEE})

	pkt := ud.BuildPacket()
	r := pkt.Reader()

	var blockCount uint32
	_ = r.Read(&blockCount)
	assert.Equal(t, uint32(2), blockCount)

	d1, _ := r.ReadNBytes(2)
	assert.Equal(t, []byte{0xAA, 0xBB}, d1)

	d2, _ := r.ReadNBytes(3)
	assert.Equal(t, []byte{0xCC, 0xDD, 0xEE}, d2)
}

func TestUpdateData_BuildPacket_WithOutOfRange(t *testing.T) {
	ud := object.NewUpdateData()

	// Add a normal block
	ud.AddUpdateBlock([]byte{0x01})

	// Add out-of-range GUID
	guid := wow.NewPlayerGUID(42)
	ud.AddOutOfRangeGUID(guid)

	pkt := ud.BuildPacket()
	r := pkt.Reader()

	var blockCount uint32
	_ = r.Read(&blockCount)
	// blockCount should be 2:1 normal + 1 out-of-range
	assert.Equal(t, uint32(2), blockCount)

	// First block: out-of-range type
	var updateType uint8
	_ = r.Read(&updateType)
	assert.Equal(t, uint8(wow.UpdateTypeOutOfRangeObjects), updateType)

	// Count of GUIDs
	var guidCount uint32
	_ = r.Read(&guidCount)
	assert.Equal(t, uint32(1), guidCount)

	// Then the normal block data
	d1, _ := r.ReadNBytes(1)
	assert.Equal(t, []byte{0x01}, d1)
}

func TestUpdateData_HasData(t *testing.T) {
	ud := object.NewUpdateData()
	assert.False(t, ud.HasData())

	ud.AddUpdateBlock([]byte{0x01})
	assert.True(t, ud.HasData())
}

func TestUpdateData_Clear(t *testing.T) {
	ud := object.NewUpdateData()
	ud.AddUpdateBlock([]byte{0x01})
	ud.AddOutOfRangeGUID(wow.NewPlayerGUID(1))

	ud.Clear()
	assert.False(t, ud.HasData())
}

func TestUpdateData_OpCode(t *testing.T) {
	ud := object.NewUpdateData()
	pkt := ud.BuildPacket()
	assert.Equal(t, int(wow.ServerUpdateObject), pkt.OpCode())
}

func TestBuildValuesUpdate_VisibilityFiltering(t *testing.T) {
	// Create two objects: source (the object being viewed) and target (the viewer)
	source := object.NewObject()
	source.InitValues(int(object.PlayerEnd))
	source.SetObjectType(wow.TypeMaskUnit)
	source.SetObjectTypeID(wow.TypeIDUnit)

	target := object.NewObject()
	target.InitValues(int(object.PlayerEnd))
	target.SetObjectType(wow.TypeMaskUnit)
	target.SetObjectTypeID(wow.TypeIDUnit)

	// Set health (PUBLIC) — visible to everyone
	source.SetUInt32Value(object.UnitFieldHealth, 500)

	// Set critter (PRIVATE) — only visible to self
	source.SetUInt32Value(object.UnitFieldCritter, 999)

	// Build values update for OTHER player (no PRIVATE access)
	mask := source.BuildFilteredUpdateMask(target, false)

	// Health should be set (PUBLIC)
	healthIdx := uint32(object.UnitFieldHealth)
	assert.True(t, mask.GetBit(healthIdx), "health should be visible to other players")

	// Critter should NOT be set (PRIVATE, and target is not self)
	critterIdx := uint32(object.UnitFieldCritter)
	assert.False(t, mask.GetBit(critterIdx), "critter should NOT be visible to other players")
}

func TestBuildValuesUpdate_SelfSeesPrivate(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(int(object.PlayerEnd))
	obj.SetObjectType(wow.TypeMaskUnit)
	obj.SetObjectTypeID(wow.TypeIDUnit)

	obj.SetUInt32Value(object.UnitFieldHealth, 500)
	obj.SetUInt32Value(object.UnitFieldCritter, 999)

	// Build for self — should see PRIVATE fields
	mask := obj.BuildFilteredUpdateMask(obj, true)

	healthIdx := uint32(object.UnitFieldHealth)
	assert.True(t, mask.GetBit(healthIdx))

	critterIdx := uint32(object.UnitFieldCritter)
	assert.True(t, mask.GetBit(critterIdx), "self should see PRIVATE fields")
}

func TestBuildValuesUpdate_IncrementalUsesChangesMask(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(int(object.UnitEnd))
	obj.SetObjectType(wow.TypeMaskUnit)
	obj.SetObjectTypeID(wow.TypeIDUnit)

	// Set some PUBLIC fields, then clear changes
	// UnitFieldHealth = ObjectEnd + 0x12 (PUBLIC)
	// UnitFieldMaxhealth = ObjectEnd + 0x1A (PUBLIC)
	obj.SetUInt32Value(object.UnitFieldHealth, 100)
	obj.SetUInt32Value(object.UnitFieldMaxhealth, 200)
	obj.ClearChanges()

	// Now change only health
	obj.SetUInt32Value(object.UnitFieldHealth, 999)

	// Build incremental mask (uses changesMask)
	mask := obj.BuildIncrementalUpdateMask(nil)

	assert.True(t, mask.GetBit(uint32(object.UnitFieldHealth)), "changed field should be set")
	assert.False(t, mask.GetBit(uint32(object.UnitFieldMaxhealth)), "unchanged field should NOT be set")
}

func TestBuildValuesUpdate_IncrementalRespectsVisibility(t *testing.T) {
	source := object.NewObject()
	source.InitValues(int(object.PlayerEnd))
	source.SetObjectType(wow.TypeMaskUnit)
	source.SetObjectTypeID(wow.TypeIDUnit)

	target := object.NewObject()
	target.InitValues(int(object.PlayerEnd))
	target.SetObjectType(wow.TypeMaskUnit)
	target.SetObjectTypeID(wow.TypeIDUnit)

	// Change PUBLIC and PRIVATE fields
	source.SetUInt32Value(object.UnitFieldHealth, 500)  // PUBLIC
	source.SetUInt32Value(object.UnitFieldCritter, 999) // PRIVATE

	// Build incremental mask for OTHER player
	mask := source.BuildIncrementalUpdateMask(target)

	healthIdx := uint32(object.UnitFieldHealth)
	assert.True(t, mask.GetBit(healthIdx), "PUBLIC changed field should be visible")

	critterIdx := uint32(object.UnitFieldCritter)
	assert.False(t, mask.GetBit(critterIdx), "PRIVATE changed field should NOT be visible to others")
}

func TestBuildValuesUpdateBlock_BinaryFormat(t *testing.T) {
	obj := object.NewObject()
	obj.InitValues(int(object.UnitEnd))
	obj.SetObjectType(wow.TypeMaskUnit)
	obj.SetObjectTypeID(wow.TypeIDUnit)

	obj.SetUInt32Value(object.UnitFieldHealth, 0xDEADBEEF)
	obj.SetUInt32Value(object.UnitFieldLevel, 42)

	// Build for self (all fields visible)
	mask := obj.BuildFilteredUpdateMask(obj, true)
	block := obj.BuildValuesUpdateBlock(mask, nil)

	// Parse: first byte is blockCount (uint8)
	blockCount := uint8(block[0])
	assert.True(t, blockCount > 0, "should have at least 1 block")

	// Find the two values we set in the values section
	// The mask+values start at offset 1. Each block is 4 bytes mask + N*4 values.
	// Since we used BuildFilteredUpdateMask, we need to find our fields.
	reader := wow.NewPacketReader(block[1:]) // skip blockCount
	var maskVal uint32
	_ = reader.Read(&maskVal)

	// Health is at index UnitFieldHealth (relative to ObjectEnd = 0x12 = 18)
	// Level is at index UnitFieldLevel (relative to ObjectEnd = 0x30 = 48)
	healthBit := uint32(object.UnitFieldHealth)

	// The mask should cover enough blocks for both bits
	assert.True(t, maskVal&(1<<(healthBit%32)) != 0, "health bit should be set in mask")
}

func TestUpdateData_BuildPacket_OpCode(t *testing.T) {
	ud := object.NewUpdateData()
	pkt := ud.BuildPacket()
	require.NotNil(t, pkt)
	assert.Equal(t, int(wow.ServerUpdateObject), pkt.OpCode())
}
