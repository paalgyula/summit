package client

import (
	"bytes"
	"testing"

	"github.com/paalgyula/summit/pkg/wow"
)

func TestCastSpellPacket(t *testing.T) {
	wc := newTestClient()

	target := wow.NewGUID(wow.UnitGUID, 9)
	wc.CastSpell(133, target)

	select {
	case pkt := <-wc.clientMessages:
		if pkt.Opcode() != wow.ClientCastSpell {
			t.Fatalf("expected CMSG_CAST_SPELL, got %s", pkt.Opcode())
		}

		r := wow.NewPacketReader(pkt.Bytes())

		var castCount uint8
		_ = r.Read(&castCount)

		var spell uint32
		_ = r.Read(&spell)

		var castFlags uint8
		_ = r.Read(&castFlags)

		var mask uint32
		_ = r.Read(&mask)

		got, err := wow.ReadPackedGUID(r)
		if err != nil {
			t.Fatalf("target guid: %v", err)
		}

		if castCount != 0 || spell != 133 || castFlags != 0 {
			t.Fatalf("unexpected header: count=%d spell=%d flags=%d", castCount, spell, castFlags)
		}

		if mask != castTargetFlagUnit {
			t.Fatalf("expected unit target mask, got 0x%x", mask)
		}

		if got != target {
			t.Fatalf("expected target %#x, got %#x", uint64(target), uint64(got))
		}
	default:
		t.Fatal("no packet was sent")
	}
}

func TestCastFailedIsRecorded(t *testing.T) {
	wc := newTestClient()

	var buf bytes.Buffer
	writeU8(&buf, 0)
	writeU32(&buf, 116)
	writeU8(&buf, spellFailedOutOfRange)
	writeU8(&buf, 0)

	wc.handleCastFailed(&ServerMessage{Opcode: wow.ServerCastFailed, Data: buf.Bytes()})

	f, ok := wc.SpellFailed(116)
	if !ok || f.Result != spellFailedOutOfRange {
		t.Fatalf("expected failure recorded, got %+v ok=%v", f, ok)
	}
}

func TestReclaimCorpsePacket(t *testing.T) {
	wc := newTestClient()
	wc.ReclaimCorpse(0)

	pkt := <-wc.clientMessages
	if pkt.Opcode() != wow.ClientReclaimCorpse {
		t.Fatalf("expected CMSG_RECLAIM_CORPSE, got %s", pkt.Opcode())
	}

	if len(pkt.Bytes()) != 8 {
		t.Fatalf("expected 8-byte guid, got %d bytes", len(pkt.Bytes()))
	}
}

func TestDeathReleaseLocSetsDead(t *testing.T) {
	wc := newTestClient()

	var buf bytes.Buffer
	writeU32(&buf, 1)
	writeF32(&buf, -612.5)
	writeF32(&buf, -4251.5)
	writeF32(&buf, 38.5)

	wc.handleDeathReleaseLoc(&ServerMessage{Opcode: wow.ServerDeathReleaseLoc, Data: buf.Bytes()})

	if !wc.IsDead() {
		t.Fatal("expected IsDead() to be true after SMSG_DEATH_RELEASE_LOC")
	}

	if loc := wc.DeathLocation(); loc.Map != 1 || loc.X != -612.5 {
		t.Fatalf("unexpected death location: %+v", loc)
	}

	// Health coming back clears the flag.
	wc.setSelfGUID(wow.NewGUID(wow.PlayerGUID, 1))
	guid := uint64(wow.NewGUID(wow.PlayerGUID, 1))

	var vu bytes.Buffer
	writeU32(&vu, 1)
	writeU8(&vu, updateTypeValues)
	writeU64(&vu, guid)

	f := make([]uint32, unitFieldMaxHealth+1)
	f[unitFieldHealth] = 50
	f[unitFieldMaxHealth] = 100
	writeValues(&vu, fieldMap(f))

	wc.handleUpdateObject(&ServerMessage{Opcode: wow.ServerUpdateObject, Data: vu.Bytes()})

	if wc.IsDead() {
		t.Fatal("expected IsDead() to clear once health is restored")
	}
}

// fieldMap converts a sparse field array into the map writeValues expects.
func fieldMap(fields []uint32) map[uint32]uint32 {
	out := make(map[uint32]uint32)
	for i, v := range fields {
		if v != 0 {
			out[uint32(i)] = v
		}
	}

	return out
}
