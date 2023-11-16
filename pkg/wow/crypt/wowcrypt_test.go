package crypt_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/stretchr/testify/assert"
)

const testString = "abcdef"

// ! TODO: write proper tests for crypt/decrypt

func TestCrypt(t *testing.T) {
	key := new(big.Int)
	key.SetString("218a3599d73b4b21f5c4eead810107f3c5f3eaa7801d609e3adac39239683395d42caa2c36ee79fd", 16)

	encrypt, err := crypt.NewWowcrypt(key, 1024)
	assert.NoError(t, err)

	enc := encrypt.Encrypt([]byte(testString))

	err = encrypt.Reset()
	assert.NoError(t, err)

	encrypt.Skip(1024)

	bb2 := encrypt.Encrypt(enc)
	fmt.Printf("%s", hex.Dump(bb2))
}
