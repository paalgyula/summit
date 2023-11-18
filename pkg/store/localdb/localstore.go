package localdb

import (
	"fmt"
	"os"

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
			log.Info().Interface("account", a).Msgf("Account found: %s", name)

			acc, err := store.NewAccount(a.Name, a.Salt, a.Verifier)
			if err != nil {
				log.Error().Err(err).Msg("account found but looks corrupt")

				return nil
			}

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
func (repo *LocalStore) GetCharacters(account string) (player.Players, error) {
	panic("not implemented") // TODO: Implement
}

// CreateCharacter persists the character in the store.
func (repo *LocalStore) CreateCharacter(account string, character *player.Player) error {
	panic("not implemented") // TODO: Implement
}

// DeleteCharacter removes character from db.
func (repo *LocalStore) DeleteCharacter(characterID int) error {
	panic("not implemented") // TODO: Implement
}

// CreateAccount creates an account with SRP6 encoded password (salt, verifier).
func (repo *LocalStore) CreateAccount(name string, salt string, verifier string) (*store.Account, error) {
	a := &Account{
		ID:       xid.New().String(),
		Name:     name,
		Salt:     salt,
		Verifier: verifier,
	}

	repo.Accounts = append(repo.Accounts, a)

	sa, _ := store.NewAccount(name, salt, verifier)

	return sa, nil
}
