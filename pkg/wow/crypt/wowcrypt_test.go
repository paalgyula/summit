package crypt_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/stretchr/testify/assert"
)

func TestCrypt(t *testing.T) {
	key := big.NewInt(0)
	key.SetString("218a3599d73b4b21f5c4eead810107f3c5f3eaa7801d609e3adac39239683395d42caa2c36ee79fd", 16)

	crypt, err := crypt.NewWowcrypt(key, 1024)
	assert.NoError(t, err)

	crypt.Skip(1024)

	bb := crypt.Encrypt([]byte("macika elment vadaszni"))
	fmt.Printf("%s", hex.Dump(bb))

	crypt.Reset()
	crypt.Skip(1024)

	bb = crypt.Encrypt(bb)
	fmt.Printf("%s", hex.Dump(bb))
}
