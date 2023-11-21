package blizzard_test

import (
	"os"
	"testing"

	"github.com/paalgyula/summit/pkg/blizzard"
	"github.com/stretchr/testify/assert"
)

func TestReadHorseM2(t *testing.T) {
	f, err := os.Open("test/horse.m2")
	assert.NoError(t, err)

	r, err := blizzard.NewM2Reader(f)
	assert.NoError(t, err)

	_ = r
}
