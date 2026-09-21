package data

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	cdbc "github.com/paalgyula/summit/pkg/converter/dbc"
	"github.com/paalgyula/summit/pkg/summit/tools/dbc"
	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"

	_ "embed"
)

var ErrDBCFileError = errors.New("cannot read dbc file")

//go:embed spawn_data.csv
var spawnData []byte

func NewConverter(dbcBasePath string) *Converter {
	return &Converter{
		dbcBase:   dbcBasePath,
		spawnData: []*SpawnRecord{},
	}
}

type Converter struct {
	dbcBase string

	spawnData []*SpawnRecord
}

func (dc *Converter) ConvertPlayerCreateInfo() ([]*basedata.PlayerCreateInfo, error) {
	data, err := dbc.Load[wotlk.CharStartOutfitEntry]("CharStartOutfit.dbc", dc.dbcBase)
	if err != nil {
		return nil, fmt.Errorf("%w: CharStartOutfit.dbc", err)
	}

	playerCreateInfo := make([]*basedata.PlayerCreateInfo, len(data))

	for i, csoe := range data {
		spawn := dc.LookupSpawn(csoe.RaceID, csoe.ClassID)

		inventory := make([]*basedata.InventorySlot, 24)

		// Fill up inventory
		for i := range inventory {
			inventory[i] = &basedata.InventorySlot{
				ItemID:        csoe.ItemID[i],
				DisplayItemID: csoe.DisplayItemID[i],
				InventoryType: wow.InventoryType(csoe.InventoryType[i]),
			}
		}

		playerCreateInfo[i] = &basedata.PlayerCreateInfo{
			Race:      wow.PlayerRace(csoe.RaceID),
			Class:     wow.PlayerClass(csoe.ClassID),
			Gender:    wow.PlayerGender(csoe.Gender),
			Map:       uint32(spawn.MapID),
			Zone:      uint32(spawn.ZoneID),
			X:         spawn.X,
			Y:         spawn.Y,
			Z:         spawn.Z,
			O:         spawn.O,
			Inventory: inventory,
		}
	}

	return playerCreateInfo, nil
}

// ConvertItems reads the item templates of Item.dbc: the 8-column client
// table (id, class, subclass, sound, material, display, inventory type,
// sheathe) or the 4-column dump (id, display, inventory type, sheathe).
func (dc *Converter) ConvertItems() ([]*basedata.ItemTemplate, error) {
	f, err := os.Open(path.Join(dc.dbcBase, "Item.dbc"))
	if err != nil {
		return nil, fmt.Errorf("%w: Item.dbc: %w", ErrDBCFileError, err)
	}
	defer f.Close()

	table, err := cdbc.Read(f)
	if err != nil {
		return nil, fmt.Errorf("%w: Item.dbc: %w", ErrDBCFileError, err)
	}

	displayCol, typeCol, sheatheCol := 5, 6, 7
	if table.Fields == 4 {
		displayCol, typeCol, sheatheCol = 1, 2, 3
	}

	items := make([]*basedata.ItemTemplate, 0, table.Records)
	for r := 0; r < table.Records; r++ {
		items = append(items, &basedata.ItemTemplate{
			Entry:         table.Uint32(r, 0),
			DisplayID:     table.Uint32(r, displayCol),
			InventoryType: wow.InventoryType(table.Uint32(r, typeCol)),
			SheatheType:   table.Uint32(r, sheatheCol),
		})
	}

	return items, nil
}

func (dc *Converter) LoadSpawnData() []*SpawnRecord {
	r := csv.NewReader(bytes.NewReader(spawnData))

	// * Header
	_, _ = r.Read()

	// * Allocate and read records
	records, _ := r.ReadAll()
	srr := make([]*SpawnRecord, len(records))

	// * Parse records into struct
	for i, rec := range records {
		var row SpawnRecord
		if err := Unmarshal(rec, &row); err != nil {
			panic(err)
		}

		srr[i] = &row
	}

	dc.spawnData = srr

	return srr
}

func (dc *Converter) LookupSpawn(race, class uint8) *SpawnRecord {
	for _, sr := range dc.spawnData {
		if sr.Class == class && sr.Race == race {
			return sr
		}
	}

	log.Error().Msgf("spawn not found for race: %v - class: %+v", wow.PlayerRace(race), wow.PlayerClass(class))

	return nil
}

func (dc *Converter) CreateSummitBaseData() error {
	var store basedata.Store

	log.Info().Msg("Loading spawn data")

	dc.spawnData = dc.LoadSpawnData()

	log.Info().Msg("Converting player create info")

	var err error

	store.PlayerCreateInfo, err = dc.ConvertPlayerCreateInfo()
	if err != nil {
		return err
	}

	log.Info().Msg("Converting item templates")

	store.Items, err = dc.ConvertItems()
	if err != nil {
		return err
	}

	f, _ := os.Create("summit.dat")

	if err := json.NewEncoder(f).Encode(store); err != nil {
		return fmt.Errorf("data.CreateSummitBaseData: %w", err)
	}

	return nil
}
