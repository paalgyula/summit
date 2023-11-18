package store

import "github.com/paalgyula/summit/pkg/summit/world/object/player"

type AccountRepo interface {
	// FindAccount retrives an account from the store or returns nil if does not exists.
	FindAccount(name string) *Account

	// CreateAccount creates an account. The account verifier and salt should be set.
	CreateAccount(account *Account) error
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
