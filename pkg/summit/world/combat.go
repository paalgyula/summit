package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Attack states.
const (
	AttackStateIdle    = 0
	AttackStateSwinging = 1
)

// Swing timer in milliseconds.
const SwingTimerMS = 2000

// HandleAttackSwing starts auto-attack on a target.
func (gc *WorldSession) HandleAttackSwing(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var targetGUID uint64
	_ = reader.Read(&targetGUID)

	gc.player.AttackTarget = targetGUID
	gc.player.AttackState = AttackStateSwinging
	gc.player.NextAttackTime = time.Now().UnixMilli() + SwingTimerMS

	gc.log.Debug().
		Uint64("target", targetGUID).
		Msg("attack swing started")
}

// HandleAttackStop stops auto-attack.
func (gc *WorldSession) HandleAttackStop(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var targetGUID uint64
	_ = reader.Read(&targetGUID)

	gc.player.AttackTarget = 0
	gc.player.AttackState = AttackStateIdle
	gc.player.NextAttackTime = 0

	// Send attack stop to self
	gc.sendAttackStop(gc.player.GUID(), wow.GUID(targetGUID))

	gc.log.Debug().Msg("attack stopped")
}

// ProcessCombatTick handles auto-attack swings.
func (gc *WorldSession) ProcessCombatTick(now time.Time) {
	if gc.player == nil || gc.player.AttackState != AttackStateSwinging {
		return
	}

	if now.UnixMilli() < gc.player.NextAttackTime {
		return
	}

	// Find target
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Try player target first
	target := gc.findAttackTarget(server)
	if target != nil {
		gc.processPlayerAttack(target, now)

		return
	}

	// Try NPC target
	npcTarget := gc.findAttackTargetNPC(server)
	if npcTarget != nil {
		gc.processNPCAttack(npcTarget, now)

		return
	}

	// No target found, stop attacking
	gc.player.AttackState = AttackStateIdle
	gc.player.AttackTarget = 0
}

// processPlayerAttack handles attacking another player.
func (gc *WorldSession) processPlayerAttack(target *player.Player, now time.Time) {
	damage := gc.player.BaseDamage

	newHealth := target.GetHealth()
	if damage > float32(newHealth) {
		damage = float32(newHealth)
	}

	newHealth -= uint32(damage)
	target.SetHealth(newHealth)

	// Grant rage for warriors/feral druids dealing damage
	gc.grantRageOnDamageDealt(uint32(damage))

	gc.sendAttackerStateUpdate(gc.player, target, uint32(damage))
	gc.broadcastHealthUpdate(target)

	if target.IsDead() {
		gc.onTargetDied(nil)
	}

	gc.player.NextAttackTime = now.UnixMilli() + SwingTimerMS
}

// processNPCAttack handles attacking an NPC.
func (gc *WorldSession) processNPCAttack(npc *NPC, now time.Time) {
	damage := gc.player.BaseDamage

	newHealth := npc.GetHealth()
	if damage > float32(newHealth) {
		damage = float32(newHealth)
	}

	newHealth -= uint32(damage)
	npc.SetHealth(newHealth)

	// Grant rage for warriors/feral druids dealing damage
	gc.grantRageOnDamageDealt(uint32(damage))

	// Send health update for NPC
	gc.broadcastNPCHealthUpdate(npc)

	if npc.IsDead() {
		gc.onNPCDied(npc)
	}

	gc.player.NextAttackTime = now.UnixMilli() + SwingTimerMS
}

// findAttackTarget finds the player's current attack target (player or NPC).
func (gc *WorldSession) findAttackTarget(server *Server) *player.Player {
	// Check if target is a player
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.GUID() == wow.GUID(gc.player.AttackTarget) {
			return other.player
		}
	}

	return nil
}

// findAttackTargetNPC finds the player's current attack target NPC.
func (gc *WorldSession) findAttackTargetNPC(server *Server) *NPC {
	return server.spawns.GetNPC(uint32(gc.player.AttackTarget))
}

// sendAttackerStateUpdate sends SMSG_ATTACKERSTATEUPDATE.
func (gc *WorldSession) sendAttackerStateUpdate(attacker, target *player.Player, damage uint32) {
	pkt := wow.NewPacket(wow.ServerAttackerstateupdate)

	// Hit info
	_ = pkt.Write(uint32(1)) // hit type (1 = normal hit)

	// Attacker GUID
	_ = pkt.Write(attacker.GUID())

	// Target GUID
	_ = pkt.Write(target.GUID())

	// Damage
	_ = pkt.Write(uint32(damage))

	// Overkill (0 for now)
	_ = pkt.Write(uint32(0))

	// School (0 = physical)
	_ = pkt.Write(uint32(0))

	// Absorbed damage (0)
	_ = pkt.Write(uint32(0))

	// Resisted damage (0)
	_ = pkt.Write(uint32(0))

	// Victim state (-1 = alive, 0 = just died)
	victimState := int32(-1)
	if target.IsDead() {
		victimState = 0
	}
	_ = pkt.Write(victimState)

	// Unk int (0)
	_ = pkt.Write(uint32(0))

	// Melee swing timer (remaining time in ms)
	swingTimer := uint32(SwingTimerMS)
	_ = pkt.Write(swingTimer)

	// Spell ID (0 for melee)
	_ = pkt.Write(uint32(0))

	// Hit info flags
	_ = pkt.Write(uint32(0))

	gc.socket.Send(pkt)
}

