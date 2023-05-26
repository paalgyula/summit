package serworm_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	br := serworm.NewBridge("logon.warmane.com:3724", "***REMOVED***", "***REMOVED***")

	assert.NotNil(t, br)
}
