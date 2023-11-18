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
	// Request session via management interface.
	ws.log.Trace().Msgf("requesting auth session for account: %s", account)

	sess := ws.authManagement.GetSession(account)

	if sess != nil {
		ws.log.Trace().Interface("authSession", sess).Msgf("session found")
	}

	return sess
}

// GetCharacters fetches the character list (with full character info) from the store.
func (ws *Server) GetCharacters(account string, characters *player.Players) error {
	chars, err := ws.charStore.GetCharacters(account)
	if err != nil {
		return err
	}

	for _, c := range chars {
		characters.Add(c)
	}

	return err
}

// CreateCharacter saves a new character into the database.
func (ws *Server) CreateCharacter(account string, character *player.Player) error {
	return ws.charStore.CreateCharacter(account, character)
}
