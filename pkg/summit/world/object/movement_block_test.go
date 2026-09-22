package object_test

import (
	"encoding/binary"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

func TestWriteMovementBlock_GameObjectRotation(t *testing.T) {
	buf := object.NewUpdateBlockBuffer()
	flags := wow.UpdateFlagLowGUID | wow.UpdateFlagRotation

	object.WriteMovementBlock(buf, flags, &object.MovementBlock{
		LowGUID:  0x2A,
		Rotation: 123456789,
	})

	b := buf.Bytes()

	// Low GUID is a uint32, the packed rotation a trailing int64.
	assert.Len(t, b, 12)
	assert.Equal(t, uint32(0x2A), binary.LittleEndian.Uint32(b[0:4]))
	assert.Equal(t, int64(123456789), int64(binary.LittleEndian.Uint64(b[4:12])))
}
