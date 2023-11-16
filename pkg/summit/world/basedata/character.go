package basedata

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

type InventorySlot struct {
	ItemID        int32
	DisplayItemID int32
	InventoryType wow.InventoryType
}

type PlayerCreateInfo struct {
	Race   wow.PlayerRace
	Class  wow.PlayerClass
	Gender wow.PlayerGender

	Map        uint32
	Zone       uint32
	X, Y, Z, O float32

	Inventory []InventorySlot
}

type RaceClassGenderKey struct {
	Race   wow.PlayerRace
	Class  wow.PlayerClass
	Gender wow.PlayerGender
}

type BaseData struct {
	playerCreateInfo map[RaceClassGenderKey]PlayerCreateInfo
}

// LookupCharacterCreateInfo looks up for creation info base data.
// Returns nil if the data not found.
func (bd *BaseData) LookupCharacterCreateInfo(race wow.PlayerRace,
	class wow.PlayerClass, gender wow.PlayerGender,
) *PlayerCreateInfo {
	info := bd.playerCreateInfo[RaceClassGenderKey{
		Race:   race,
		Class:  class,
		Gender: gender,
	}]

	return &info
}

// LoadFromFile loads the base data from database.
func LoadFromFile() (*BaseData, error) {
	start := time.Now()
	log.Info().Msg("Loading base data")

	s, _ := os.Open("summit.dat")

	dec := gob.NewDecoder(s)

	var pci []PlayerCreateInfo
	if err := dec.Decode(&pci); err != nil {
		return nil, fmt.Errorf("cannot decode player create info from file: %w", err)
	}

	data := new(BaseData)
	data.playerCreateInfo = map[RaceClassGenderKey]PlayerCreateInfo{}

	for _, i := range pci {
		data.playerCreateInfo[RaceClassGenderKey{
			Race:   i.Race,
			Class:  i.Class,
			Gender: i.Gender,
		}] = i
	}

	log.Debug().Msgf("BaseData: loaded with %d player create infos", len(pci))
	log.Info().Msgf("Base Data loaded in %s", time.Since(start).String())

	return data, nil
}
