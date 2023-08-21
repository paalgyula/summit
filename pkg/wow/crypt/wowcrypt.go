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

// WowCrypt is a wrapper for rc4 ciphers.
type WowCrypt struct {
	encoder *rc4.Cipher
	decoder *rc4.Cipher

	encKey []byte
	decKey []byte
}

func NewWowcrypt(key *big.Int) (*WowCrypt, error) {
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

	wc.Reset() // Initializes the ciphers with the keys.
	wc.Skip(1024)

	return wc, nil
}

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
		return fmt.Errorf("crypt.NewWowcrypt: %w", err)
	}

	wc.encoder, err = rc4.NewCipher(wc.encKey)
	if err != nil {
		return fmt.Errorf("crypt.NewWowcrypt: %w", err)
	}

	return nil
}

func (wc *WowCrypt) Encrypt(data []byte) []byte {
	bb := make([]byte, len(data))
	wc.encoder.XORKeyStream(bb, data)

	return bb
}

func (wc *WowCrypt) Decrypt(data []byte) []byte {
	bb := make([]byte, len(data))
	wc.decoder.XORKeyStream(bb, data)

	return bb
}
