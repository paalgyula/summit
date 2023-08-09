package db

import (
	"fmt"
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
	// Characters []*Character `yaml:"characters"`
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
	log.Info().Msg("Saving world state to the database")

	f, _ := os.Create(SummitConfig)
	_ = yaml.NewEncoder(f).Encode(db)
}

func initYamlDatabase() {
	instance = &Database{}
	err := instance.Load(SummitConfig)

	if err != nil {
		// BIG TODO - if this error occurs due to a failed unmarshal, the database file gets WIPED
		log.Warn().Err(err).Msgf("database file: %s not found", SummitConfig)

		instance.Accounts = make([]*Account, 0)
	}

	log.Info().Msgf("Loaded database with %d accounts", len(instance.Accounts))
}

func GetInstance() *Database {
	once.Do(initYamlDatabase)

	return instance
}
