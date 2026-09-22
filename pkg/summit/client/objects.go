package client

import (
	"math"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// objectEnd is the size of the shared object header (ObjectField*), mirroring
// pkg/summit/world/object/update_fields.gen.go.
const objectEnd = 0x0006

// Update field word indices (3.3.5a).
const (
	fieldEntry = 0x0003

	unitFieldHealth          = objectEnd + 0x0012
	unitFieldMaxHealth       = objectEnd + 0x001a
	unitFieldLevel           = objectEnd + 0x0030
	unitFieldFactionTemplate = objectEnd + 0x0031
	unitFieldFlags           = objectEnd + 0x0035
	unitFieldDisplayID       = objectEnd + 0x003d
	unitFieldNpcFlags        = objectEnd + 0x004c
)

// UNIT_FIELD_FLAGS bits that make a unit non-attackable.
const (
	unitFlagNonAttackable  = 0x00000002
	unitFlagNotAttackable  = 0x00000004
	unitFlagNotAttackable1 = 0x00000080
)

// SMSG_UPDATE_OBJECT block types.
const (
	updateTypeValues        = 0
	updateTypeMovement      = 1
	updateTypeCreateObject  = 2
	updateTypeCreateObject2 = 3
	updateTypeOutOfRange    = 4
	updateTypeNearObjects   = 5
)

// Update flags of a create / movement block (u16 in 3.3.5a).
const (
	updateFlagSelf               = 0x0001
	updateFlagTransport          = 0x0002
	updateFlagHasTarget          = 0x0004
	updateFlagUnknown            = 0x0008
	updateFlagLowGUID            = 0x0010
	updateFlagLiving             = 0x0020
	updateFlagStationaryPosition = 0x0040
	updateFlagVehicle            = 0x0080
	updateFlagPosition           = 0x0100
	updateFlagRotation           = 0x0200
)

// Movement flags carried by the living part of an update block.
const (
	movementFlagOnTransport     = 0x0200
	movementFlagFalling         = 0x1000
	movementFlagSwimming        = 0x200000
	movementFlagFlying          = 0x1000000
	movementFlagSplineElevation = 0x4000000
)

// HitInfo bits of SMSG_ATTACKERSTATEUPDATE.
const (
	hitInfoMiss          = 0x00000010
	hitInfoFullAbsorb    = 0x00000020
	hitInfoPartialAbsorb = 0x00000040
	hitInfoFullResist    = 0x00000080
	hitInfoPartialResist = 0x00000100
	hitInfoCritical      = 0x00000200
	hitInfoBlock         = 0x00002000
)

// Entity is a snapshot of a world object the client learned about from
// SMSG_UPDATE_OBJECT. Snapshots are copies, safe to read from another goroutine.
type Entity struct {
	GUID        wow.GUID
	Type        wow.TypeID
	UpdateFlags uint16
	Fields      []uint32
	Pos         player.WorldLocation
	HasPos      bool
	MoveFlags   uint32
	Name        string

	// DamageDealtBySelf accumulates the melee damage this client dealt to the
	// entity. The server does not broadcast NPC health, so the bot infers a
	// creature's death when this reaches MaxHealth.
	DamageDealtBySelf uint32
}

func (e *Entity) field(idx int) uint32 {
	if idx >= 0 && idx < len(e.Fields) {
		return e.Fields[idx]
	}

	return 0
}

// Health returns UNIT_FIELD_HEALTH.
func (e *Entity) Health() uint32 { return e.field(unitFieldHealth) }

// MaxHealth returns UNIT_FIELD_MAXHEALTH.
func (e *Entity) MaxHealth() uint32 { return e.field(unitFieldMaxHealth) }

// Level returns UNIT_FIELD_LEVEL.
func (e *Entity) Level() uint32 { return e.field(unitFieldLevel) }

// Entry returns OBJECT_FIELD_ENTRY (the creature template id).
func (e *Entity) Entry() uint32 { return e.field(fieldEntry) }

// NpcFlags returns UNIT_NPC_FLAGS.
func (e *Entity) NpcFlags() uint32 { return e.field(unitFieldNpcFlags) }

// UnitFlags returns UNIT_FIELD_FLAGS.
func (e *Entity) UnitFlags() uint32 { return e.field(unitFieldFlags) }

// DisplayID returns UNIT_FIELD_DISPLAYID.
func (e *Entity) DisplayID() uint32 { return e.field(unitFieldDisplayID) }

// FactionTemplate returns UNIT_FIELD_FACTIONTEMPLATE.
func (e *Entity) FactionTemplate() uint32 { return e.field(unitFieldFactionTemplate) }

// IsUnit reports whether the object is a unit (creature or player).
func (e *Entity) IsUnit() bool {
	return e.Type == wow.TypeIDUnit || e.Type == wow.TypeIDPlayer
}

// IsAlive reports whether the entity still has health. Objects without a
// health field are treated as alive.
func (e *Entity) IsAlive() bool {
	if e.Health() == 0 && e.MaxHealth() == 0 {
		return true
	}

	return e.Health() > 0
}

// SlainByDamage reports whether the bot's own melee brought the creature down.
// The server never sends NPC health updates, so this is the death signal.
func (e *Entity) SlainByDamage() bool {
	hp := e.MaxHealth()

	return hp > 0 && e.DamageDealtBySelf >= hp
}

// Attackable reports whether the entity is a valid melee target for the bot:
// a living creature without service NPC flags or non-attackable unit flags.
func (e *Entity) Attackable() bool {
	if e.Type != wow.TypeIDUnit || !e.HasPos {
		return false
	}

	if !e.IsAlive() || e.SlainByDamage() {
		return false
	}

	if e.NpcFlags() != 0 {
		return false
	}

	if e.UnitFlags()&(unitFlagNonAttackable|unitFlagNotAttackable|unitFlagNotAttackable1) != 0 {
		return false
	}

	return true
}

func (e *Entity) clone() *Entity {
	c := *e
	if e.Fields != nil {
		c.Fields = make([]uint32, len(e.Fields))
		copy(c.Fields, e.Fields)
	}

	return &c
}

// SelfGUID returns the GUID of the bot's own player (from the character login
// or the UPDATEFLAG_SELF create block).
func (wc *WorldClient) SelfGUID() wow.GUID {
	wc.objectsMu.RLock()
	defer wc.objectsMu.RUnlock()

	return wc.selfGUID
}

// SelfEntity returns a snapshot of the bot's own player object, or nil.
func (wc *WorldClient) SelfEntity() *Entity {
	return wc.Entity(wc.SelfGUID())
}

// Entity returns a snapshot of an object by GUID, or nil.
func (wc *WorldClient) Entity(guid wow.GUID) *Entity {
	wc.objectsMu.RLock()
	defer wc.objectsMu.RUnlock()

	if e, ok := wc.objects[guid]; ok {
		return e.clone()
	}

	return nil
}

// Entities returns snapshots of every known object.
func (wc *WorldClient) Entities() []*Entity {
	wc.objectsMu.RLock()
	defer wc.objectsMu.RUnlock()

	out := make([]*Entity, 0, len(wc.objects))
	for _, e := range wc.objects {
		out = append(out, e.clone())
	}

	return out
}

// NearestAttackable returns the closest attackable creature within maxDist of
// from, or nil.
func (wc *WorldClient) NearestAttackable(from *player.WorldLocation, maxDist float32) *Entity {
	if from == nil {
		return nil
	}

	wc.objectsMu.RLock()
	defer wc.objectsMu.RUnlock()

	var (
		best     *Entity
		bestDist = maxDist
	)

	for _, e := range wc.objects {
		if !e.Attackable() {
			continue
		}

		d := distance2D(from.X, from.Y, e.Pos.X, e.Pos.Y)
		if d < bestDist {
			bestDist = d
			best = e
		}
	}

	if best == nil {
		return nil
	}

	return best.clone()
}

func distance2D(x1, y1, x2, y2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1

	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

func (wc *WorldClient) ensureEntityLocked(guid wow.GUID, typeID wow.TypeID) *Entity {
	e := wc.objects[guid]
	if e == nil {
		e = &Entity{GUID: guid, Type: typeID}
		wc.objects[guid] = e
	} else if typeID != wow.TypeIDObject {
		e.Type = typeID
	}

	return e
}

// setFieldsLocked merges an update-field values block into an entity.
func (wc *WorldClient) setFieldsLocked(e *Entity, values map[uint32]uint32) {
	for idx, v := range values {
		if int(idx) >= len(e.Fields) {
			grown := make([]uint32, idx+1)
			copy(grown, e.Fields)
			e.Fields = grown
		}

		e.Fields[idx] = v
	}
}

// handleUpdateObject decodes SMSG_UPDATE_OBJECT.
func (wc *WorldClient) handleUpdateObject(msg *ServerMessage) {
	r := msg.Reader()

	var blockCount uint32
	if err := r.Read(&blockCount); err != nil {
		wc.log.Error().Err(err).Msg("cannot read update block count")
		return
	}

	for i := uint32(0); i < blockCount; i++ {
		var blockType uint8
		if err := r.Read(&blockType); err != nil {
			return
		}

		switch blockType {
		case updateTypeValues:
			wc.readValuesUpdate(r)
		case updateTypeMovement:
			wc.readMovementUpdate(r)
		case updateTypeCreateObject, updateTypeCreateObject2:
			wc.readCreateUpdate(r)
		case updateTypeOutOfRange:
			var n uint32
			if err := r.Read(&n); err != nil {
				return
			}

			guids := make([]wow.GUID, 0, n)
			for k := uint32(0); k < n; k++ {
				g, err := wow.ReadPackedGUID(r)
				if err != nil {
					return
				}

				guids = append(guids, g)
			}

			wc.removeEntities(guids, false)
		case updateTypeNearObjects:
			var n uint32
			if err := r.Read(&n); err != nil {
				return
			}
			for k := uint32(0); k < n; k++ {
				if _, err := wow.ReadPackedGUID(r); err != nil {
					return
				}
			}
		default:
			wc.log.Debug().Uint8("type", blockType).Msg("unknown update block type, dropping rest of packet")
			return
		}
	}
}

func (wc *WorldClient) readValuesUpdate(r *wow.PacketReader) {
	var guid uint64
	if err := r.Read(&guid); err != nil {
		return
	}

	values := readValuesBlock(r)

	wc.objectsMu.Lock()
	e := wc.ensureEntityLocked(wow.GUID(guid), wow.TypeIDObject)
	wc.setFieldsLocked(e, values)
	wc.objectsMu.Unlock()
}

func (wc *WorldClient) readMovementUpdate(r *wow.PacketReader) {
	var guid uint64
	if err := r.Read(&guid); err != nil {
		return
	}

	var flags uint16
	if err := r.Read(&flags); err != nil {
		return
	}

	loc, moveFlags, hasPos := readMovementBlock(r, flags)

	wc.objectsMu.Lock()
	e := wc.ensureEntityLocked(wow.GUID(guid), wow.TypeIDObject)
	if hasPos {
		e.Pos = loc
		e.HasPos = true
		e.MoveFlags = moveFlags
	}
	wc.objectsMu.Unlock()
}

func (wc *WorldClient) readCreateUpdate(r *wow.PacketReader) {
	var (
		guid   uint64
		typeID uint8
		flags  uint16
	)

	if err := r.Read(&guid); err != nil {
		return
	}

	if err := r.Read(&typeID); err != nil {
		return
	}

	if err := r.Read(&flags); err != nil {
		return
	}

	loc, moveFlags, hasPos := readMovementBlock(r, flags)
	values := readValuesBlock(r)

	isSelf := flags&updateFlagSelf != 0

	var (
		needCreatureQuery bool
		entry             uint32
	)

	wc.objectsMu.Lock()
	e := wc.ensureEntityLocked(wow.GUID(guid), wow.TypeID(typeID))
	e.UpdateFlags = flags

	if hasPos {
		e.Pos = loc
		e.HasPos = true
		e.MoveFlags = moveFlags
	}

	wc.setFieldsLocked(e, values)

	if isSelf {
		wc.selfGUID = wow.GUID(guid)
	}

	if e.Type == wow.TypeIDUnit {
		entry = e.Entry()
		if entry != 0 && e.Name == "" {
			if name, ok := wc.creatureNames[entry]; ok {
				e.Name = name
			} else {
				needCreatureQuery = true
			}
		}
	}

	wc.objectsMu.Unlock()

	if needCreatureQuery {
		wc.QueryCreature(entry, wow.GUID(guid))
	}
}

// removeEntities drops entities from the object manager.
func (wc *WorldClient) removeEntities(guids []wow.GUID, onDeath bool) {
	wc.objectsMu.Lock()
	defer wc.objectsMu.Unlock()

	for _, g := range guids {
		_ = onDeath // the death flag only affects animations client-side

		delete(wc.objects, g)

		if g == wc.selfGUID {
			wc.selfGUID = 0
		}
	}
}

// handleDestroyObject decodes SMSG_DESTROY_OBJECT (u64 guid, optional u8 onDeath).
func (wc *WorldClient) handleDestroyObject(msg *ServerMessage) {
	r := msg.Reader()

	var guid uint64
	if err := r.Read(&guid); err != nil {
		return
	}

	onDeath := false
	if r.ReadedCount() < len(msg.Data) {
		var b uint8
		if r.Read(&b) == nil {
			onDeath = b != 0
		}
	}

	wc.removeEntities([]wow.GUID{wow.GUID(guid)}, onDeath)
}

// handleNameQueryResponse decodes SMSG_NAME_QUERY_RESPONSE.
func (wc *WorldClient) handleNameQueryResponse(msg *ServerMessage) {
	r := msg.Reader()

	guid, err := wow.ReadPackedGUID(r)
	if err != nil {
		return
	}

	var status uint8
	if err := r.Read(&status); err != nil || status != 0 {
		return
	}

	var name string
	if err := r.ReadString(&name); err != nil {
		return
	}

	wc.objectsMu.Lock()
	if e, ok := wc.objects[guid]; ok {
		e.Name = name
	}
	wc.objectsMu.Unlock()

	wc.log.Debug().Str("name", name).Msg("resolved player name")
}

// handleCreatureQueryResponse decodes SMSG_CREATURE_QUERY_RESPONSE and names
// every known creature of that entry.
func (wc *WorldClient) handleCreatureQueryResponse(msg *ServerMessage) {
	r := msg.Reader()

	var entry uint32
	if err := r.Read(&entry); err != nil {
		return
	}

	if entry&0x80000000 != 0 {
		return // unknown creature
	}

	var name, unused1, unused2, unused3, subName, icon string
	_ = r.ReadString(&name)
	_ = r.ReadString(&unused1)
	_ = r.ReadString(&unused2)
	_ = r.ReadString(&unused3)
	_ = r.ReadString(&subName)
	_ = r.ReadString(&icon)

	wc.objectsMu.Lock()
	wc.creatureNames[entry] = name
	for _, e := range wc.objects {
		if e.Type == wow.TypeIDUnit && e.Entry() == entry {
			e.Name = name
		}
	}
	wc.objectsMu.Unlock()

	wc.log.Debug().Uint32("entry", entry).Str("name", name).Msg("resolved creature name")
}

// handleMonsterMove is a placeholder for SMSG_MONSTER_MOVE; NPC positions from
// movement packets are not tracked yet (stationary creatures are unaffected).
func (wc *WorldClient) handleMonsterMove(msg *ServerMessage) {
	wc.log.Trace().Int("size", len(msg.Data)).Msg("monster move")
}

// addDamageDealt records melee damage the bot dealt to a victim.
func (wc *WorldClient) addDamageDealt(victim wow.GUID, damage uint32) {
	wc.objectsMu.Lock()
	defer wc.objectsMu.Unlock()

	if e, ok := wc.objects[victim]; ok {
		e.DamageDealtBySelf += damage
	}
}

// readValuesBlock decodes: u8 blockCount, blockCount×u32 mask, then a u32 per
// set bit.
func readValuesBlock(r *wow.PacketReader) map[uint32]uint32 {
	var blockCount uint8
	if err := r.Read(&blockCount); err != nil {
		return nil
	}

	mask := make([]uint32, blockCount)
	for i := range mask {
		if err := r.Read(&mask[i]); err != nil {
			return nil
		}
	}

	values := make(map[uint32]uint32)

	for block := 0; block < int(blockCount); block++ {
		bits := mask[block]
		if bits == 0 {
			continue
		}

		for bit := 0; bit < 32; bit++ {
			if bits&(1<<uint(bit)) == 0 {
				continue
			}

			var v uint32
			if err := r.Read(&v); err != nil {
				return values
			}

			values[uint32(block*32+bit)] = v
		}
	}

	return values
}

// readMovementBlock consumes the movement part of a create/movement block and
// returns the position (when present), the movement flags and whether a
// position was decoded. It mirrors object.WriteMovementBlock on the server.
func readMovementBlock(r *wow.PacketReader, flags uint16) (player.WorldLocation, uint32, bool) {
	var (
		loc       player.WorldLocation
		moveFlags uint32
		hasPos    bool
	)

	switch {
	case flags&updateFlagLiving != 0:
		var (
			flags2 uint16
			t      uint32
		)

		_ = r.Read(&moveFlags)
		_ = r.Read(&flags2)
		_ = r.Read(&t)
		_ = r.Read(&loc.X)
		_ = r.Read(&loc.Y)
		_ = r.Read(&loc.Z)
		_ = r.Read(&loc.O)
		hasPos = true

		if moveFlags&movementFlagOnTransport != 0 {
			_, _ = wow.ReadPackedGUID(r)

			for i := 0; i < 4; i++ {
				var f float32
				_ = r.Read(&f)
			}

			var (
				transportTime uint32
				seat          uint8
			)
			_ = r.Read(&transportTime)
			_ = r.Read(&seat)
		}

		if moveFlags&(movementFlagSwimming|movementFlagFlying) != 0 {
			var pitch float32
			_ = r.Read(&pitch)
		}

		var fallTime uint32
		_ = r.Read(&fallTime)

		if moveFlags&movementFlagFalling != 0 {
			for i := 0; i < 4; i++ {
				var f float32
				_ = r.Read(&f)
			}
		}

		if moveFlags&movementFlagSplineElevation != 0 {
			var f float32
			_ = r.Read(&f)
		}

		for i := 0; i < 9; i++ {
			var f float32
			_ = r.Read(&f)
		}

	case flags&updateFlagPosition != 0:
		_, _ = wow.ReadPackedGUID(r)

		for i := 0; i < 3; i++ {
			var f float32
			_ = r.Read(&f)
		}

		_ = r.Read(&loc.X)
		_ = r.Read(&loc.Y)
		_ = r.Read(&loc.Z)
		_ = r.Read(&loc.O)
		hasPos = true

		var corpseO float32
		_ = r.Read(&corpseO)

	case flags&updateFlagStationaryPosition != 0:
		_ = r.Read(&loc.X)
		_ = r.Read(&loc.Y)
		_ = r.Read(&loc.Z)
		_ = r.Read(&loc.O)
		hasPos = true
	}

	if flags&updateFlagUnknown != 0 {
		var v uint32
		_ = r.Read(&v)
	}

	if flags&updateFlagLowGUID != 0 {
		var v uint32
		_ = r.Read(&v)
	}

	if flags&updateFlagHasTarget != 0 {
		_, _ = wow.ReadPackedGUID(r)
	}

	if flags&updateFlagTransport != 0 {
		var v uint32
		_ = r.Read(&v)
	}

	if flags&updateFlagVehicle != 0 {
		var (
			v uint32
			f float32
		)
		_ = r.Read(&v)
		_ = r.Read(&f)
	}

	if flags&updateFlagRotation != 0 {
		var v uint64
		_ = r.Read(&v)
	}

	return loc, moveFlags, hasPos
}
