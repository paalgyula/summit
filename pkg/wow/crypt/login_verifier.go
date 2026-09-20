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
	hash.Write(reverse(SessionKeyBytes(sessionKey)))

	return hash.Sum(nil)
}

// SessionKeyLength is the size of the SRP6 session key K on the wire.
const SessionKeyLength = 40

// SessionKeyBytes left-pads a big-endian session key to its 40-byte wire size.
// big.Int drops leading zero bytes, which would otherwise shift every
// derivation (proof digest, header cipher keys) for 1 in 256 logins.
func SessionKeyBytes(key []byte) []byte {
	if len(key) >= SessionKeyLength {
		return key[len(key)-SessionKeyLength:]
	}
	padded := make([]byte, SessionKeyLength)
	copy(padded[SessionKeyLength-len(key):], key)
	return padded
}
