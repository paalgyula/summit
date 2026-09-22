package world

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func packU32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func packU64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func packF32(v float32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	return b
}

// useItemHeader builds a CMSG_USE_ITEM payload up to castFlags (no targets).
// The item GUID is a raw u64 on the wire (wowm `Guid item`), not packed.
func useItemHeader(bag, slot, castCount uint8, spellID uint32, itemGUID wow.GUID, glyph uint32, castFlags uint8) []byte {
	buf := []byte{bag, slot, castCount}
	buf = append(buf, packU32(spellID)...)
	buf = append(buf, packU64(uint64(itemGUID))...)
	buf = append(buf, packU32(glyph)...)
	buf = append(buf, castFlags)
	return buf
}

func TestReadSpellCastTargets_None(t *testing.T) {
	r := wow.NewPacketReader(packU32(TargetFlagNone))
	targets, err := readSpellCastTargets(r, nil)
	require.NoError(t, err)
	require.NotNil(t, targets)
	assert.Equal(t, TargetFlagNone, targets.TargetMask)
	assert.Nil(t, targets.UnitTarget)
}

func TestReadSpellCastTargets_Unit(t *testing.T) {
	guid := wow.NewGUID(wow.UnitGUID, 0x1234)
	buf := packU32(TargetFlagUnit)
	buf = append(buf, guid.Pack()...)

	r := wow.NewPacketReader(buf)
	var resolved wow.GUID
	targets, err := readSpellCastTargets(r, func(g wow.GUID) Unit {
		resolved = g
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, targets)
	assert.Equal(t, TargetFlagUnit, targets.TargetMask)
	assert.Equal(t, guid, resolved)
}

func TestReadSpellCastTargets_DestLocation(t *testing.T) {
	buf := packU32(TargetFlagDestLocation)
	buf = append(buf, wow.GUID(0).Pack()...) // zero transport
	buf = append(buf, packF32(1.5)...)
	buf = append(buf, packF32(-2.5)...)
	buf = append(buf, packF32(100)...)

	r := wow.NewPacketReader(buf)
	targets, err := readSpellCastTargets(r, nil)
	require.NoError(t, err)
	require.NotNil(t, targets)
	assert.InDelta(t, float32(1.5), targets.DstPosition.X, 0.001)
	assert.InDelta(t, float32(-2.5), targets.DstPosition.Y, 0.001)
	assert.InDelta(t, float32(100), targets.DstPosition.Z, 0.001)
}

func TestUseItemHeaderLayout(t *testing.T) {
	guid := wow.NewGUID(wow.ItemGUID, 42)
	raw := useItemHeader(0, 23, 1, 2054, guid, 0, 0)
	raw = append(raw, packU32(TargetFlagNone)...)

	r := wow.NewPacketReader(raw)

	var bag, slot, castCount uint8
	require.NoError(t, r.Read(&bag))
	require.NoError(t, r.Read(&slot))
	require.NoError(t, r.Read(&castCount))
	assert.Equal(t, uint8(0), bag)
	assert.Equal(t, uint8(23), slot)
	assert.Equal(t, uint8(1), castCount)

	var spellID uint32
	require.NoError(t, r.Read(&spellID))
	assert.Equal(t, uint32(2054), spellID)

	var gotGUIDRaw uint64
	require.NoError(t, r.Read(&gotGUIDRaw))
	assert.Equal(t, guid, wow.GUID(gotGUIDRaw))

	var glyph uint32
	require.NoError(t, r.Read(&glyph))
	assert.Equal(t, uint32(0), glyph)

	var castFlags uint8
	require.NoError(t, r.Read(&castFlags))
	assert.Equal(t, uint8(0), castFlags)

	targets, err := readSpellCastTargets(r, nil)
	require.NoError(t, err)
	assert.Equal(t, TargetFlagNone, targets.TargetMask)
}

func TestCanUseItem_AliveAndClassRace(t *testing.T) {
	p := player.NewPlayer()
	p.Health = 100
	p.Level = 10
	p.Class = 1 // warrior
	p.Race = 1  // human

	tpl := &basedata.ItemTemplate{
		AllowableClass: -1,
		AllowableRace:  -1,
		RequiredLevel:  1,
	}
	assert.Equal(t, player.EquipResultOK, player.CanUseItem(p, nil, tpl))

	p.Health = 0
	assert.Equal(t, player.EquipResultYouAreDead, player.CanUseItem(p, nil, tpl))
	p.Health = 100

	tpl2 := &basedata.ItemTemplate{
		AllowableClass: int32(1) << 8,
		AllowableRace:  -1,
	}
	assert.Equal(t, player.EquipResultYouCanNeverUse, player.CanUseItem(p, nil, tpl2))

	tpl3 := &basedata.ItemTemplate{
		AllowableClass: -1,
		AllowableRace:  int32(1) << 7,
	}
	assert.Equal(t, player.EquipResultYouCanNeverUse, player.CanUseItem(p, nil, tpl3))

	tpl4 := &basedata.ItemTemplate{
		AllowableClass: -1,
		AllowableRace:  -1,
		RequiredLevel:  60,
	}
	assert.Equal(t, player.EquipResultCantEquipLevel, player.CanUseItem(p, nil, tpl4))

	assert.Equal(t, player.EquipResultItemNotFound, player.CanUseItem(p, nil, nil))
}

func TestItemSoulBound(t *testing.T) {
	item := player.NewItem(12345, wow.NewPlayerGUID(1))
	assert.False(t, item.IsSoulBound())

	item.SetBinding(true)
	assert.True(t, item.IsSoulBound())

	item.SetBinding(false)
	assert.False(t, item.IsSoulBound())
}

func TestSpellInfoCanBeUsedInCombat(t *testing.T) {
	si := &SpellInfo{Attributes: 0}
	assert.True(t, si.CanBeUsedInCombat())

	si.Attributes = SpellAttr0NotInCombatOnlyPeaceful
	assert.False(t, si.CanBeUsedInCombat())
}
