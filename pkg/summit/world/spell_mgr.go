package world

import (
	"fmt"
	"time"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc"
	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/rs/zerolog/log"
)

// SpellChainNode represents spell rank chain (Rank 1 -> Rank 2 -> Rank 3).
type SpellChainNode struct {
	Previous uint32 // previous rank spell ID (0 = none)
	Next     uint32 // next rank spell ID (0 = none)
	First    uint32 // first rank spell ID
	Last     uint32 // last rank spell ID
	Rank     uint8  // rank number (1-based)
}

// SpellProcEntry defines when/how a spell procs.
type SpellProcEntry struct {
	SpellId     uint32
	ProcFlags   uint32
	ProcChance  uint32
	ProcCharges uint32
	ProcEx      uint32
	Cooldown    uint32
	HitMask     uint32
}

// SpellBonusEntry defines spell power coefficients.
type SpellBonusEntry struct {
	DirectDamage    float32
	DotDamage       float32
	DirectHeal      float32
	HotHeal         float32
	OtpDirectDamage float32
	OtpDotDamage    float32
}

// SpellGroupEntry maps a spell to a group.
type SpellGroupEntry struct {
	GroupId int32
	SpellId uint32
}

// SpellGroupStackRule defines how spells in a group stack.
type SpellGroupStackRule struct {
	GroupId int32
	Rule    int32 // 0 = unique, 1 = exclusive, etc.
}

// SpellMgr manages all spell data loaded from DBC files.
type SpellMgr struct {
	spellInfo map[uint32]*SpellInfo

	// DBC data
	castTimes map[uint32]wotlk.SpellCastTimeEntry
	durations map[uint32]wotlk.SpellDurationEntry
	ranges    map[uint32]wotlk.SpellRangeEntry

	// Spell chain (rank) data
	spellChains map[uint32]*SpellChainNode

	// Spell proc data
	spellProcs map[uint32]*SpellProcEntry

	// Spell bonus data
	spellBonuses map[uint32]*SpellBonusEntry

	// Spell group data: spellId -> entries
	spellGroupMap map[uint32][]*SpellGroupEntry

	// Spell group stack rules: groupId -> rule
	spellGroupStackRules map[int32]*SpellGroupStackRule
}

// NewSpellMgr creates a new SpellMgr.
func NewSpellMgr() *SpellMgr {
	return &SpellMgr{
		spellInfo:            make(map[uint32]*SpellInfo),
		castTimes:            make(map[uint32]wotlk.SpellCastTimeEntry),
		durations:            make(map[uint32]wotlk.SpellDurationEntry),
		ranges:               make(map[uint32]wotlk.SpellRangeEntry),
		spellChains:          make(map[uint32]*SpellChainNode),
		spellProcs:           make(map[uint32]*SpellProcEntry),
		spellBonuses:         make(map[uint32]*SpellBonusEntry),
		spellGroupMap:        make(map[uint32][]*SpellGroupEntry),
		spellGroupStackRules: make(map[int32]*SpellGroupStackRule),
	}
}

// LoadSpells loads all spell data from DBC files in the given directory.
func (sm *SpellMgr) LoadSpells(dbcPath string) error {
	start := time.Now()

	// Load supporting DBCs
	if err := sm.loadSupportingDBC(dbcPath); err != nil {
		return fmt.Errorf("load supporting DBC: %w", err)
	}

	// Load Spell.dbc
	entries, err := dbc.Load[wotlk.SpellEntryEntry]("Spell.dbc", dbcPath)
	if err != nil {
		return fmt.Errorf("load Spell.dbc: %w", err)
	}

	// Build SpellInfo for each entry
	castTimeSlice := sm.castTimesSlice()
	durationSlice := sm.durationsSlice()
	rangeSlice := sm.rangesSlice()

	for i := range entries {
		entry := &entries[i]
		if entry.Id == 0 {
			continue
		}
		si := NewSpellInfo(entry, castTimeSlice, durationSlice, rangeSlice)
		sm.spellInfo[entry.Id] = si
	}

	log.Info().Msgf("SpellMgr: loaded %d spells in %s", len(sm.spellInfo), time.Since(start).String())

	return nil
}

func (sm *SpellMgr) loadSupportingDBC(dbcPath string) error {
	// Load SpellCastTime.dbc
	castTimes, err := dbc.Load[wotlk.SpellCastTimeEntry]("SpellCastTime.dbc", dbcPath)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load SpellCastTime.dbc, using defaults")
	} else {
		for _, ct := range castTimes {
			sm.castTimes[ct.ID] = ct
		}
	}

	// Load SpellDuration.dbc
	durations, err := dbc.Load[wotlk.SpellDurationEntry]("SpellDuration.dbc", dbcPath)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load SpellDuration.dbc, using defaults")
	} else {
		for _, d := range durations {
			sm.durations[d.ID] = d
		}
	}

	// Load SpellRange.dbc
	ranges, err := dbc.Load[wotlk.SpellRangeEntry]("SpellRange.dbc", dbcPath)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load SpellRange.dbc, using defaults")
	} else {
		for _, r := range ranges {
			sm.ranges[r.ID] = r
		}
	}

	return nil
}

func (sm *SpellMgr) castTimesSlice() []wotlk.SpellCastTimeEntry {
	s := make([]wotlk.SpellCastTimeEntry, 0, len(sm.castTimes))
	for _, v := range sm.castTimes {
		s = append(s, v)
	}
	return s
}

