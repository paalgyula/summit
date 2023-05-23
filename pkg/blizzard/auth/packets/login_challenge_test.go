package packets_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/blizzard/auth/packets"
	"github.com/stretchr/testify/assert"
)

// 0      4   5   6   7   8
const testLoginPacket = "WoW\x00\x02\x04\x03\x9e!68x\x00niW\x00SUne<\x00\x00\x00\x7f\x00\x00\x01\x04TEST"

func TestLoginChallenge(t *testing.T) {
	data := packets.RData{
		Data: []byte(testLoginPacket),
	}

	var lp packets.ClientLoginChallenge
	data.Unmarshal(&lp)

	assert.Equal(t, lp.GameName, "WoW\x00")

	assert.Equal(t, lp.Version[0], uint8(2))
	assert.Equal(t, lp.Version[1], uint8(4))
	assert.Equal(t, lp.Version[2], uint8(3))

	assert.Equal(t, uint16(8606), lp.Build)

	assert.Equal(t, "68x\x00", lp.Platform)
	assert.Equal(t, "niW\x00", lp.OS)
	assert.Equal(t, "SUne", lp.Locale)

	assert.EqualValues(t, uint8(0x7f), lp.IP[0])
	assert.EqualValues(t, uint8(0x0), lp.IP[1])
	assert.EqualValues(t, uint8(0x0), lp.IP[2])
	assert.EqualValues(t, uint8(0x1), lp.IP[3])

	assert.Equal(t, "TEST", lp.AccountName)

	t.Run("TestRemarshal", func(t *testing.T) {
		assert.EqualValues(t, []byte(testLoginPacket), lp.MarshalPacket())
	})

	// fmt.Printf("%s", hex.Dump(p.MarshalPacket()))
}
