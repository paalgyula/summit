package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/loot"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

// FillLoot generates the game object's loot from its template loot id, if it
// has not been generated yet. Uses the game object loot store plus the
// reference_loot_template store for `-ref` entries.
func (g *GameObject) FillLoot(bd *basedata.Store) {
	if g.Loot != nil || g.Template == nil || bd == nil {
		return
	}

	lootID := g.Template.GetLootID()
	if lootID == 0 {
		return
	}

	store := loot.NewLootStore("gameobject")
	for _, e := range bd.LookupGameObjectLoot(lootID) {
		store.AddEntry(e)
	}

	var refStore *loot.LootStore

	if len(bd.ReferenceLoots) > 0 {
		refStore = loot.NewLootStore("reference")
		for _, entries := range bd.ReferenceLoots {
			for _, e := range entries {
				refStore.AddEntry(e)
			}
		}
	}

	l := loot.NewLoot()
	_ = l.FillLoot(lootID, store, loot.LootModeDefault, refStore)
	l.GenerateMoneyLoot(g.Template.MinGold, g.Template.MaxGold)

	g.Loot = l
}

// SetLootRecipient records the player allowed to loot this game object.
func (g *GameObject) SetLootRecipient(guid wow.GUID) {
	g.LootRecipient = guid
}

// GetLootRecipient returns the player allowed to loot this game object (0 when
// unrestricted).
func (g *GameObject) GetLootRecipient() wow.GUID {
	return g.LootRecipient
}

// IsLootAllowedFor reports whether the given player may loot this game object.
func (g *GameObject) IsLootAllowedFor(playerGUID wow.GUID) bool {
	return g.LootRecipient == 0 || g.LootRecipient == playerGUID
}

// openGameObjectLoot opens the loot window for a game object: generates loot if
// needed, marks the player as recipient and sends SMSG_LOOT_RESPONSE.
func (gc *WorldSession) openGameObjectLoot(g *GameObject) {
	if gc.player == nil {
		return
	}

	bd := basedata.GetInstance()
	if bd == nil {
		return
	}

	g.FillLoot(bd)

	if g.Loot == nil {
		g.Loot = loot.NewLoot()
	}

	g.SetLootRecipient(gc.player.GUID())

	gc.player.LootGUID = uint64(g.GetGUID())
	gc.player.ActiveLoot = &lootSource{
		GUID:     g.GetGUID(),
		Loot:     g.Loot,
		LootType: loot.LootCorpse,
	}

	gc.sendLootResponse(g.GetGUID(), loot.LootCorpse, g.Loot)

	log.Debug().
		Uint32("entry", g.Entry).
		Uint32("guid", g.ID).
		Msg("opened game object loot")
}

// sendGameObjectPageText sends SMSG_GAMEOBJECT_PAGETEXT, telling the client to
// open the GO's text page. Mirrors GameObject::Use's goober page handling.
func (gc *WorldSession) sendGameObjectPageText(g *GameObject, pageID uint32) {
	pkt := wow.NewPacket(wow.ServerGameobjectPagetext)
	_ = pkt.Write(g.GetGUID())
	gc.Send(pkt)

	log.Debug().
		Uint32("entry", g.Entry).
		Uint32("page", pageID).
		Msg("sent game object page text")
}
