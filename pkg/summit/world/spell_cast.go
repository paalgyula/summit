package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleCastSpell handles CMSG_CAST_SPELL from the client.
func (gc *WorldSession) HandleCastSpell(data wow.PacketData) {
	if gc.player == nil || gc.player.IsGhost {
		return
	}

	reader := wow.NewPacketReader(data)

	var castID uint32
	_ = reader.Read(&castID)

	var spellID uint32
	_ = reader.Read(&spellID)

	var targetGUID uint64
	_ = reader.Read(&targetGUID)

	// Find the spell template
	server, ok := gc.ws.(*Server)
	if !ok || server.spellManager == nil {
		return
	}

	spell := server.spellManager.GetSpell(spellID)
	if spell == nil {
		gc.log.Debug().Uint32("spell", spellID).Msg("unknown spell")
		gc.sendCastFailed(spellID, SpellCastResult(6)) // Unknown spell
		return
	}

	// Check if player knows this spell
	if !gc.player.KnowsSpell(spellID) {
		gc.sendCastFailed(spellID, SpellCastResult(6))
		return
	}

	// Check power cost
	if !gc.player.HasPowerForSpell(int(spell.PowerType), spell.PowerCost) {
		gc.sendCastFailed(spellID, SpellCastNoMana)
		return
	}

	// Check if spell is on cooldown
	if gc.player.IsSpellOnCooldown(spellID) {
		gc.sendCastFailed(spellID, SpellCastResult(14)) // Not ready yet
		return
	}

	// Check range
	var target *player.Player
	if spell.TargetType != SpellTargetSelf && targetGUID != 0 {
		target = gc.findPlayerByGUID(server, targetGUID)
		if target == nil {
			gc.sendCastFailed(spellID, SpellCastResult(5)) // Invalid target
			return
		}

		if !gc.player.IsWithinRange(target, spell.Range) {
			gc.sendCastFailed(spellID, SpellCastTargetTooFar)
			return
		}
	}

	// Send cast start to client
	gc.sendSpellGo(castID, spell, target)

	// Consume power
	gc.player.ConsumePowerForSpell(int(spell.PowerType), spell.PowerCost)

	// Apply instant effects
	if spell.CastTime == 0 {
		gc.applySpellEffects(spell, target)
		gc.sendSpellCastComplete(castID, spell)
	} else {
		// Start cast timer for non-instant spells
		gc.startCasting(spell, castID, target, targetGUID)
	}

	gc.log.Debug().
		Uint32("spell", spellID).
		Str("name", spell.Name).
		Msg("spell cast")
}

// startCasting begins the cast time for a spell.
func (gc *WorldSession) startCasting(spell *SpellTemplate, castID uint32, target *player.Player, targetGUID uint64) {
	// Create active spell
	activeSpell := &ActiveSpell{
		Spell:      spell,
		Caster:     gc.player,
		Target:     target,
		TargetGUID: targetGUID,
		State:      SpellStateCasting,
		StartTime:  time.Now(),
		CastEndTime: time.Now().Add(spell.CastTime),
		CastTime:   spell.CastTime,
	}

	// Store active spell on player
	gc.player.ActiveSpell = activeSpell
}

// ProcessSpellTick processes ongoing spell casts.
func (gc *WorldSession) ProcessSpellTick(now time.Time) {
	if gc.player == nil || gc.player.ActiveSpell == nil {
		return
	}

	// Type assert to *ActiveSpell
	activeSpell, ok := gc.player.ActiveSpell.(*ActiveSpell)
	if !ok {
		return
	}

	switch activeSpell.State {
	case SpellStateCasting:
		if now.After(activeSpell.CastEndTime) {
			// Cast complete - apply effects
			gc.applySpellEffects(activeSpell.Spell, activeSpell.Target)
			gc.sendSpellCastComplete(0, activeSpell.Spell)
			gc.player.ActiveSpell = nil
		}
	}
}

// applySpellEffects applies all effects of a spell.
func (gc *WorldSession) applySpellEffects(spell *SpellTemplate, target *player.Player) {
	for _, effect := range spell.Effects {
		if effect.Type == SpellEffectNone {
			continue
		}

		switch effect.Type {
		case SpellEffectSchoolDamage:
			gc.applyDamageEffect(spell, effect, target)
		case SpellEffectHeal:
			gc.applyHealEffect(spell, effect, target)
		case SpellEffectApplyAura:
			gc.applyAuraEffect(spell, effect, target)
		case SpellEffectEnergize:
			gc.applyEnergizeEffect(spell, effect, target)
		}
	}
}

// applyDamageEffect applies a damage spell effect.
func (gc *WorldSession) applyDamageEffect(spell *SpellTemplate, effect SpellEffectData, target *player.Player) {
	if target == nil {
		return
	}

	damage := uint32(effect.BasePoints)
	if damage == 0 {
		damage = 1
	}

	// Apply damage
	newHealth := target.GetHealth()
	if damage > newHealth {
		damage = newHealth
	}

	newHealth -= damage
	target.SetHealth(newHealth)

	// Send damage update
	gc.broadcastPlayerStatsToAll(target)

	// Send spell log
	gc.sendSpellLog(spell, target, damage, 0)

	// Check if target died
	if target.IsDead() {
		target.Die()
	}
}

