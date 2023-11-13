//nolint:gosec
package crypt

import (
	"crypto/sha1"
)

// Generates verifier hash and client seed.
func AuthSessionProof(accountName string, serverSeed, clientSeed []byte, sessionKey []byte) []byte {
	hash := sha1.New()

	hash.Write([]byte(accountName))
	hash.Write([]byte{0, 0, 0, 0}) // padding
	hash.Write(reverse(clientSeed))
	hash.Write(reverse(serverSeed))
	hash.Write(reverse(sessionKey))

	return hash.Sum(nil)
}
