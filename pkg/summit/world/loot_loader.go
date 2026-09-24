package world

import (
	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/loot"
	"github.com/rs/zerolog/log"
)

// newLootManager loads every loot template from the world store into a
// loot.Manager. A missing store (or empty tables) yields an empty manager so
// the server still starts.
func newLootManager(worldStore store.WorldRepo) *loot.Manager {
	mgr := loot.NewManager()

	if worldStore == nil {
		return mgr
	}

	tables, err := worldStore.GetAllLootTemplates()
	if err != nil {
		log.Warn().Err(err).Msg("cannot load loot templates from the world store")

		return mgr
	}

	add := func(t loot.StoreType, entries map[uint32][]store.LootTemplate) {
		for _, list := range entries {
			for _, e := range list {
				mgr.Add(t, loot.LootEntry{
					Entry:     e.Entry,
					Item:      e.Item,
					Ref:       e.Reference,
					Chance:    e.Chance,
					NeedQuest: e.QuestRequired,
					LootMode:  loot.LootMode(e.LootMode),
					GroupID:   e.GroupID,
					MinCount:  e.MinCount,
					MaxCount:  e.MaxCount,
				})
			}
		}
	}

	add(loot.StoreCreature, tables.Creature)
	add(loot.StoreGameObject, tables.GameObject)
	add(loot.StoreItem, tables.Item)
	add(loot.StoreReference, tables.Reference)

	log.Info().
		Int("creature", mgr.Count(loot.StoreCreature)).
		Int("gameobject", mgr.Count(loot.StoreGameObject)).
		Int("item", mgr.Count(loot.StoreItem)).
		Int("reference", mgr.Count(loot.StoreReference)).
		Msg("loaded loot templates from store")

	return mgr
}
