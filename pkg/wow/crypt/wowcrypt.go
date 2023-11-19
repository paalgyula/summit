//nolint:all
package crypt

import (
	"crypto/hmac"
	"crypto/rc4"
	"crypto/sha1"
	"errors"
	"fmt"
	"math/big"

	"github.com/paalgyula/summit/pkg/wow"
)

// CryptRecvLength the length of the cryptable header.
const CryptRecvLength = 6

// CryptSendLength the length of the outbound cryptable header.
const CryptSendLength = 4

var (
	s = []byte{0xC2, 0xB3, 0x72, 0x3C, 0xC6, 0xAE, 0xD9, 0xB5, 0x34, 0x3C, 0x53, 0xEE, 0x2F, 0x43, 0x67, 0xCE}
	r = []byte{0xCC, 0x98, 0xAE, 0x04, 0xE8, 0x97, 0xEA, 0xCA, 0x12, 0xDD, 0xC0, 0x93, 0x42, 0x91, 0x53, 0x57}
)

const digestLength = 20

// ErrSizeNotMach the error when the digest size is not 20 bytes long.
var ErrSizeNotMach = errors.New("digest size is not 20 bytes long")

// WowCrypt is a wrapper for rc4 ciphers. This crypter can be initialized with
// NewWowcrypt constructor.
type WowCrypt struct {
	encoder *rc4.Cipher
	decoder *rc4.Cipher

	encKey []byte
	decKey []byte
}

// NewServerWowcrypt initializes the wow packet header crypter. The key should be the session key
// The session key should be 40 bytes long.
// which has been created on the auth session packet. The default skip for WOTLK client is 1024.
func NewServerWowcrypt(key *big.Int, skip int) (*WowCrypt, error) {
	if len(key.Bytes()) != 40 {
		panic("the crypt key should be 40 bytes long")
	}

	wc := new(WowCrypt)

	// Encoder setup
	h := hmac.New(sha1.New, r) // r -> server to client
	_, _ = h.Write(wow.ReverseBytes(key.Bytes()))
	wc.encKey = h.Sum(nil)

	if h.Size() != digestLength {
		return nil, ErrSizeNotMach
	}

	// Decoder setup
	h = hmac.New(sha1.New, s) // s -> client to server
	_, _ = h.Write(wow.ReverseBytes(key.Bytes()))
	wc.decKey = h.Sum(nil)

	if h.Size() != digestLength {
		return nil, ErrSizeNotMach
	}

	if err := wc.Reset(); err != nil { // Initializes the ciphers with the keys.
		return nil, fmt.Errorf("cannot initialize ciphers: %w", err)
	}
	wc.Skip(skip)

	return wc, nil
}

// NewClientWoWCrypt initializes a client side crypter. The only difference from
// the server crypter is the encoder/decoder key setup.
func NewClientWoWCrypt(key *big.Int, skip int) (*WowCrypt, error) {
	if len(key.Bytes()) != 40 {
		panic("the crypt key should be 40 bytes long")
	}

	wc := new(WowCrypt)

	// Encoder setup
	h := hmac.New(sha1.New, s) // s -> client to server
	_, _ = h.Write(wow.ReverseBytes(key.Bytes()))
	wc.encKey = h.Sum(nil)

	if h.Size() != digestLength {
		return nil, ErrSizeNotMach
	}

	// Decoder setup
	h = hmac.New(sha1.New, r) // r -> server to client
	_, _ = h.Write(wow.ReverseBytes(key.Bytes()))
	wc.decKey = h.Sum(nil)

	if h.Size() != digestLength {
		return nil, ErrSizeNotMach
	}

	if err := wc.Reset(); err != nil { // Initializes the ciphers with the keys.
		return nil, fmt.Errorf("cannot initialize ciphers: %w", err)
	}
	wc.Skip(skip)

	return wc, nil
}

// Skip skips n bytes in both encrypter and decrypter.
func (wc *WowCrypt) Skip(n int) {
	skip := make([]byte, n)

	wc.Encrypt(skip)
	wc.Decrypt(skip)
}

// This method will jumps back to the beginning of the stream again.
func (wc *WowCrypt) Reset() error {
	var err error

	wc.decoder, err = rc4.NewCipher(wc.decKey)
	if err != nil {
		return fmt.Errorf("crypt.NewWowCrypt: %w", err)
	}

	wc.encoder, err = rc4.NewCipher(wc.encKey)
	if err != nil {
		return fmt.Errorf("crypt.NewWowCrypt: %w", err)
	}

	return nil
}

// Encrypt uses the encoder to encode the given data.
func (wc *WowCrypt) Encrypt(data []byte) []byte {
	bb := make([]byte, len(data))
	wc.encoder.XORKeyStream(bb, data)

	return bb
}

// Decrypt uses decoder to convert back the encrypted data.
func (wc *WowCrypt) Decrypt(data []byte) []byte {
	bb := make([]byte, len(data))
	wc.decoder.XORKeyStream(bb, data)

	return bb
}
