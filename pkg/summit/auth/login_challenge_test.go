package auth_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/stretchr/testify/assert"
)

const testLoginPacket = "WoW\x00\x02\x04\x03\x9e!68x\x00niW\x00SUne<\x00\x00\x00\x7f\x00\x00\x01\x04TEST"

func TestLoginChallenge(t *testing.T) {
	data := auth.RData{
		Data: []byte(testLoginPacket),
	}

	var lp auth.ClientLoginChallenge

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
}

func TestLoginSession(t *testing.T) {
	c := crypt.NewSRP6(7, 3, big.NewInt(0))
	B := c.GenerateClientPubkey()
	salt := c.RandomSalt()

	lc := auth.ServerLoginChallenge{
		Status:  auth.ChallengeStatusSuccess,
		B:       *B,
		Salt:    *salt,
		SaltCRC: make([]byte, 16),
		G:       uint8(c.GValue()),
		N:       *c.N(),
	}

	fmt.Printf("%+v Size: %d", lc, len(lc.B.Bytes()))
}