// sendAttackStop sends SMSG_ATTACKSTOP.
func (gc *WorldSession) sendAttackStop(attacker, target wow.GUID) {
	pkt := wow.NewPacket(wow.ServerAttackstop)

	_ = pkt.Write(attacker)
	_ = pkt.Write(target)
	_ = pkt.Write(uint32(0)) // extra attacks

	gc.socket.Send(pkt)
}

// broadcastHealthUpdate sends health update for a target to all nearby players.
func (gc *WorldSession) broadcastHealthUpdate(target *player.Player) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Build a values update with just the health field
	upd := &Updater{}
	upd.updateFlags = uint8(wow.UpdateFlagLowGUID | wow.UpdateFlagHighGUID | wow.UpdateFlagLiving | wow.UpdateFlagHasPosition)

	// Create update mask with only health
	mask := &object.UpdateMask{}
	mask.SetCount(uint32(target.Object.ValuesCount()))
	mask.SetBit(uint32(object.UnitFieldHealth))

	// Build the values block
	pkt := wow.NewPacket(wow.ServerUpdateObject)

	_ = pkt.WriteUint32(1) // block count
	_ = pkt.WriteOne(0)    // has transport

	// Update type
	_ = pkt.WriteOne(wow.UpdateTypeValues)
	_ = pkt.Write(target.GUID())

	// Write mask
	blockCount := mask.GetUpdateBlockCount()
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

	// Write health value
	_ = pkt.Write(target.GetHealth())

	// Send to all
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
}

// onTargetDied handles when a player target dies.
func (gc *WorldSession) onTargetDied(target *player.Player) {
	// Stop attacking
	gc.player.AttackState = AttackStateIdle
	gc.player.AttackTarget = 0

	// Kill the target player
	target.Die()

	// Send death packet to the dead player
	server, ok := gc.ws.(*Server)
	if ok {
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && other.player.IsInWorld {
				other.broadcastPlayerStats()
			}
		}
	}

	gc.log.Debug().Str("target", target.Name).Msg("player killed")
}

// onNPCDied handles when an NPC dies.
func (gc *WorldSession) onNPCDied(npc *NPC) {
	// Stop attacking
	gc.player.AttackState = AttackStateIdle
	gc.player.AttackTarget = 0

	// Grant XP for the kill
	levelsGained := gc.player.Kill(npc.Level)

	// Send XP/level up packets
	if levelsGained > 0 {
		gc.sendLevelUpInfo(levelsGained)
	}

	// Broadcast updated stats (level, health, etc)
	gc.broadcastPlayerStats()

	// Send destroy object to all players
	server, ok := gc.ws.(*Server)
	if ok {
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && other.player.IsInWorld {
				other.sendDestroyObject(npc.GetGUID())
			}
		}
	}

	gc.log.Debug().
		Str("npc", npc.Name).
		Uint32("xp", gc.player.XP).
		Uint8("level", gc.player.Level).
		Msg("NPC killed")
}

// broadcastNPCHealthUpdate sends health update for an NPC to all nearby players.
func (gc *WorldSession) broadcastNPCHealthUpdate(npc *NPC) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}

	// Build a values update with just the health field
	mask := &object.UpdateMask{}
	mask.SetCount(uint32(npc.Object.ValuesCount()))
	mask.SetBit(uint32(object.UnitFieldHealth))

	// Create update packet
	pkt := wow.NewPacket(wow.ServerUpdateObject)

	_ = pkt.WriteUint32(1) // block count
	_ = pkt.WriteOne(0)    // has transport

	// Update type
	_ = pkt.WriteOne(wow.UpdateTypeValues)
	_ = pkt.Write(npc.GetGUID())

	// Write mask
	blockCount := mask.GetUpdateBlockCount()
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

	// Write health value
	_ = pkt.Write(npc.GetHealth())

	// Send to all
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
}

// grantRageOnDamageDealt grants rage when a warrior/feral druid deals damage.
// In WoW, rage is gained from dealing damage (5 rage per melee hit, more for crits).
func (gc *WorldSession) grantRageOnDamageDealt(damage uint32) {
	// Only grant rage for rage-using classes (warrior, feral druid)
	powerType := gc.player.GetPrimaryPowerType()
	if powerType != wow.PowerTypeRage {
		return
	}

	// Rage gain: 1 rage per 10 damage dealt, minimum 5 rage per hit
	rageGain := damage / 10
	if rageGain < 5 {
		rageGain = 5
	}

	// Cap rage at 100
	maxRage := uint32(100)
	newRage := gc.player.GetPower(powerType) + rageGain
	if newRage > maxRage {
		newRage = maxRage
	}

	gc.player.SetPower(powerType, newRage)
}

// grantRageOnDamageTaken grants rage when a warrior takes damage.
// In WoW, warriors gain rage from damage taken.
func (gc *WorldSession) grantRageOnDamageTaken(damage uint32) {
	// Only grant rage for rage-using classes
	powerType := gc.player.GetPrimaryPowerType()
	if powerType != wow.PowerTypeRage {
		return
	}

	// Rage gain from damage taken: 1 rage per 60 damage taken
	rageGain := damage / 60
	if rageGain < 1 && damage > 0 {
		rageGain = 1
	}

	// Cap rage at 100
	maxRage := uint32(100)
	newRage := gc.player.GetPower(powerType) + rageGain
	if newRage > maxRage {
		newRage = maxRage
	}

	gc.player.SetPower(powerType, newRage)
}
