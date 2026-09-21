package basedata

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/loot"
	"github.com/rs/zerolog/log"
)

// Store holds all loaded base data (player create info, items, gameobjects).
type Store struct {
	playerCreateInfo map[RaceClassGenderKey]*PlayerCreateInfo
	items            map[uint32]*ItemTemplate

	GameObjectTemplates map[uint32]*GameObjectTemplate
	GameObjectSpawns    map[uint32]*GameObjectSpawn

	CreatureLoots   map[uint32][]loot.LootEntry // creature_loot_template keyed by entry
	GameObjectLoots map[uint32][]loot.LootEntry // gameobject_loot_template keyed by entry
	ItemLoots       map[uint32][]loot.LootEntry // item_loot_template keyed by entry
	ReferenceLoots  map[uint32][]loot.LootEntry // reference_loot_template keyed by ref id

	PlayerCreateInfo []*PlayerCreateInfo
	Items            []*ItemTemplate

	GOTemplates []*GameObjectTemplate  `bson:"gameobjectTemplates,omitempty" json:"gameobjectTemplates,omitempty"`
	GOSpawns    []*GameObjectSpawn     `bson:"gameobjectSpawns,omitempty" json:"gameobjectSpawns,omitempty"`
	GOLoots     []*loot.LootEntry      `bson:"gameobjectLoots,omitempty" json:"gameobjectLoots,omitempty"`
	CreatureLootEntries  []*loot.LootEntry `bson:"creatureLoots,omitempty" json:"creatureLoots,omitempty"`
	ItemLootEntries      []*loot.LootEntry `bson:"itemLoots,omitempty" json:"itemLoots,omitempty"`
	ReferenceLootEntries []*loot.LootEntry `bson:"referenceLoots,omitempty" json:"referenceLoots,omitempty"`
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

	// Loot templates
	bd.GameObjectLoots = indexLootEntries(bd.GOLoots)
	bd.CreatureLoots = indexLootEntries(bd.CreatureLootEntries)
	bd.ItemLoots = indexLootEntries(bd.ItemLootEntries)
	bd.ReferenceLoots = indexLootEntries(bd.ReferenceLootEntries)
}

// indexLootEntries builds a map of entry → []LootEntry from a slice.
func indexLootEntries(entries []*loot.LootEntry) map[uint32][]loot.LootEntry {
	m := make(map[uint32][]loot.LootEntry, len(entries))
	for _, e := range entries {
		m[e.Entry] = append(m[e.Entry], *e)
	}
	return m
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

// LookupGameObjectLoot returns all loot entries for the given GO loot ID, or nil.
func (bd *Store) LookupGameObjectLoot(lootID uint32) []loot.LootEntry {
	if bd == nil {
		return nil
	}

	entries := bd.GameObjectLoots[lootID]
	if len(entries) == 0 {
		return nil
	}

	return entries
}

// LookupCreatureLoot returns all loot entries for the given creature loot ID, or nil.
func (bd *Store) LookupCreatureLoot(lootID uint32) []loot.LootEntry {
	if bd == nil {
		return nil
	}

	entries := bd.CreatureLoots[lootID]
	if len(entries) == 0 {
		return nil
	}

	return entries
}

// LookupItemLoot returns all loot entries for the given item loot ID, or nil.
func (bd *Store) LookupItemLoot(lootID uint32) []loot.LootEntry {
	if bd == nil {
		return nil
	}

	entries := bd.ItemLoots[lootID]
	if len(entries) == 0 {
		return nil
	}

	return entries
}

// LookupReferenceLoot returns all loot entries for the given reference ID, or nil.
func (bd *Store) LookupReferenceLoot(refID uint32) []loot.LootEntry {
	if bd == nil {
		return nil
	}

	entries := bd.ReferenceLoots[refID]
	if len(entries) == 0 {
		return nil
	}

	return entries
}
