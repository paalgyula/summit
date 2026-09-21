package basedata

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// Store holds all loaded base data (player create info, items, gameobjects).
type Store struct {
	playerCreateInfo map[RaceClassGenderKey]*PlayerCreateInfo
	items            map[uint32]*ItemTemplate

	GameObjectTemplates map[uint32]*GameObjectTemplate
	GameObjectSpawns    map[uint32]*GameObjectSpawn
	GameObjectLoots     map[uint32][]*GameObjectLootEntry // keyed by loot entry ID

	PlayerCreateInfo []*PlayerCreateInfo
	Items            []*ItemTemplate

	GOTemplates []*GameObjectTemplate  `bson:"gameobjectTemplates,omitempty" json:"gameobjectTemplates,omitempty"`
	GOSpawns    []*GameObjectSpawn     `bson:"gameobjectSpawns,omitempty" json:"gameobjectSpawns,omitempty"`
	GOLoots     []*GameObjectLootEntry `bson:"gameobjectLoots,omitempty" json:"gameobjectLoots,omitempty"`
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

	// GameObject templates
	bd.GameObjectTemplates = make(map[uint32]*GameObjectTemplate, len(bd.GOTemplates))
	for _, t := range bd.GOTemplates {
		bd.GameObjectTemplates[t.Entry] = t
	}

	// GameObject spawns
	bd.GameObjectSpawns = make(map[uint32]*GameObjectSpawn, len(bd.GOSpawns))
	for _, s := range bd.GOSpawns {
		bd.GameObjectSpawns[s.GUID] = s
	}

	// GameObject loot (multiple items per entry)
	bd.GameObjectLoots = make(map[uint32][]*GameObjectLootEntry, len(bd.GOLoots))
	for _, l := range bd.GOLoots {
		bd.GameObjectLoots[l.Entry] = append(bd.GameObjectLoots[l.Entry], l)
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

	log.Debug().Msgf("BaseData: loaded with %d player create infos, %d item templates, %d GO templates, %d GO spawns",
		len(data.playerCreateInfo), len(data.items),
		len(data.GOTemplates), len(data.GOSpawns))
	log.Info().Msgf("Base Data loaded in %s", time.Since(start).String())

	SetInstance(&data)

	return &data, nil
}

// --- GameObject lookups ---

// LookupGameObjectTemplate returns the template for the given entry, or nil.
func (bd *Store) LookupGameObjectTemplate(entry uint32) *GameObjectTemplate {
	if bd == nil {
		return nil
	}

	return bd.GameObjectTemplates[entry]
}

// LookupGameObjectSpawn returns the spawn record for the given GUID, or nil.
func (bd *Store) LookupGameObjectSpawn(guid uint32) *GameObjectSpawn {
	if bd == nil {
		return nil
	}

	return bd.GameObjectSpawns[guid]
}

// LookupGameObjectLoot returns all loot entries for the given loot ID, or nil.
func (bd *Store) LookupGameObjectLoot(lootID uint32) []GameObjectLootEntry {
	if bd == nil {
		return nil
	}

	entries := bd.GameObjectLoots[lootID]
	if len(entries) == 0 {
		return nil
	}

	result := make([]GameObjectLootEntry, len(entries))

	for i, e := range entries {
		result[i] = *e
	}

	return result
}
