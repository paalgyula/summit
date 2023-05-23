package main_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/paalgyula/summit/server/world/data/static"
	"github.com/paalgyula/summit/server/world/packet"
	"github.com/stretchr/testify/assert"
)

func TestBitshift(t *testing.T) {
	sar := packet.ServerAuthResponse{
		StatusCode:           static.AuthOK,
		BillingTimeRemaining: 0x7830,
		BillingPlanFlags:     0,
		BillingTimeRested:    0,
		Expansion:            2,
	}

	bb, err := sar.ToBytes()

	assert.NoError(t, err)
	assert.Len(t, bb, 11)

	fmt.Printf("%s", hex.Dump(bb))
}
