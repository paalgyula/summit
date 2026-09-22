package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleCastSpell handles CMSG_CAST_SPELL from the client.
// SpellCastTargets masks of CMSG_CAST_SPELL.
const (
	TargetFlagUnit   = 0x2
	TargetFlagObject = 0x800
)

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

	var targetMask uint32
	_ = reader.Read(&targetMask)

	targetGUID := uint64(gc.player.GUID())

	if targetMask&(TargetFlagUnit|TargetFlagObject) != 0 {
		if guid, err := wow.ReadPackedGUID(reader); err == nil {
			targetGUID = uint64(guid)
		}
	}

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

	// Set up targets
	targets := SpellCastTargets{}

	// Find target unit if needed
	var target CombatUnit
	if !spellInfo.IsSelfCast() && targetGUID != 0 {
		target = gc.resolveCombatUnit(wow.GUID(targetGUID))
		if target == nil {
			gc.sendCastFailed(spellID, SpellCastResult(5)) // Invalid target
			return
		}
		targets.UnitTarget = target
	}

	// Prepare the spell (validates and starts cast)
	result := spell.Prepare(&targets)
	if result != SpellCastSuccess {
		gc.sendCastFailed(spellID, result)
		return
	}

	// Send spell go packet
	gc.sendSpellGo(castID, spellInfo, target)

	// Store active spell on player for non-instant casts
	if spell.State == SpellStateCasting {
		gc.player.ActiveSpell = &ActiveSpell{
			Spell:      spell,
			Caster:     gc.player,
			Target:     target,
			TargetGUID: targetGUID,
			State:      spell.State,
			StartTime:  time.Now(),
			CastTime:   time.Duration(spell.CastTimeLeft) * time.Millisecond,
		}
		gc.player.ActiveSpell.(*ActiveSpell).CastEndTime = time.Now().Add(time.Duration(spell.CastTimeLeft) * time.Millisecond)
	}

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
	if !stillCasting {
		gc.player.ActiveSpell = nil
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

// sendCastFailed sends SMSG_CAST_FAILED when a spell cast fails.
func (gc *WorldSession) sendCastFailed(spellID uint32, result SpellCastResult) {
	pkt := wow.NewPacket(wow.ServerCastFailed)

	_ = pkt.Write(uint32(0)) // cast ID
	_ = pkt.Write(spellID)
	_ = pkt.Write(uint32(result)) // failed reason

	gc.socket.Send(pkt)
}

// sendSpellGo sends SMSG_SPELL_GO when a spell is cast.
func (gc *WorldSession) sendSpellGo(castID uint32, spellInfo *SpellInfo, target CombatUnit) {
	pkt := wow.NewPacket(wow.ServerSpellGo)

	// Spell ID
	_ = pkt.Write(spellInfo.Id)

	// Cast ID
	_ = pkt.Write(castID)

	// Optional cast count
	_ = pkt.WriteOne(0)

	// Flags
	_ = pkt.Write(uint32(0))

	// Cast time
	_ = pkt.Write(uint32(0))

	// Hit target count
	if target != nil {
		_ = pkt.WriteOne(1)
		_ = pkt.Write(target.GetGUID())
	} else {
		_ = pkt.WriteOne(0)
	}

	// Miss target count
	_ = pkt.WriteOne(0)

	// Target flags
	if target != nil {
		_ = pkt.Write(uint32(1)) // TARGET_FLAG_UNIT
		_ = pkt.Write(target.GetGUID())
	}

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
