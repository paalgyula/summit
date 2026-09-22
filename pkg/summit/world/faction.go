package world

import (
	"fmt"
	"sync"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc"
	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
)

// Faction masks from AC's DBCEnums.h.
const (
	FactionMaskPlayer   uint32 = 1 // any player
	FactionMaskAlliance uint32 = 2 // player or creature from alliance team
	FactionMaskHorde    uint32 = 4 // player or creature from horde team
	FactionMaskMonster  uint32 = 8 // aggressive creature from monster team
)

// Reputation ranks from AC's SharedDefines.h.
type ReputationRank int32

const (
	RepHated      ReputationRank = 0
	RepHostile    ReputationRank = 1
	RepUnfriendly ReputationRank = 2
	RepNeutral    ReputationRank = 3
	RepFriendly   ReputationRank = 4
	RepHonored    ReputationRank = 5
	RepRevered    ReputationRank = 6
	RepExalted    ReputationRank = 7
)

// FactionManager holds loaded faction template data and provides reaction lookups.
type FactionManager struct {
	mu        sync.RWMutex
	templates map[uint32]*wotlk.FactionTemplateEntry
	loaded    bool
}

// NewFactionManager creates a new FactionManager instance.
func NewFactionManager() *FactionManager {
	return &FactionManager{
		templates: make(map[uint32]*wotlk.FactionTemplateEntry),
	}
}

// global faction manager
var (
	globalFactionMgr     *FactionManager
	globalFactionMgrOnce sync.Once
)

// GetFactionManager returns the global faction manager.
func GetFactionManager() *FactionManager {
	globalFactionMgrOnce.Do(func() {
		globalFactionMgr = &FactionManager{
			templates: make(map[uint32]*wotlk.FactionTemplateEntry),
		}
	})
	return globalFactionMgr
}

// LoadFromDBC loads faction templates from the DBC data.
func (fm *FactionManager) LoadFromDBC(templates []*wotlk.FactionTemplateEntry) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.templates = make(map[uint32]*wotlk.FactionTemplateEntry, len(templates))
	for _, t := range templates {
		if t != nil {
			fm.templates[t.ID] = t
		}
	}
	fm.loaded = true
}

// LoadDBC loads faction templates from FactionTemplate.dbc in the given directory.
func (fm *FactionManager) LoadDBC(dbcPath string) error {
	entries, err := dbc.Load[wotlk.FactionTemplateEntry]("FactionTemplate.dbc", dbcPath)
	if err != nil {
		entries, err = dbc.Load[wotlk.FactionTemplateEntry]("factiontemplate.dbc", dbcPath)
	}
	if err != nil {
		return fmt.Errorf("load faction templates: %w", err)
	}

	ptrEntries := make([]*wotlk.FactionTemplateEntry, len(entries))
	for i := range entries {
		ptrEntries[i] = &entries[i]
	}
	fm.LoadFromDBC(ptrEntries)
	return nil
}

// GetTemplate returns the faction template for the given ID.
func (fm *FactionManager) GetTemplate(id uint32) *wotlk.FactionTemplateEntry {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.templates[id]
}

// IsLoaded returns true if faction templates have been loaded.
func (fm *FactionManager) IsLoaded() bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.loaded
}

// GetReactionTo determines the reaction between two faction templates.
// Maps to AC's FactionTemplateEntry::IsHostileTo/IsFriendlyTo logic.
func (fm *FactionManager) GetReactionTo(selfFaction, targetFaction uint32) ReputationRank {
	self := fm.GetTemplate(selfFaction)
	target := fm.GetTemplate(targetFaction)

	if self == nil || target == nil {
		return RepNeutral
	}

	// Same faction is always friendly
	if self.Faction == target.Faction {
		return RepFriendly
	}

	// Check enemy factions list
	if target.Faction != 0 {
		for _, enemy := range GetEnemyFactions(self) {
			if enemy == target.Faction {
				return RepHostile
			}
		}
		for _, friend := range GetFriendFactions(self) {
			if friend == target.Faction {
				return RepFriendly
			}
		}
	}

	// Check mask-based hostility
	if self.HostileMask&target.OurMask != 0 {
		return RepHostile
	}

	// Check mask-based friendliness
	if self.FriendlyMask&target.OurMask != 0 {
		return RepFriendly
	}
	if self.OurMask&target.FriendlyMask != 0 {
		return RepFriendly
	}

	return RepNeutral
}

// IsHostileTo returns true if selfFaction is hostile to targetFaction.
func (fm *FactionManager) IsHostileTo(selfFaction, targetFaction uint32) bool {
	return fm.GetReactionTo(selfFaction, targetFaction) <= RepHostile
}

// IsFriendlyTo returns true if selfFaction is friendly to targetFaction.
func (fm *FactionManager) IsFriendlyTo(selfFaction, targetFaction uint32) bool {
	return fm.GetReactionTo(selfFaction, targetFaction) >= RepFriendly
}

// GetPlayerFaction returns the faction template ID for a player based on race.
// Alliance races → FactionMaskAlliance, Horde races → FactionMaskHorde.
func GetPlayerFaction(race uint32) uint32 {
	// Race IDs from ChrRaces.dbc:
	// Alliance: 1(Human), 3(Dwarf), 4(NightElf), 7(Gnome), 11(Draenei)
	// Horde: 2(Orc), 5(Undead), 6(Tauren), 8(Troll), 10(BloodElf), 22(Goblin)
	switch race {
	case 1, 3, 4, 7, 11: // Alliance
		return 1 // Default Alliance faction template
	case 2, 5, 6, 8, 10, 22: // Horde
		return 2 // Default Horde faction template
	default:
		return 0
	}
}

// GetEnemyFactions returns the enemy faction IDs from a template.
// These are stored in Field6-Field9 of FactionTemplate.dbc.
func GetEnemyFactions(t *wotlk.FactionTemplateEntry) []uint32 {
	if t == nil {
		return nil
	}

	var result []uint32

	for _, f := range []uint32{t.Field6, t.Field7, t.Field8, t.Field9} {
		if f != 0 {
			result = append(result, f)
		}
	}

	return result
}

// GetFriendFactions returns the friend faction IDs from a template.
// These are stored in Field10-Field13 of FactionTemplate.dbc.
func GetFriendFactions(t *wotlk.FactionTemplateEntry) []uint32 {
	if t == nil {
		return nil
	}

	var result []uint32

	for _, f := range []uint32{t.Field10, t.Field11, t.Field12, t.Field13} {
		if f != 0 {
			result = append(result, f)
		}
	}

	return result
}
