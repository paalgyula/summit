package store

import "github.com/paalgyula/summit/pkg/summit/world/object/player"

type AccountRepo interface {
	// FindAccount retrives an account from the store or returns nil if does not exists.
	FindAccount(name string) *Account

	// CreateAccount creates an account with SRP6 encoded password (salt, verifier).
	CreateAccount(name, salt, verifier string) (*Account, error)
}

type CharacterRepo interface {
	// Retrives characters from the store for the specified account.
	GetCharacters(account string) (player.Players, error)

	// CreateCharacter persists the character in the store.
	CreateCharacter(account string, character *player.Player) error

	// DeleteCharacter removes character from db.
	DeleteCharacter(characterID int) error
}

type WorldRepo interface{}
