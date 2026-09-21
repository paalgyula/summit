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
	items            map[uint32]*ItemTemplate

	PlayerCreateInfo []*PlayerCreateInfo
	Items            []*ItemTemplate
}

// Index builds the lookup maps of the loaded tables.
func (bd *Store) Index() {
	bd.playerCreateInfo = map[RaceClassGenderKey]*PlayerCreateInfo{}
	for _, i := range bd.PlayerCreateInfo {
		bd.playerCreateInfo[RaceClassGenderKey{
			Race:   i.Race,
			Class:  i.Class,
			Gender: i.Gender,
		}] = i
	}

	bd.items = make(map[uint32]*ItemTemplate, len(bd.Items))
	for _, it := range bd.Items {
		bd.items[it.Entry] = it
	}
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

	if err := json.NewDecoder(s).Decode(&data); err != nil {
		return nil, fmt.Errorf("cannot decode player create info from file: %w", err)
	}

	data.Index()

	log.Debug().Msgf("BaseData: loaded with %d player create infos and %d item templates", len(data.playerCreateInfo), len(data.items))
	log.Info().Msgf("Base Data loaded in %s", time.Since(start).String())

	store = &data

	return &data, nil
}
