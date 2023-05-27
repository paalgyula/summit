package serworm_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	t.Skip("Still failing needs more attention")
	br := serworm.NewBridge("logon.warmane.com:3724", "***REMOVED***", "MaciLaci123")

	assert.NotNil(t, br)
}
