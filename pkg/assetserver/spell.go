package assetserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/paalgyula/summit/pkg/converter/dbc"
)

// SpellPrefix is the route of spell manifests: spell/<id>.json.
const SpellPrefix = "spell/"

// SpellInfo is what the client needs to show a spell on the action bar and in the spellbook.
type SpellInfo struct {
	ID          uint32 `json:"id"`
	Name        string `json:"name"`
	Rank        string `json:"rank,omitempty"`
	Description string `json:"description,omitempty"`
	Tooltip     string `json:"tooltip,omitempty"`
	// Icon is the asset path of the icon without extension (Interface/Icons/...): append .webp.
	Icon string `json:"icon"`
	// CastTimeMs is the base cast time (0 = instant).
	CastTimeMs int32 `json:"castTimeMs"`
	// RangeMin / RangeMax in yards (hostile range).
	RangeMin float32 `json:"rangeMin"`
	RangeMax float32 `json:"rangeMax"`
	// PowerType (0 mana, 1 rage, 2 focus, 3 energy, 6 runic power) and cost.
	PowerType uint32 `json:"powerType"`
	PowerCost uint32 `json:"powerCost"`
	// CooldownMs / CategoryCooldownMs / global cooldown (StartRecoveryTime).
	CooldownMs         uint32 `json:"cooldownMs"`
	CategoryCooldownMs uint32 `json:"categoryCooldownMs"`
	GlobalCooldownMs   uint32 `json:"globalCooldownMs"`
	SchoolMask         uint32 `json:"schoolMask"`
	// Attributes is the first Spell.dbc attribute word (SPELL_ATTR0_*: 0x40 passive, 0x80 hidden...).
	Attributes uint32 `json:"attributes"`
	// Effects are the three Effect columns (SPELL_EFFECT_*), 0 when unused.
	Effects [3]uint32 `json:"effects"`
}

// SPELL_ATTR0 bits used by the client.
const (
	SpellAttrPassive          = 0x40
	SpellAttrHiddenClientside = 0x80
)

// spellTables holds the parsed spell DBCs, loaded on first use.
type spellTables struct {
	once sync.Once
	err  error

	spells    map[uint32]int // id → row
	file      *dbc.File
	icons     map[uint32]string
	castTimes map[uint32]int32
	ranges    map[uint32][2]float32
}

// Spell resolves one spell id from Spell.dbc and its lookup tables.
func (s *Server) Spell(id uint32) (*SpellInfo, error) {
	t := &s.spells
	t.once.Do(func() { t.err = t.load(s) })

	if t.err != nil {
		return nil, t.err
	}

	r, ok := t.spells[id]
	if !ok {
		return nil, fmt.Errorf("spell %d: not in Spell.dbc", id)
	}

	f := t.file
	info := &SpellInfo{
		ID:                 id,
		Name:               f.String(r, 136),
		Rank:               f.String(r, 153),
		Description:        f.String(r, 170),
		Tooltip:            f.String(r, 187),
		Icon:               iconAssetPath(t.icons[f.Uint32(r, 133)]),
		CastTimeMs:         t.castTimes[f.Uint32(r, 28)],
		PowerType:          f.Uint32(r, 41),
		PowerCost:          f.Uint32(r, 42),
		CooldownMs:         f.Uint32(r, 29),
		CategoryCooldownMs: f.Uint32(r, 30),
		GlobalCooldownMs:   f.Uint32(r, 206),
		SchoolMask:         f.Uint32(r, 225),
		Attributes:         f.Uint32(r, 4),
		Effects:            [3]uint32{f.Uint32(r, 71), f.Uint32(r, 72), f.Uint32(r, 73)},
	}

	if rg, ok := t.ranges[f.Uint32(r, 46)]; ok {
		info.RangeMin, info.RangeMax = rg[0], rg[1]
	}

	return info, nil
}

// load parses Spell.dbc (3.3.5a, 234 columns), SpellIcon, SpellCastTimes and SpellRange.
func (t *spellTables) load(s *Server) error {
	read := func(name string) (*dbc.File, error) {
		data, err := s.readSource("DBFilesClient/" + name + ".dbc")
		if err != nil {
			return nil, fmt.Errorf("%s.dbc: %w", name, err)
		}

		f, err := dbc.Read(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%s.dbc: %w", name, err)
		}

		return f, nil
	}

	spell, err := read("Spell")
	if err != nil {
		return err
	}

	if spell.Fields != 234 {
		return fmt.Errorf("Spell.dbc: expected 234 columns (3.3.5a), got %d", spell.Fields)
	}

	icons, err := read("SpellIcon")
	if err != nil {
		return err
	}

	castTimes, err := read("SpellCastTimes")
	if err != nil {
		return err
	}

	ranges, err := read("SpellRange")
	if err != nil {
		return err
	}

	t.file = spell
	t.spells = make(map[uint32]int, spell.Records)
	for r := 0; r < spell.Records; r++ {
		t.spells[spell.Uint32(r, 0)] = r
	}

	t.icons = make(map[uint32]string, icons.Records)
	for r := 0; r < icons.Records; r++ {
		t.icons[icons.Uint32(r, 0)] = icons.String(r, 1)
	}

	t.castTimes = make(map[uint32]int32, castTimes.Records)
	for r := 0; r < castTimes.Records; r++ {
		t.castTimes[castTimes.Uint32(r, 0)] = castTimes.Int32(r, 1)
	}

	t.ranges = make(map[uint32][2]float32, ranges.Records)
	for r := 0; r < ranges.Records; r++ {
		// columns: id, min hostile, min friend, max hostile, max friend
		t.ranges[ranges.Uint32(r, 0)] = [2]float32{ranges.Float32(r, 1), ranges.Float32(r, 3)}
	}

	return nil
}

// iconAssetPath turns Interface\Icons\Foo into the asset path without extension.
func iconAssetPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")

	return strings.TrimSuffix(p, path.Ext(p))
}

// spellID parses spell/<id>.json; ok is false for any other path.
func spellID(relPath string) (uint32, bool) {
	if !strings.HasPrefix(relPath, SpellPrefix) || !strings.HasSuffix(relPath, ".json") {
		return 0, false
	}

	id, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(relPath, SpellPrefix), ".json"), 10, 32)
	if err != nil {
		return 0, false
	}

	return uint32(id), true
}

// tryJITSpell writes the manifest of a spell into the cache.
func (s *Server) tryJITSpell(id uint32, cachedPath string) error {
	info, err := s.Spell(id)
	if err != nil {
		return err
	}

	return writeCacheFile(cachedPath, func(f io.Writer) error {
		return json.NewEncoder(f).Encode(info)
	})
}
