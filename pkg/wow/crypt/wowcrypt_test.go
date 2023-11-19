package crypt_test

import (
	"math/big"
	"testing"

	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/stretchr/testify/assert"
)

func TestWoWCrypt(t *testing.T) {
	testString := []byte("Macilaci Malnazik")

	key := new(big.Int)
	key.SetString("218a3599d73b4b21f5c4eead810107f3c5f3eaa7801d609e3adac39239683395d42caa2c36ee79fd", 16)

	serverCrypt, err := crypt.NewServerWowcrypt(key, 1024)
	assert.NoError(t, err)

	clientCrypt, err := crypt.NewClientWoWCrypt(key, 1024)
	assert.NoError(t, err)

	t.Run("TestServerToClient", func(t *testing.T) {
		encrypted := serverCrypt.Encrypt(testString)

		decrypted := clientCrypt.Decrypt(encrypted)

		// Server to Client encrypt-decrypt
		assert.Equal(t, decrypted, testString)
	})

	t.Run("TestClientToServer", func(t *testing.T) {
		enc := clientCrypt.Encrypt(testString)
		dec := serverCrypt.Decrypt(enc)

		// Client to Server encrypt-decrypt
		assert.Equal(t, testString, dec)
	})
}
