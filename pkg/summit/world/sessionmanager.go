package world

import (
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

type SessionManager interface {
	// AddClient adds client to the connection set.
	AddClient(gc *GameClient)

	// Removes and finalizes the connection.
	Disconnected(reason string)

	// GetAuthSession retrives the auth session from login (auth) server.
	GetAuthSession(account string) *auth.Session

	// GetCharacters fetches the character list (with full character info) from the store.
	GetCharacters(account string, characters *player.Players) error

	// CreateCharacter saves a new character into the database.
	CreateCharacter(account string, character *player.Player) error
}

// GetAuthSession retrives the auth session from login (auth) server.
func (ws *Server) GetAuthSession(account string) *auth.Session {
	panic("not implemented") // TODO: Implement
}

// GetCharacters fetches the character list (with full character info) from the store.
func (ws *Server) GetCharacters(account string, characters *player.Players) (err error) {
	*characters, err = ws.characterStore.GetCharacters(account)

	return err
}

// CreateCharacter saves a new character into the database.
func (ws *Server) CreateCharacter(account string, character *player.Player) error {
	return ws.characterStore.CreateCharacter(account, character)
}
