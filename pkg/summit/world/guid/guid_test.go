package guid_test

import (
	"fmt"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/guid"
	"github.com/stretchr/testify/assert"
)

func TestGuid(t *testing.T) {
	g := guid.NewGUID(guid.Corpse, 15)

	assert.Equal(t, guid.Corpse, g.High())
	fmt.Printf("%64b\nValue: %d Hex: 0x%x\n", g.RawValue(), g.RawValue(), g.RawValue())

	assert.Equal(t, uint32(15), g.Counter())
}
