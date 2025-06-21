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

	// GetCharacterForLogin retrieves the minimal set of data required for a player to log in.
	// It also implicitly verifies that the character belongs to the given accountName.
	GetCharacterForLogin(characterGUID wow.GUID, accountName string) (*PlayerDataForLogin, error)
}

// PlayerDataForLogin holds the essential data for a character to enter the world.
// This is a subset of the full player.Player data.
type PlayerDataForLogin struct {
	GUID     wow.GUID // Changed to wow.GUID
	Name     string
	Race     wow.PlayerRace   // Changed to wow.PlayerRace
	Class    wow.PlayerClass  // Changed to wow.PlayerClass
	Gender   wow.PlayerGender // Changed to wow.PlayerGender
	Level    uint8
	Location player.WorldLocation // Make sure this matches player.WorldLocation in pkg/summit/world/object/player/player.go
	// Add other essential fields like DisplayID, etc. as they become necessary for initial packets.
}

type WorldRepo interface{}
