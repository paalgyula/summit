//nolint:gosec
package crypt

import (
	"crypto/sha1"
)

// Generates verifier hash and client seed.
//
// The seeds and key are reversed into scratch copies: reverse() works in place,
// and the client sends the very clientSeed it passes here, so mutating it would
// ship the reversed seed on the wire and break the digest check.
func AuthSessionProof(accountName string, serverSeed, clientSeed []byte, sessionKey []byte) []byte {
	hash := sha1.New()

	hash.Write([]byte(accountName))
	hash.Write([]byte{0, 0, 0, 0}) // padding
	hash.Write(reverseCopy(clientSeed))
	hash.Write(reverseCopy(serverSeed))
	hash.Write(reverseCopy(SessionKeyBytes(sessionKey)))

	return hash.Sum(nil)
}

// reverseCopy returns a reversed copy of b, leaving b untouched.
func reverseCopy(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[len(b)-1-i]
	}

	return out
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
