package client

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

func writeU8(buf *bytes.Buffer, v uint8)   { _ = buf.WriteByte(v) }
func writeU16(buf *bytes.Buffer, v uint16) { _ = binary.Write(buf, binary.LittleEndian, v) }
func writeU32(buf *bytes.Buffer, v uint32) { _ = binary.Write(buf, binary.LittleEndian, v) }
func writeU64(buf *bytes.Buffer, v uint64) { _ = binary.Write(buf, binary.LittleEndian, v) }
func writeF32(buf *bytes.Buffer, v float32) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// writeValues encodes an update-field values block with the given fields.
func writeValues(buf *bytes.Buffer, fields map[uint32]uint32) {
	blockCount := uint8(0)
	for idx := range fields {
		if b := idx/32 + 1; uint8(b) > blockCount {
			blockCount = uint8(b)
		}
	}

	mask := make([]uint32, blockCount)
	for idx := range fields {
		mask[idx/32] |= 1 << (idx % 32)
	}

	writeU8(buf, blockCount)
	for _, m := range mask {
		writeU32(buf, m)
	}

	for block := 0; block < int(blockCount); block++ {
		for bit := 0; bit < 32; bit++ {
			if mask[block]&(1<<uint(bit)) == 0 {
				continue
			}

			writeU32(buf, fields[uint32(block*32+bit)])
		}
	}
}

// creatureCreateBlock builds an SMSG_UPDATE_OBJECT payload with a single
// creature create block, mirroring map.sendNPCCreateToPlayer.
func creatureCreateBlock(guid wow.GUID, entry, health, maxHealth, level, npcFlags uint32, pos player.WorldLocation) []byte {
	var buf bytes.Buffer

	writeU32(&buf, 1)            // block count
	writeU8(&buf, 2)             // UPDATETYPE_CREATE_OBJECT
	writeU64(&buf, uint64(guid)) // raw GUID
	writeU8(&buf, uint8(wow.TypeIDUnit))
	writeU16(&buf, updateFlagLowGUID|updateFlagLiving|updateFlagStationaryPosition)

	// movement block (LIVING)
	writeU32(&buf, 0) // movement flags
	writeU16(&buf, 0) // flags2
	writeU32(&buf, 0) // time
	writeF32(&buf, pos.X)
	writeF32(&buf, pos.Y)
	writeF32(&buf, pos.Z)
	writeF32(&buf, pos.O)
	writeU32(&buf, 0) // fall time
	for i := 0; i < 9; i++ {
		writeF32(&buf, 0) // speeds
	}
	writeU32(&buf, 0x0B) // low GUID

	writeValues(&buf, map[uint32]uint32{
		fieldEntry:         entry,
		unitFieldHealth:    health,
		unitFieldMaxHealth: maxHealth,
		unitFieldLevel:     level,
		unitFieldNpcFlags:  npcFlags,
	})

	return buf.Bytes()
}

func TestDecodeCreatureCreate(t *testing.T) {
	wc := newTestClient()

	guid := wow.NewGUID(wow.UnitGUID, 4242)
	pos := player.WorldLocation{X: -8949.95, Y: -132.66, Z: 83.53, O: 1, Map: 0}

	wc.handleUpdateObject(&ServerMessage{
		Opcode: wow.ServerUpdateObject,
		Data:   creatureCreateBlock(guid, 100001, 50, 50, 1, 0, pos),
	})

	e := wc.Entity(guid)
	if e == nil {
		t.Fatal("creature was not added to the object manager")
	}

	if e.Type != wow.TypeIDUnit {
		t.Fatalf("expected unit, got %v", e.Type)
	}

	if !e.HasPos || e.Pos.X != pos.X || e.Pos.Y != pos.Y {
		t.Fatalf("unexpected position: %+v", e.Pos)
	}

	if e.Entry() != 100001 || e.MaxHealth() != 50 || e.Level() != 1 {
		t.Fatalf("unexpected fields: entry=%d hp=%d maxhp=%d level=%d",
			e.Entry(), e.Health(), e.MaxHealth(), e.Level())
	}

	if !e.Attackable() {
		t.Fatal("creature with no npc flags should be attackable")
	}
}

