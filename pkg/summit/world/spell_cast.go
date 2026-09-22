package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleCastSpell handles CMSG_CAST_SPELL from the client.
// SpellCastTargets masks of CMSG_CAST_SPELL / CMSG_USE_ITEM (AC SpellInfo.h TargetFlags).
const (
	TargetFlagNone           uint32 = 0x00000000
	TargetFlagUnit           uint32 = 0x00000002
	TargetFlagItem           uint32 = 0x00000010
	TargetFlagSourceLocation uint32 = 0x00000020
	TargetFlagDestLocation   uint32 = 0x00000040
	TargetFlagUnitEnemy      uint32 = 0x00000080
	TargetFlagUnitAlly       uint32 = 0x00000100
	TargetFlagCorpseEnemy    uint32 = 0x00000200
	TargetFlagGameobject     uint32 = 0x00000800
	TargetFlagTradeItem      uint32 = 0x00001000
	TargetFlagString         uint32 = 0x00002000
	TargetFlagCorpseAlly     uint32 = 0x00008000
	TargetFlagUnitMinipet    uint32 = 0x00010000
)

// objectTargetMask returns the mask bits that carry a packed object GUID on the wire.
func objectTargetMask(mask uint32) uint32 {
	return mask & (TargetFlagUnit | TargetFlagUnitMinipet | TargetFlagGameobject |
		TargetFlagCorpseEnemy | TargetFlagCorpseAlly)
}

// itemTargetMask returns the mask bits that carry a packed item GUID on the wire.
func itemTargetMask(mask uint32) uint32 {
	return mask & (TargetFlagItem | TargetFlagTradeItem)
}

// readSpellCastTargets reads the SpellCastTargets payload from a packet reader
// (AC SpellCastTargets::Read). resolve maps a unit/object GUID to a Unit, or
// may return nil when the target is unknown.
func readSpellCastTargets(reader *wow.PacketReader, resolve func(wow.GUID) Unit) (*SpellCastTargets, error) {
	var targetMask uint32
	if err := reader.Read(&targetMask); err != nil {
		return nil, err
	}

	targets := &SpellCastTargets{TargetMask: targetMask}
	if targetMask == TargetFlagNone {
		return targets, nil
	}

	if objectTargetMask(targetMask) != 0 {
		guid, err := wow.ReadPackedGUID(reader)
		if err != nil {
			return nil, err
		}
		if resolve != nil {
			targets.UnitTarget = resolve(guid)
		}
	}

	if itemTargetMask(targetMask) != 0 {
		// Packed item GUID; item targeting is not modelled yet — consume bytes.
		if _, err := wow.ReadPackedGUID(reader); err != nil {
			return nil, err
		}
	}

	readXYZ := func() (Position, error) {
		var pos Position
		if err := reader.Read(&pos.X); err != nil {
			return pos, err
		}
		if err := reader.Read(&pos.Y); err != nil {
			return pos, err
		}
		if err := reader.Read(&pos.Z); err != nil {
			return pos, err
		}
		return pos, nil
	}

	if targetMask&TargetFlagSourceLocation != 0 {
		transport, err := wow.ReadPackedGUID(reader)
		if err != nil {
			return nil, err
		}
		pos, err := readXYZ()
		if err != nil {
			return nil, err
		}
		_ = transport // transports are not modelled yet
		targets.SrcPosition = pos
	}

	if targetMask&TargetFlagDestLocation != 0 {
		transport, err := wow.ReadPackedGUID(reader)
		if err != nil {
			return nil, err
		}
		pos, err := readXYZ()
		if err != nil {
			return nil, err
		}
		_ = transport
		targets.DstPosition = pos
	}

	if targetMask&TargetFlagString != 0 {
		var s string
		if err := reader.ReadString(&s); err != nil {
			return nil, err
		}
	}

	return targets, nil
}

// readClientCastFlags reads the optional projectile payload when castFlags&0x02
// (AC HandleClientCastFlags). Movement data is consumed and ignored for now.
func readClientCastFlags(reader *wow.PacketReader, castFlags uint8) error {
	if castFlags&0x02 == 0 {
		return nil
	}

	var elevation, speed float32
	if err := reader.Read(&elevation); err != nil {
		return err
	}
	if err := reader.Read(&speed); err != nil {
		return err
	}

	var hasMovement uint8
	if err := reader.Read(&hasMovement); err != nil {
		return err
	}

	// Embedded movement packet: u32 opcode + body. Body size is not known
	// without a full opcode table walk; drop the rest of the buffer.
	if hasMovement != 0 {
		_, _ = reader.ReadAll()
	}

	return nil
}

