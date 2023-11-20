package localdb

import (
	"fmt"
	"os"
	"strings"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const SummitConfig = "summit.yaml"

func InitYamlDatabase(dbpath string) *LocalStore {
	store := &LocalStore{
		Accounts: make([]*Account, 0),
	}

	if err := store.Load(dbpath); err != nil {
		log.Warn().Err(err).Msgf("database file: %s not found", dbpath)
		log.Warn().Msg("auth server initialized with empty database")
	}

	log.Info().Msgf("Loaded database with %d accounts", len(store.Accounts))

	return store
}

type LocalStore struct {
	Accounts []*Account `yaml:"accounts"`
}

func (db *LocalStore) FindAccount(name string) *store.Account {
	for _, a := range db.Accounts {
		if a.Name == name {
			log.Info().Msgf("Account found: %s, banned: %t", name, a.BanInfo != nil)

			acc, err := store.AccountFromCreds(a.Name, a.Salt, a.Verifier)
			if err != nil {
				log.Error().Err(err).Msg("account found but looks corrupt")

				return nil
			}

			acc.Email = a.Email
			acc.CreatedAt = a.CreatedAt
			acc.LastLogin = a.LastLogin
			acc.Activated = a.Activated

			// TODO: #39 implement proper marshal/unmarshal for account ban
			// acc.Ban = a.BanInfo

			return acc
		}
	}

	return nil
}

func (db *LocalStore) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("database.Load: %w", err)
	}

	err = yaml.NewDecoder(f).Decode(db)
	if err != nil {
		return fmt.Errorf("database.Load: %w", err)
	}

	return nil
}

func (db *LocalStore) SaveAll() {
	log.Info().Msg("Saving world state to the database")

	f, _ := os.Create(SummitConfig)
	_ = yaml.NewEncoder(f).Encode(db)
}

// Retrives characters from the store for the specified account.
func (db *LocalStore) GetCharacters(account string) (player.Players, error) {
	var pp player.Players

	for _, a := range db.Accounts {
		if a.Name == strings.ToUpper(account) {
			if err := a.Characters(&pp); err != nil {
				return nil, err
			}
		}
	}

	return pp, nil
}

// CreateCharacter persists the character in the store.
func (db *LocalStore) CreateCharacter(account string, character *player.Player) error {
	panic("not implemented") // TODO: Implement
}

// DeleteCharacter removes character from db.
func (db *LocalStore) DeleteCharacter(characterID int) error {
	panic("not implemented") // TODO: Implement
}

// CreateAccount creates an account with SRP6 encoded password (salt, verifier).
func (db *LocalStore) CreateAccount(acc *store.Account) error {
	a := &Account{
		ID:        xid.New().String(),
		Name:      acc.Name,
		Email:     acc.Email,
		Salt:      acc.Salt.Text(16),
		Verifier:  acc.Verifier.Text(16),
		CreatedAt: acc.CreatedAt,
		LastLogin: acc.LastLogin,
		Activated: acc.Activated,
		BanInfo:   acc.Ban,
	}

	acc.ID = a.ID

	db.Accounts = append(db.Accounts, a)

	return nil
}
