package crypt_test

import (
	"encoding/hex"
	"testing"

	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/stretchr/testify/assert"
)

// AccountName: TEST
// ClientSeed: 0x31a601d4
// Digest: 5a1bcebd52a7d2934faacaf7aeb0037602610c9c
// ServerSeed: 0x0
// SKey: 7a825336427f9d5f0ce1c45b89dff495764113c1f44721e0e1caa8bacfa7aaf859552e9d6ee04ff2

func TestLoginVerifier(t *testing.T) {
	accountName := "TEST"

	clientSeed, err := hex.DecodeString("31a601d4")
	assert.NoError(t, err)

	digest, err := hex.DecodeString("5a1bcebd52a7d2934faacaf7aeb0037602610c9c")
	serverSeed := []byte{0, 0, 0, 0}

	assert.NoError(t, err)

	sessionKey, err := hex.DecodeString("7a825336427f9d5f0ce1c45b89dff495764113c1f44721e0e1caa8bacfa7aaf859552e9d6ee04ff2")
	assert.NoError(t, err)

	proof := crypt.AuthSessionProof(accountName, serverSeed, clientSeed, sessionKey)

	assert.Equal(t, digest, proof)
}
