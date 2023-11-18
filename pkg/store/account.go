package store

import (
	"errors"
	"fmt"
	"math/big"
	"time"
)

var ErrInvalidHexNumber = errors.New("invalid hex number")

// Initializes a new account object.
func NewAccount(name, salt, verifier string) (*Account, error) {
	v, succ := new(big.Int).SetString(verifier, 16)
	if !succ {
		return nil, fmt.Errorf("cannot convert verifier: %w", ErrInvalidHexNumber)
	}

	s, succ := new(big.Int).SetString(salt, 16)
	if !succ {
		return nil, fmt.Errorf("store.NewAccount verifier: %w", ErrInvalidHexNumber)
	}

	return &Account{
		Name:     name,
		Verifier: v,
		Salt:     s,
	}, nil
}

type AccountBan struct {
	BanReason string
	BannedAt  string
	BannedBy  string // Username maybe?
	Expires   *time.Time
}

type Account struct {
	// Name username of the account
	// Deprecated: don't use it this will be replaced by the email address completely.
	Name string

	// Email address of the account owner.
	Email string

	// CreatedAt account creation time.
	CreatedAt time.Time

	// LastLogin latest login.
	LastLogin *time.Time

	// Activated Unused yet.
	Activated bool

	// Verifier SPR6 verifier of the password.
	Verifier *big.Int

	// Salt is a private salt for SPR6 passwordless verification.
	Salt *big.Int

	// Ban information about account suspension. When the ban is
	// not empty, the account has been banned.
	// The reason, expiry and ban time can be found in the AccounBan object.
	Ban *AccountBan
}