func (sm *SpellMgr) durationsSlice() []wotlk.SpellDurationEntry {
	s := make([]wotlk.SpellDurationEntry, 0, len(sm.durations))
	for _, v := range sm.durations {
		s = append(s, v)
	}
	return s
}

func (sm *SpellMgr) rangesSlice() []wotlk.SpellRangeEntry {
	s := make([]wotlk.SpellRangeEntry, 0, len(sm.ranges))
	for _, v := range sm.ranges {
		s = append(s, v)
	}
	return s
}

// GetSpellInfo returns the SpellInfo for the given spell ID, or nil if not found.
func (sm *SpellMgr) GetSpellInfo(spellId uint32) *SpellInfo {
	return sm.spellInfo[spellId]
}

// GetSpellInfoStoreSize returns the total number of loaded spells.
func (sm *SpellMgr) GetSpellInfoStoreSize() uint32 {
	return uint32(len(sm.spellInfo))
}

// ---------------------------------------------------------------------------
// Spell chain (rank) methods
// ---------------------------------------------------------------------------

// GetSpellChainNode returns the chain node for the given spell, or nil if not found.
func (sm *SpellMgr) GetSpellChainNode(spellId uint32) *SpellChainNode {
	return sm.spellChains[spellId]
}

// GetFirstSpellInChain returns the first rank spell ID in the chain, or 0 if not found.
func (sm *SpellMgr) GetFirstSpellInChain(spellId uint32) uint32 {
	if node, ok := sm.spellChains[spellId]; ok {
		return node.First
	}
	return 0
}

// GetLastSpellInChain returns the last rank spell ID in the chain, or 0 if not found.
func (sm *SpellMgr) GetLastSpellInChain(spellId uint32) uint32 {
	if node, ok := sm.spellChains[spellId]; ok {
		return node.Last
	}
	return 0
}

// GetNextSpellInChain returns the next rank spell ID, or 0 if none.
func (sm *SpellMgr) GetNextSpellInChain(spellId uint32) uint32 {
	if node, ok := sm.spellChains[spellId]; ok {
		return node.Next
	}
	return 0
}

// GetPrevSpellInChain returns the previous rank spell ID, or 0 if none.
func (sm *SpellMgr) GetPrevSpellInChain(spellId uint32) uint32 {
	if node, ok := sm.spellChains[spellId]; ok {
		return node.Previous
	}
	return 0
}

// GetSpellRank returns the rank of the spell (1-based), or 0 if not in a chain.
func (sm *SpellMgr) GetSpellRank(spellId uint32) uint8 {
	if node, ok := sm.spellChains[spellId]; ok {
		return node.Rank
	}
	return 0
}

// ---------------------------------------------------------------------------
// Proc methods
// ---------------------------------------------------------------------------

// GetSpellProcEntry returns the proc entry for the given spell, or nil if not found.
func (sm *SpellMgr) GetSpellProcEntry(spellId uint32) *SpellProcEntry {
	return sm.spellProcs[spellId]
}

// ---------------------------------------------------------------------------
// Bonus methods
// ---------------------------------------------------------------------------

// GetSpellBonusData returns the bonus entry for the given spell, or nil if not found.
func (sm *SpellMgr) GetSpellBonusData(spellId uint32) *SpellBonusEntry {
	return sm.spellBonuses[spellId]
}

// ---------------------------------------------------------------------------
// Group methods
// ---------------------------------------------------------------------------

// GetSpellGroupSpellMap returns all group entries for the given spell, or nil if not found.
func (sm *SpellMgr) GetSpellGroupSpellMap(spellId uint32) []*SpellGroupEntry {
	return sm.spellGroupMap[spellId]
}

// GetSpellGroupStackRule returns the stack rule for the given group, or nil if not found.
func (sm *SpellMgr) GetSpellGroupStackRule(groupId int32) *SpellGroupStackRule {
	return sm.spellGroupStackRules[groupId]
}

// ---------------------------------------------------------------------------
// Stub loading methods (to be implemented with DB queries)
// ---------------------------------------------------------------------------

// LoadSpellRanks loads spell rank chain data from the spell_ranks DB table.
func (sm *SpellMgr) LoadSpellRanks() {
	log.Info().Msg("SpellMgr: LoadSpellRanks not yet implemented")
}

// LoadSpellProcs loads spell proc entries from the spell_proc DB table.
func (sm *SpellMgr) LoadSpellProcs() {
	log.Info().Msg("SpellMgr: LoadSpellProcs not yet implemented")
}

// LoadSpellBonuses loads spell bonus data from the spell_bonus_data DB table.
func (sm *SpellMgr) LoadSpellBonuses() {
	log.Info().Msg("SpellMgr: LoadSpellBonuses not yet implemented")
}

// LoadSpellGroups loads spell group mappings from the spell_group DB table.
func (sm *SpellMgr) LoadSpellGroups() {
	log.Info().Msg("SpellMgr: LoadSpellGroups not yet implemented")
}

// LoadSpellGroupStackRules loads spell group stack rules from the spell_group_stack_rules DB table.
func (sm *SpellMgr) LoadSpellGroupStackRules() {
	log.Info().Msg("SpellMgr: LoadSpellGroupStackRules not yet implemented")
}
