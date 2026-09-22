package world_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGameObjectQueryResponse(t *testing.T) {
	tmpl := &basedata.GameObjectTemplate{
		Entry:     1002,
		Type:      0,
		DisplayID: 57,
		Name:      "Test Door",
		IconName:  "Door",
		Size:      1.5,
		Data:      [24]int32{0, 1234, 5000},
	}

	pkt := world.BuildGameObjectQueryResponse(1002, tmpl)
	require.Equal(t, wow.ServerGameobjectQueryResponse, pkt.Opcode())

	r := pkt.Reader()

	var entry, goType, displayID uint32
	require.NoError(t, r.Read(&entry))
	require.NoError(t, r.Read(&goType))
	require.NoError(t, r.Read(&displayID))

	assert.Equal(t, uint32(1002), entry)
	assert.Equal(t, uint32(0), goType)
	assert.Equal(t, uint32(57), displayID)

	var name string
	require.NoError(t, r.ReadString(&name))
	assert.Equal(t, "Test Door", name)

	var n2, n3, n4 uint8
	require.NoError(t, r.Read(&n2))
	require.NoError(t, r.Read(&n3))
	require.NoError(t, r.Read(&n4))
	assert.Equal(t, []uint8{0, 0, 0}, []uint8{n2, n3, n4})

	var icon, castBar, unk1 string
	require.NoError(t, r.ReadString(&icon))
	require.NoError(t, r.ReadString(&castBar))
	require.NoError(t, r.ReadString(&unk1))
	assert.Equal(t, "Door", icon)

	var data [24]uint32
	require.NoError(t, r.Read(&data))
	assert.Equal(t, uint32(1234), data[1])
	assert.Equal(t, uint32(5000), data[2])

	var size float32
	require.NoError(t, r.Read(&size))
	assert.InDelta(t, 1.5, size, 1e-6)

	var questItems [6]uint32
	require.NoError(t, r.Read(&questItems))
	assert.Equal(t, [6]uint32{}, questItems)
}

func TestBuildGameObjectQueryResponse_Missing(t *testing.T) {
	pkt := world.BuildGameObjectQueryResponse(4242, nil)

	r := pkt.Reader()

	var entry uint32
	require.NoError(t, r.Read(&entry))
	assert.Equal(t, uint32(4242|0x80000000), entry)
}
