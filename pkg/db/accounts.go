package db

import (
	"fmt"
	"math/big"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const SummitConfig = "summit.yaml"

var once sync.Once
var instance *Database

type Database struct {
	Accounts []*Account `yaml:"accounts"`
}

func (db *Database) FindAccount(name string) *Account {
	for _, a := range db.Accounts {
		if a.Name == name {
			log.Info().Interface("account", a).Msgf("Account found: %s", name)

			return a
		}
	}

	log.Info().Msgf("Account %s not found", name)

	return nil
}

func (db *Database) Load(path string) error {
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

func (db *Database) SaveAll() {
	log.Info().Msg("Saving world state to the datbase")

	f, _ := os.Create(SummitConfig)
	_ = yaml.NewEncoder(f).Encode(db)
}

func initDatabase() {
	instance = &Database{}
	err := instance.Load(SummitConfig)

	if err != nil {
		panic(err)
	}

	log.Info().Msgf("Loaded database with %d accounts", len(instance.Accounts))
}

func GetInstance() *Database {
	once.Do(initDatabase)

	return instance
}

type Account struct {
	Name    string `yaml:"name"`
	V       string `yaml:"verifier"`
	S       string `yaml:"salt"`
	Session string `yaml:"-"`

	verifier   *big.Int
	salt       *big.Int
	sessionKey *big.Int
}

func (a *Account) SetKey(k *big.Int) {
	a.sessionKey = k
	a.Session = k.Text(16)
}

// Verifier gets a big.Int version of the account verifier.
func (a *Account) Verifier() *big.Int {
	if a.verifier == nil {
		a.verifier, _ = new(big.Int).SetString(a.V, 16)
	}

	return a.verifier
}

// Salt gets a big.Int version of the account salt.
func (a *Account) Salt() *big.Int {
	if a.salt == nil {
		a.salt, _ = new(big.Int).SetString(a.S, 16)
	}

	return a.salt
}

// SessionKey gets a big.Int version of the account session key.
func (a *Account) SessionKey() *big.Int {
	return a.sessionKey
}
