package wow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuid(t *testing.T) {
	g := NewGUID(CorpseGuid, 0x9739)
	g.PrintRAW()

	assert.Equal(t, CorpseGuid, g.High())

	assert.Equal(t, uint32(0x9739), g.Counter())

	// 0x60000000067cabec
	g = NewPlayerGUID(0x67cabec)
	// g.rawValue = 0x60000000067cabec
	g.PrintRAW()
}
