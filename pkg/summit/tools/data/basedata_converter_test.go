package data_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/tools/data"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

func TestSpawnData(t *testing.T) {
	converter := data.NewConverter("../../../../dbc")

	sdata := converter.LoadSpawnData()
	assert.Len(t, sdata, 170)

	spawn := converter.LookupSpawn(uint8(wow.RaceGnome), uint8(wow.ClassWarior))
	assert.NotNil(t, spawn)
	assert.Equal(t, spawn.X, float32(-6240.32))
	assert.Equal(t, spawn.Y, float32(331.033))
}
