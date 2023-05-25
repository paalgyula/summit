package serworm_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/blizzard/serworm"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	br := serworm.NewBridge("logon.warmane.com:3724", "gmgoofy", "0027472")

	assert.NotNil(t, br)
}