func TestServiceNPCNotAttackable(t *testing.T) {
	wc := newTestClient()

	guid := wow.NewGUID(wow.UnitGUID, 7)
	pos := player.WorldLocation{X: 1, Y: 2, Z: 3, Map: 0}

	wc.handleUpdateObject(&ServerMessage{
		Opcode: wow.ServerUpdateObject,
		Data:   creatureCreateBlock(guid, 123, 100, 100, 5, 0x80 /* vendor */, pos),
	})

	if wc.Entity(guid).Attackable() {
		t.Fatal("vendor NPC must not be attackable")
	}
}

func TestValuesUpdateChangesHealth(t *testing.T) {
	wc := newTestClient()

	guid := wow.NewGUID(wow.UnitGUID, 9)
	pos := player.WorldLocation{X: 5, Y: 6, Z: 7, Map: 0}
	wc.handleUpdateObject(&ServerMessage{
		Opcode: wow.ServerUpdateObject,
		Data:   creatureCreateBlock(guid, 100001, 50, 50, 1, 0, pos),
	})

	var buf bytes.Buffer
	writeU32(&buf, 1) // block count
	writeU8(&buf, updateTypeValues)
	writeU64(&buf, uint64(guid))
	writeValues(&buf, map[uint32]uint32{unitFieldHealth: 20})

	wc.handleUpdateObject(&ServerMessage{Opcode: wow.ServerUpdateObject, Data: buf.Bytes()})

	if got := wc.Entity(guid).Health(); got != 20 {
		t.Fatalf("expected health 20, got %d", got)
	}
}

func TestNearestAttackablePicksClosest(t *testing.T) {
	wc := newTestClient()

	far := wow.NewGUID(wow.UnitGUID, 1)
	near := wow.NewGUID(wow.UnitGUID, 2)

	wc.handleUpdateObject(&ServerMessage{
		Data: creatureCreateBlock(far, 1, 10, 10, 1, 0, player.WorldLocation{X: 40, Y: 0, Map: 0}),
	})
	wc.handleUpdateObject(&ServerMessage{
		Data: creatureCreateBlock(near, 1, 10, 10, 1, 0, player.WorldLocation{X: 5, Y: 0, Map: 0}),
	})

	from := player.WorldLocation{X: 0, Y: 0, Map: 0}

	got := wc.NearestAttackable(&from, 100)
	if got == nil || got.GUID != near {
		t.Fatalf("expected the near creature, got %+v", got)
	}

	if wc.NearestAttackable(&from, 3) != nil {
		t.Fatal("expected no target within 3 yards")
	}
}

func TestSlainByDamage(t *testing.T) {
	e := &Entity{ //nolint:exhaustruct
		Type: wow.TypeIDUnit,
		Fields: func() []uint32 {
			f := make([]uint32, unitFieldMaxHealth+1)
			f[unitFieldMaxHealth] = 50
			return f
		}(),
		DamageDealtBySelf: 30,
	}

	if e.SlainByDamage() {
		t.Fatal("30/50 damage is not a kill")
	}

	e.DamageDealtBySelf = 50

	if !e.SlainByDamage() {
		t.Fatal("50/50 damage should be a kill")
	}

	if e.Attackable() {
		t.Fatal("a slain creature is not attackable")
	}
}

func TestDestroyObjectRemovesEntity(t *testing.T) {
	wc := newTestClient()

	guid := wow.NewGUID(wow.UnitGUID, 11)
	wc.handleUpdateObject(&ServerMessage{
		Data: creatureCreateBlock(guid, 1, 10, 10, 1, 0, player.WorldLocation{X: 1, Y: 1, Map: 0}),
	})

	var buf bytes.Buffer
	writeU64(&buf, uint64(guid))
	writeU8(&buf, 1) // on death

	wc.handleDestroyObject(&ServerMessage{Opcode: wow.ServerDestroyObject, Data: buf.Bytes()})

	if wc.Entity(guid) != nil {
		t.Fatal("entity should have been destroyed")
	}
}
