package localdb

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

type Account struct {
	ID       string         `yaml:"id"`
	Name     string         `yaml:"name"`
	Salt     string         `yaml:"salt"`
	Verifier string         `yaml:"verifier"`
	Data     map[string]any `yaml:"data"`
}

func (a *Account) Characters(destination any) error {
	if s, ok := a.Data["characters"].(string); ok {
		if err := json.Unmarshal([]byte(s), destination); err != nil {
			return fmt.Errorf("Account.Characters: %w", err)
		}

		return nil
	}

	if err := json.Unmarshal([]byte{}, destination); err != nil {
		return fmt.Errorf("Account.Characters: %w", err)
	}

	return nil
}

func (a *Account) UpdateCharacters(data any) {
	bb, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Send()
	}

	a.Data["characters"] = string(bb)
}
