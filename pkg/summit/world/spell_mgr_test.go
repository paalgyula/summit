package world_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpellMgr_LoadSpells(t *testing.T) {
	mgr := world.NewSpellMgr()
	err := mgr.LoadSpells("../../../dbc")
	require.NoError(t, err)

	si := mgr.GetSpellInfo(686)
	require.NotNil(t, si)
	assert.Equal(t, uint32(686), si.Id)
}
