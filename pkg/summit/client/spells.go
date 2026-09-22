package client

import (
	"time"

	"github.com/paalgyula/summit/pkg/wow"
)

// CMSG_CAST_SPELL target mask bits (pkg/summit/world/spell_cast.go).
const (
	castTargetFlagUnit        = 0x00000002
	castTargetFlagItem        = 0x00000010
	castTargetFlagSourceLoc   = 0x00000020
	castTargetFlagDestLoc     = 0x00000040
	castTargetFlagGameObject  = 0x00000800
	castTargetFlagTradeItem   = 0x00001000
	castTargetFlagString      = 0x00002000
	castTargetFlagCorpseEnemy = 0x00000200
	castTargetFlagCorpseAlly  = 0x00008000
	castTargetFlagUnitMinipet = 0x00010000
)

// SpellCastResult values the bot cares about.
const (
	spellFailedOutOfRange = 0x07
	spellFailedNotReady   = 0x3A
	spellFailedNoTarget   = 0x10
)

// CastSpell sends CMSG_CAST_SPELL targeting a unit.
func (wc *WorldClient) CastSpell(spellID uint32, target wow.GUID) {
	pkt := wow.NewPacket(wow.ClientCastSpell)
	_ = pkt.WriteOne(0)    // cast count
	_ = pkt.Write(spellID) // spell id
	_ = pkt.WriteOne(0)    // cast flags
	_ = pkt.Write(uint32(castTargetFlagUnit))
	pkt.WriteBytes(target.Pack())
	wc.Send(pkt)
}

// CastSpellSelf sends a self-targeted CMSG_CAST_SPELL (empty target mask).
func (wc *WorldClient) CastSpellSelf(spellID uint32) {
	pkt := wow.NewPacket(wow.ClientCastSpell)
	_ = pkt.WriteOne(0)
	_ = pkt.Write(spellID)
	_ = pkt.WriteOne(0)
	_ = pkt.Write(uint32(0))
	wc.Send(pkt)
}

// KnownSpells returns the spellbook learned from SMSG_INITIAL_SPELLS.
func (wc *WorldClient) KnownSpells() []uint32 {
	p := wc.Player()
	if p == nil {
		return nil
	}

	out := make([]uint32, len(p.KnownSpells))
	copy(out, p.KnownSpells)

	return out
}

// SpellFailure is the last cast rejection the server reported for a spell.
type SpellFailure struct {
	Result uint8
	At     time.Time
}

// SpellFailed returns the last SpellCastResult the server reported for a spell.
func (wc *WorldClient) SpellFailed(spellID uint32) (SpellFailure, bool) {
	wc.objectsMu.RLock()
	defer wc.objectsMu.RUnlock()

	f, ok := wc.spellFailures[spellID]

	return f, ok
}

func (wc *WorldClient) recordSpellFailure(spellID uint32, result uint8) {
	wc.objectsMu.Lock()
	wc.spellFailures[spellID] = SpellFailure{Result: result, At: time.Now()}
	wc.objectsMu.Unlock()
}

// handleCastFailed decodes SMSG_CAST_FAILED and records the rejection so the
// bot backs the spell off.
func (wc *WorldClient) handleCastFailed(msg *ServerMessage) {
	spellID, result, ok := parseCastFailed(msg.Data)
	if !ok {
		return
	}

	wc.recordSpellFailure(spellID, result)

	wc.log.Debug().
		Uint32("spell", spellID).
		Uint8("result", result).
		Msg("spell cast failed")
}

// parseCastFailed accepts both wire layouts seen in the wild:
//   - the 3.3.5a layout (u8 castCount, u32 spell, u8 result, u8 multipleCasts),
//   - the legacy all-u32 layout emitted by older server builds
//     (u32 castCount, u32 spell, u32 result).
func parseCastFailed(data []byte) (uint32, uint8, bool) {
	r := wow.NewPacketReader(data)

	if len(data) >= 12 {
		var castCount, spell, result uint32
		_ = r.Read(&castCount)
		_ = r.Read(&spell)
		_ = r.Read(&result)

		return spell, uint8(result), true
	}

	if len(data) >= 7 {
		var (
			castCount uint8
			spell     uint32
			result    uint8
		)
		_ = r.Read(&castCount)
		_ = r.Read(&spell)
		_ = r.Read(&result)

		return spell, result, true
	}

	return 0, 0, false
}

// handleSpellStart decodes SMSG_SPELL_START. Only the head is consumed; the
// target list is skipped.
func (wc *WorldClient) handleSpellStart(msg *ServerMessage) {
	r := msg.Reader()

	if _, err := wow.ReadPackedGUID(r); err != nil { // cast item
		return
	}

	if _, err := wow.ReadPackedGUID(r); err != nil { // caster
		return
	}

	var castCount uint8
	_ = r.Read(&castCount)

	var spellID uint32
	_ = r.Read(&spellID)

	var flags, timer uint32
	_ = r.Read(&flags)
	_ = r.Read(&timer)

	wc.log.Debug().
		Uint32("spell", spellID).
		Uint32("castTime", timer).
		Msg("spell cast started")
}

// handleSpellGo decodes SMSG_SPELL_GO's head (caster + spell id).
func (wc *WorldClient) handleSpellGo(msg *ServerMessage) {
	r := msg.Reader()

	if _, err := wow.ReadPackedGUID(r); err != nil { // cast item
		return
	}

	if _, err := wow.ReadPackedGUID(r); err != nil { // caster
		return
	}

	var extraCasts uint8
	_ = r.Read(&extraCasts)

	var spellID uint32
	_ = r.Read(&spellID)

	wc.log.Debug().Uint32("spell", spellID).Msg("spell cast completed")
}

// handleSpellFailure decodes SMSG_SPELL_FAILURE: u64? The 3.3.5a form is
// caster (packed) + u32 spell + u8 reason in some cores; keep it defensive.
func (wc *WorldClient) handleSpellFailure(msg *ServerMessage) {
	r := msg.Reader()

	caster, err := wow.ReadPackedGUID(r)
	if err != nil {
		return
	}

	var spellID uint32
	_ = r.Read(&spellID)

	wc.log.Debug().
		Uint64("caster", uint64(caster)).
		Uint32("spell", spellID).
		Msg("spell failure")
}