func (gc *WorldSession) HandleCastSpell(data wow.PacketData) {
	if gc.player == nil || gc.player.IsGhost {
		return
	}

	// 3.3.5a layout: u8 cast count, u32 spell, u8 cast flags, u32 target mask,
	// then a packed GUID when the mask names a unit (0x2) or object (0x800).
	reader := wow.NewPacketReader(data)

	var castCount uint8
	_ = reader.Read(&castCount)

	castID := uint32(castCount)

	var spellID uint32
	_ = reader.Read(&spellID)

	var castFlags uint8
	_ = reader.Read(&castFlags)

	targetGUID := uint64(gc.player.GUID())

	targets, err := readSpellCastTargets(reader, func(guid wow.GUID) Unit {
		return gc.resolveCombatUnit(guid)
	})
	if err != nil || targets == nil {
		// Fall back to previous behaviour: self target on parse failure.
		targets = &SpellCastTargets{TargetMask: TargetFlagNone}
	}

	if targets.UnitTarget != nil {
		targetGUID = uint64(targets.UnitTarget.GetGUID())
	} else if targetGUID == 0 {
		targetGUID = uint64(gc.player.GUID())
	}

	_ = readClientCastFlags(reader, castFlags)

	// Find the spell info from DBC data
	server, ok := gc.ws.(*Server)
	if !ok || server.spellMgr == nil {
		return
	}

	spellInfo := server.spellMgr.GetSpellInfo(spellID)
	if spellInfo == nil {
		gc.log.Debug().Uint32("spell", spellID).Msg("unknown spell")
		gc.sendCastFailed(spellID, SpellCastFailedUnknownSpell)
		return
	}

	// Check if player knows this spell
	if !gc.player.KnowsSpell(spellID) {
		gc.sendCastFailed(spellID, SpellCastFailedUnknownSpell)
		return
	}

	// Create the spell execution engine
	spell := NewSpell(gc.player, spellInfo, TriggeredNone)
	if spell == nil {
		gc.sendCastFailed(spellID, SpellCastFailedSpellFailed)
		return
	}

	// Set up targets from the parsed SpellCastTargets payload
	var target CombatUnit
	if targets.UnitTarget == nil && !spellInfo.IsSelfCast() && targetGUID != 0 {
		target = gc.resolveCombatUnit(wow.GUID(targetGUID))
		if target == nil {
			gc.sendCastFailed(spellID, SpellCastResult(5)) // Invalid target
			return
		}
		targets.UnitTarget = target
	} else if targets.UnitTarget != nil {
		target, _ = targets.UnitTarget.(CombatUnit)
	}

	// Prepare the spell (validates and starts cast)
	result := spell.Prepare(targets)
	if result != SpellCastSuccess {
		gc.sendCastFailed(spellID, result)
		return
	}

	gc.beginSpellCast(castID, 0, spell, target)

	gc.log.Debug().
		Uint32("spell", spellID).
		Uint32("spellId", spellInfo.Id).
		Msg("spell cast")
}

// ProcessSpellTick processes ongoing spell casts.
func (gc *WorldSession) ProcessSpellTick(now time.Time) {
	if gc.player == nil || gc.player.ActiveSpell == nil {
		return
	}

	activeSpell, ok := gc.player.ActiveSpell.(*ActiveSpell)
	if !ok {
		return
	}

	if activeSpell.Spell == nil {
		gc.player.ActiveSpell = nil
		return
	}

	// Tick the spell engine (50ms per world update)
	const tickMs int32 = 50

	stillCasting := activeSpell.Spell.Update(tickMs)
	if stillCasting {
		return
	}

	// The cast ran out: announce the outcome (SMSG_SPELL_GO when it landed,
	// SMSG_CAST_FAILED when the completion check rejected it), then clear it.
	if activeSpell.Spell.Succeeded {
		gc.sendSpellGo(activeSpell.CastCount, activeSpell.CastItem, activeSpell.Spell.Info, activeSpell.Target)
		gc.applySpellCompletion(activeSpell.Spell)
	} else {
		gc.sendCastFailed(activeSpell.Spell.Info.Id, SpellCastFailedSpellFailed)
	}

	gc.player.ActiveSpell = nil
}

