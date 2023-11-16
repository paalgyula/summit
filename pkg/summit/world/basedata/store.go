package basedata

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type Store struct {
	playerCreateInfo map[RaceClassGenderKey]*PlayerCreateInfo

	PlayerCreateInfo []*PlayerCreateInfo
}

// LoadFromFile loads the base data from database.
func LoadFromFile(dataPath string) (*Store, error) {
	start := time.Now()

	log.Info().Msg("Loading base data")

	s, err := os.Open(dataPath)
	if err != nil {
		return nil, fmt.Errorf("data load error: %w", err)
	}

	var data Store
	data.playerCreateInfo = map[RaceClassGenderKey]*PlayerCreateInfo{}

	if err := json.NewDecoder(s).Decode(&data); err != nil {
		return nil, fmt.Errorf("cannot decode player create info from file: %w", err)
	}

	for _, i := range data.PlayerCreateInfo {
		data.playerCreateInfo[RaceClassGenderKey{
			Race:   i.Race,
			Class:  i.Class,
			Gender: i.Gender,
		}] = i
	}

	log.Debug().Msgf("BaseData: loaded with %d player create infos", len(data.playerCreateInfo))
	log.Info().Msgf("Base Data loaded in %s", time.Since(start).String())

	store = &data

	return &data, nil
}
