package db

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog/log"
)

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

func (db *Database) CreateAccount(name, pass string) (*Account, error) {
	// need to build verifier from name, pass, salt
	srp := crypt.NewSRP6(0, 0, big.NewInt(0))

	acc := &Account{Name: strings.ToUpper(name), salt: srp.RandomSalt(), Data: make(map[string]string)}

	acc.verifier = srp.GenerateVerifier(acc.Name, strings.ToUpper(pass), acc.salt)

	// store as hex
	acc.S = acc.salt.Text(16)
	acc.V = acc.verifier.Text(16)

	// todo - check if account already exists, return error if it does

	db.Accounts = append(db.Accounts, acc)

	return acc, nil
}

type AccountData struct {
	Data string `yaml:"data"`
	Time uint32 `yaml:"time"`
}

type Account struct {
	Name    string `yaml:"name"`
	V       string `yaml:"verifier"`
	S       string `yaml:"salt"`
	Session string `yaml:"-"`

	verifier   *big.Int
	salt       *big.Int
	sessionKey *big.Int

	Data map[string]string `yaml:"data"`

	Metadata [wow.NUM_ACCOUNT_DATA_TYPES]AccountData `yaml:"metadata"`
}

func (a *Account) Characters(destination any) error {
	if s, ok := a.Data["characters"]; ok {
		return json.Unmarshal([]byte(s), destination)
	}

	return json.Unmarshal([]byte{}, destination)
}

func (a *Account) UpdateCharacters(data any) {
	bb, _ := json.Marshal(data)
	a.Data["characters"] = string(bb)
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