// applySpellCompletion runs the world-level side effects of a landed cast that
// the spell engine only flags, because they need packets or session state.
func (gc *WorldSession) applySpellCompletion(spell *Spell) {
	// TELEPORT_UNITS on an item (the hearthstone) sends the caster home. Spells
	// with an explicit destination would need Teleport.dbc, which is not loaded.
	if spell.HomeTeleport && spell.CastItem != nil {
		gc.TeleportToBindPoint()
	}
}

// broadcastPlayerStatsToAll sends health/mana updates for a player to all nearby players.
func (gc *WorldSession) broadcastPlayerStatsToAll(target *player.Player) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Build update mask with health and power
	mask := &object.UpdateMask{}
	mask.SetCount(uint32(target.Object.ValuesCount()))

	powerType := target.GetPrimaryPowerType()

	// Always update health
	mask.SetBit(uint32(object.UnitFieldHealth))

	// Update primary power
	if powerType >= 0 && int(powerType) < wow.MaxPowerTypes {
		mask.SetBit(uint32(object.UpdateField(int(object.UnitFieldPower1) + int(powerType))))
	}

	// Build values block
	blockCount := mask.GetUpdateBlockCount()

	// Create values update packet
	pkt := wow.NewPacket(wow.ServerUpdateObject)

	_ = pkt.WriteUint32(1) // block count
	_ = pkt.WriteOne(0)    // has transport

	// Update type
	_ = pkt.WriteOne(wow.UpdateTypeValues)
	_ = pkt.Write(target.GUID())

	// Write mask
	for i := uint32(0); i < blockCount; i++ {
		val := uint32(0)
		for b := uint32(0); b < 32; b++ {
			idx := i*32 + b
			if mask.GetBit(idx) {
				val |= 1 << b
			}
		}
		_ = pkt.Write(val)
	}

	// Write values
	for i := uint32(0); i < blockCount*32; i++ {
		if mask.GetBit(i) && int(i) < target.Object.ValuesCount() {
			_ = pkt.Write(target.Object.GetUInt32Value(object.UpdateField(i)))
		}
	}

	// Send to all
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
}

// findPlayerByGUID finds a player session by GUID.
func (gc *WorldSession) findPlayerByGUID(server *Server, guid uint64) *player.Player {
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.GUID() == wow.GUID(guid) {
			return other.player
		}
	}

	return nil
}

// writePackedGUID appends a packed GUID (mask byte then the non-zero bytes).
func writePackedGUID(pkt *wow.Packet, guid wow.GUID) {
	_, _ = pkt.WriteBytes(guid.Pack())
}

// writeSpellCastTargets writes a SpellCastTargets payload (AC
// SpellCastTargets::Write): a u32 mask and, when it names a unit, the target's
// packed GUID.
func writeSpellCastTargets(pkt *wow.Packet, target CombatUnit) {
	if target == nil {
		_ = pkt.Write(TargetFlagNone)
		return
	}

	_ = pkt.Write(TargetFlagUnit)
	writePackedGUID(pkt, target.GetGUID())
}

// beginSpellCast announces a prepared spell. Cast-time spells send
// SMSG_SPELL_START and finish through ProcessSpellTick; instant spells send
// SMSG_SPELL_GO right away.
func (gc *WorldSession) beginSpellCast(castCount uint32, castItem wow.GUID, spell *Spell, target CombatUnit) {
	if spell.State != SpellStateCasting {
		gc.sendSpellGo(castCount, castItem, spell.Info, target)
		gc.applySpellCompletion(spell)
		return
	}

	gc.sendSpellStart(castCount, castItem, spell.Info, uint32(spell.CastTimeLeft), target)

	targetGUID := uint64(gc.player.GUID())
	if target != nil {
		targetGUID = uint64(target.GetGUID())
	}
	now := time.Now()
	castTime := time.Duration(spell.CastTimeLeft) * time.Millisecond
	gc.player.ActiveSpell = &ActiveSpell{
		Spell:       spell,
		Caster:      gc.player,
		Target:      target,
		TargetGUID:  targetGUID,
		State:       spell.State,
		StartTime:   now,
		CastTime:    castTime,
		CastEndTime: now.Add(castTime),
		CastCount:   castCount,
		CastItem:    castItem,
	}
}

