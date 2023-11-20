package mysqldb

import "github.com/paalgyula/summit/pkg/summit/world/object/player"

type CharacterStore struct{}

// Retrives characters from the store for the specified account.
func (store *CharacterStore) GetCharacters(account string) (player.Players, error) {
	panic("not implemented") // TODO: Implement
}

// CreateCharacter persists the character in the store.
func (store *CharacterStore) CreateCharacter(account string, character *player.Player) error {
	panic("not implemented") // TODO: Implement
}

// DeleteCharacter removes character from db.
func (store *CharacterStore) DeleteCharacter(characterID int) error {
	panic("not implemented") // TODO: Implement
}