// applyHealEffect applies a heal spell effect.
func (gc *WorldSession) applyHealEffect(spell *SpellTemplate, effect SpellEffectData, target *player.Player) {
	if target == nil {
		target = gc.player // heal self if no target
	}

	heal := uint32(effect.BasePoints)
	if heal == 0 {
		heal = 1
	}

	// Apply heal
	newHealth := target.GetHealth() + heal
	if newHealth > target.GetMaxHealth() {
		newHealth = target.GetMaxHealth()
	}

	target.SetHealth(newHealth)

	// Send heal update
	gc.broadcastPlayerStatsToAll(target)

	// Send spell log
	gc.sendSpellLog(spell, target, 0, heal)
}

// applyAuraEffect applies an aura (buff/debuff) effect.
func (gc *WorldSession) applyAuraEffect(spell *SpellTemplate, effect SpellEffectData, target *player.Player) {
	if target == nil {
		target = gc.player
	}

	// Create aura
	aura := &Aura{
		SpellID:    spell.ID,
		Spell:      spell,
		CasterGUID: uint64(gc.player.GUID()),
		Duration:   spell.Duration,
		Remaining:  spell.Duration,
		StackCount: 1,
		StartTime:  time.Now(),
		Effects: []AuraEffect{
			{
				Type:   effect.TriggerAura,
				Value:  effect.BasePoints,
				Period: effect.ApplyAuraPeriod,
				LastTick: time.Now(),
			},
		},
	}

	// Add aura to target (append to auras slice)
	target.Auras = append(target.Auras, aura)

	// Send aura update
	gc.sendAuraUpdate(target, aura)
}

// applyEnergizeEffect applies a power restoration effect.
func (gc *WorldSession) applyEnergizeEffect(spell *SpellTemplate, effect SpellEffectData, target *player.Player) {
	if target == nil {
		target = gc.player
	}

	amount := uint32(effect.BasePoints)
	powerType := spell.PowerType

	// Restore power
	currentPower := target.GetPower(wow.PowerType(powerType))
	maxPower := target.GetMaxPower(wow.PowerType(powerType))

	newPower := currentPower + amount
	if newPower > maxPower {
		newPower = maxPower
	}

	target.SetPower(wow.PowerType(powerType), newPower)

	// Send power update
	gc.broadcastPlayerStatsToAll(target)
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
		mask.SetBit(uint32(object.UpdateField(int(object.UnitFieldPower1)+int(powerType))))
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
func (gc *WorldSession) sendSpellGo(castID uint32, spell *SpellTemplate, target *player.Player) {
	pkt := wow.NewPacket(wow.ServerSpellGo)

	// Spell ID
	_ = pkt.Write(spell.ID)

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
		_ = pkt.Write(target.GUID())
	} else {
		_ = pkt.WriteOne(0)
	}

	// Miss target count
	_ = pkt.WriteOne(0)

	// Target flags
	if target != nil {
		_ = pkt.Write(uint32(1)) // TARGET_FLAG_UNIT
		_ = pkt.Write(target.GUID())
	}

	gc.socket.Send(pkt)
}

// sendSpellCastComplete sends SMSG_SPELL_GO when cast completes.
func (gc *WorldSession) sendSpellCastComplete(castID uint32, spell *SpellTemplate) {
	// For instant spells, this was already sent
	// For channeled spells, this would be sent when channel ends
}

// sendSpellLog sends SMSG_SPELLLOGEXECUTE with damage/heal information.
func (gc *WorldSession) sendSpellLog(spell *SpellTemplate, target *player.Player, damage, heal uint32) {
	pkt := wow.NewPacket(wow.ServerSpelllogexecute)

	_ = pkt.Write(spell.ID)
	_ = pkt.Write(target.GUID())
	_ = pkt.Write(uint32(0)) // spell log flags
	_ = pkt.Write(uint32(0)) // amount
	_ = pkt.Write(uint32(0)) // overkill
	_ = pkt.Write(uint32(0)) // school
	_ = pkt.Write(uint32(0)) // absorbed
	_ = pkt.Write(uint32(0)) // resisted

	gc.socket.Send(pkt)
}

// sendAuraUpdate sends SMSG_AURA_UPDATE for an aura change.
func (gc *WorldSession) sendAuraUpdate(target *player.Player, aura *Aura) {
	pkt := wow.NewPacket(wow.ServerAuraUpdate)

	_ = pkt.Write(target.GUID())
	_ = pkt.WriteOne(0) // removed (0 = add/update)

	// Aura slot (0 = first slot)
	_ = pkt.WriteOne(0)

	// Spell ID
	_ = pkt.Write(aura.SpellID)

	// Stack count
	_ = pkt.Write(aura.StackCount)

	// Duration
	_ = pkt.Write(int32(aura.Duration / time.Millisecond))

	// Max duration
	_ = pkt.Write(int32(aura.Spell.Duration / time.Millisecond))

	// Flags
	_ = pkt.Write(uint32(0))

	// Level
	_ = pkt.Write(uint8(1))

	// Item stack count
	_ = pkt.WriteOne(0)

	gc.socket.Send(pkt)
}
