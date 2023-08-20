package auth_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/stretchr/testify/assert"
)

// s:  9398c11e0e7128c7a56e3fde45b418744ffe9c7f41aaed48ac27e62d3700e223
// K:  6c7d263656772ffbfaedecc5f50367a69d41d8ea62819ee95c88784c2e5e67b23ce343256ae95891
// B:  2d293798fdffb5e93c7367f4930437061993767d566ec5dab6b91c9d8375f835
// M:  503e5d93085afd3c51c836508c2cc20b6702e32b
// V:  3e3f49a5a14a43b870f8de5534e318c63394738c364a71f205a8ba277bb56ff6

func TestMarshalPacket(t *testing.T) {
	// pkt := new(ClientLoginProof)
	var s, B big.Int

	var Mc, Ms, K *big.Int

	_, _ = Ms, K

	s.SetString("9398c11e0e7128c7a56e3fde45b418744ffe9c7f41aaed48ac27e62d3700e223", 16)
	B.SetString("2d293798fdffb5e93c7367f4930437061993767d566ec5dab6b91c9d8375f835", 16)

	c := crypt.NewSRP6(7, 3, big.NewInt(0))
	A := c.GenerateClientPubkey()

	_, Mc = c.CalculateClientSessionKey(&s, &B, "TEST", "TEST")

	clp := auth.ClientLoginProof{
		A:             *A,
		M:             *Mc,
		CRCHash:       []byte{},
		NumberOfKeys:  0,
		SecurityFlags: 0,
	}

	clpbytes := clp.MarshalPacket()
	// reader := bytes.NewReader(clpbytes)

	clp.UnmarshalPacket(wow.PacketData(clpbytes))

	assert.True(t, A.Cmp(&clp.A) == 0)
	assert.True(t, Mc.Cmp(&clp.M) == 0)

	slp := &auth.ServerLoginChallenge{
		Status:  0,
		B:       B,
		Salt:    s,
		SaltCRC: make([]byte, 16),
		G:       7,
		N:       *c.N(),
	}

	t.Run("MarshalPacket", func(t *testing.T) {
		packetBytes := slp.MarshalPacket()

		slp = new(auth.ServerLoginChallenge)
		slp.ReadPacket(bytes.NewReader(packetBytes))

		assert.True(t, slp.B.Cmp(&B) == 0)
		assert.True(t, slp.Salt.Cmp(&s) == 0)
		assert.EqualValues(t, slp.G, 7)
		assert.True(t, slp.N.Cmp(c.N()) == 0)
	})
}
