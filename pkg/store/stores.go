package store

import "github.com/paalgyula/summit/pkg/summit/world/object/player"

// AccountRepo is the interface for account persistence.
type AccountRepo interface {
	// FindAccount retrives an account from the store or returns nil if does not exists.
	FindAccount(name string) *Account

	// CreateAccount creates an account. The account verifier and salt should be set.
	CreateAccount(account *Account) error
}

// CharacterRepo is the interface for character persistence.
type CharacterRepo interface {
	// GetCharacters retrives characters from the store for the specified account.
	GetCharacters(account string) (player.Players, error)

	// GetCharacter retrieves a single character by GUID.
	GetCharacter(guid uint32) (*player.Player, error)

	// CreateCharacter persists the character in the store.
	CreateCharacter(account string, character *player.Player) error

	// UpdateCharacter updates an existing character in the store.
	UpdateCharacter(character *player.Player) error

	// DeleteCharacter removes character from db.
	DeleteCharacter(characterID int) error
}

// WorldRepo is the interface for world data persistence (future use).
type WorldRepo interface{}

//go:generate mockgen -destination=mock_store/mock_account_repo.go -package=mock_store . AccountRepo
//go:generate mockgen -destination=mock_store/mock_character_repo.go -package=mock_store . CharacterRepo
//go:generate mockgen -destination=mock_store/mock_world_repo.go -package=mock_store . WorldRepo