// sendSpellStart sends SMSG_SPELL_START (3.3.5a): packed castItem, packed
// caster, u8 castCount, u32 spell, u32 flags, u32 timer, SpellCastTargets.
// `castItem` is zero for a plain spell cast; the caster GUID is used instead.
func (gc *WorldSession) sendSpellStart(castCount uint32, castItem wow.GUID, spellInfo *SpellInfo, castTimeMs uint32, target CombatUnit) {
	pkt := wow.NewPacket(wow.ServerSpellStart)

	caster := gc.player.GUID()
	if castItem == 0 {
		castItem = caster
	}
	writePackedGUID(pkt, castItem)
	writePackedGUID(pkt, caster)

	_ = pkt.WriteOne(int(castCount & 0xff))
	_ = pkt.Write(spellInfo.Id)
	_ = pkt.Write(uint32(0)) // CastFlags
	_ = pkt.Write(castTimeMs)
	writeSpellCastTargets(pkt, target)

	gc.socket.Send(pkt)
}

// sendCastFailed sends SMSG_CAST_FAILED (3.3.5a): u8 castCount, u32 spell,
// u8 result, u8 multipleCasts.
func (gc *WorldSession) sendCastFailed(spellID uint32, result SpellCastResult) {
	pkt := wow.NewPacket(wow.ServerCastFailed)

	_ = pkt.WriteOne(0)           // cast count
	_ = pkt.Write(spellID)        // spell
	_ = pkt.WriteOne(int(result)) // result
	_ = pkt.WriteOne(0)           // multiple casts (false)

	gc.socket.Send(pkt)
}

// sendSpellGo sends SMSG_SPELL_GO (3.3.5a): packed castItem, packed caster,
// u8 extraCasts, u32 spell, u32 flags, u32 timestamp, the hit and miss lists,
// then SpellCastTargets.
func (gc *WorldSession) sendSpellGo(castCount uint32, castItem wow.GUID, spellInfo *SpellInfo, target CombatUnit) {
	pkt := wow.NewPacket(wow.ServerSpellGo)

	caster := gc.player.GUID()
	if castItem == 0 {
		castItem = caster
	}
	writePackedGUID(pkt, castItem)
	writePackedGUID(pkt, caster)

	_ = pkt.WriteOne(int(castCount & 0xff)) // extra casts
	_ = pkt.Write(spellInfo.Id)
	_ = pkt.Write(uint32(0)) // cast flags
	_ = pkt.Write(uint32(0)) // timestamp

	// Hits: the resolved target, drawn as a raw GUID.
	if target != nil {
		_ = pkt.WriteOne(1)
		_ = pkt.Write(target.GetGUID())
	} else {
		_ = pkt.WriteOne(0)
	}

	_ = pkt.WriteOne(0) // misses

	writeSpellCastTargets(pkt, target)

	gc.socket.Send(pkt)
}

// sendSpellLog sends SMSG_SPELLLOGEXECUTE with damage/heal information.
func (gc *WorldSession) sendSpellLog(spellInfo *SpellInfo, target CombatUnit, damage, heal uint32) {
	pkt := wow.NewPacket(wow.ServerSpelllogexecute)

	_ = pkt.Write(spellInfo.Id)
	if target != nil {
		_ = pkt.Write(target.GetGUID())
	} else {
		_ = pkt.Write(uint64(0))
	}
	_ = pkt.Write(uint32(0)) // spell log flags
	_ = pkt.Write(uint32(0)) // amount
	_ = pkt.Write(uint32(0)) // overkill
	_ = pkt.Write(uint32(0)) // school
	_ = pkt.Write(uint32(0)) // absorbed
	_ = pkt.Write(uint32(0)) // resisted

	gc.socket.Send(pkt)
}

// sendAuraUpdate sends SMSG_AURA_UPDATE for an aura change.
func (gc *WorldSession) sendAuraUpdate(target CombatUnit, aura *Aura) {
	pkt := wow.NewPacket(wow.ServerAuraUpdate)

	if target != nil {
		_ = pkt.Write(target.GetGUID())
	} else {
		_ = pkt.Write(uint64(0))
	}
	_ = pkt.WriteOne(0) // removed (0 = add/update)

	// Aura slot (0 = first slot)
	_ = pkt.WriteOne(0)

	// Spell ID
	var spellID uint32
	if aura.SpellInfo != nil {
		spellID = aura.SpellInfo.Id
	}
	_ = pkt.Write(spellID)

	// Stack count
	_ = pkt.Write(aura.StackAmount)

	// Duration (ms)
	_ = pkt.Write(aura.Duration)

	// Max duration (ms)
	_ = pkt.Write(aura.MaxDuration)

	// Flags
	_ = pkt.Write(uint32(0))

	// Level
	_ = pkt.Write(uint8(1))

	// Item stack count
	_ = pkt.WriteOne(0)

	gc.socket.Send(pkt)
}
