package world

import (
	"math"
	"math/rand"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/loot"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Attack states.
const (
	AttackStateIdle     = 0
	AttackStateSwinging = 1
)

// Unit dynamic flags (UNIT_DYNAMIC_FLAGS).
// Source: AzerothCore SharedDefines.h:3351-3362.
const (
	UnitDynFlagNone               uint32 = 0x0000
	UnitDynFlagLootable           uint32 = 0x0001
	UnitDynFlagTrackUnit          uint32 = 0x0002
	UnitDynFlagTapped             uint32 = 0x0004
	UnitDynFlagTappedByPlayer     uint32 = 0x0008
	UnitDynFlagSpecialInfo        uint32 = 0x0010
	UnitDynFlagDead               uint32 = 0x0020
	UnitDynFlagReferAFriend       uint32 = 0x0040
	UnitDynFlagTappedByAllThreat  uint32 = 0x0080
)

// Swing timer in milliseconds.
const SwingTimerMS = 2000

// Base miss chance (5%) at equal weapon skill.
const baseMissChance = 5.0

// Base glancing chance for attackers below victim level.
const baseGlancingChance = 10.0

// Base crushing chance multiplier.
const baseCrushingChance = 15.0

// HandleAttackSwing starts auto-attack on a target.
// Maps to AC's WorldSession::HandleAttackSwingOpcode (CombatHandler.cpp:28).
func (gc *WorldSession) HandleAttackSwing(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	reader := wow.NewPacketReader(data)

	var targetGUID uint64
	_ = reader.Read(&targetGUID)

	gc.log.Debug().
		Uint64("target", targetGUID).
		Msg("CMSG_ATTACKSWING received")

	// Resolve target GUID to CombatUnit
	target := gc.resolveCombatUnit(wow.GUID(targetGUID))
	if target == nil {
		// Target not found — stop attack state at client
		gc.sendAttackStop(gc.player.GUID(), wow.GUID(targetGUID))
		return
	}

	// Validate attack target
	if !gc.isValidAttackTarget(target) {
		gc.sendAttackStop(gc.player.GUID(), wow.GUID(targetGUID))
		return
	}

	// Call Attack() to setup attack state
	gc.playerAttack(target, true)
}

// HandleAttackStop stops auto-attack.
// Maps to AC's WorldSession::HandleAttackStopOpcode (CombatHandler.cpp:68).
func (gc *WorldSession) HandleAttackStop(data wow.PacketData) {
	if gc.player == nil {
		return
	}

	gc.playerAttackStop()
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
	target := gc.resolveCombatUnit(wow.GUID(gc.player.AttackTarget))
	if target == nil || !target.IsAlive() {
		gc.playerAttackStop()
		return
	}

	// Execute melee swing
	gc.processCombatAttack(target, now)
}

// processCombatAttack handles a single melee swing against any CombatUnit.
func (gc *WorldSession) processCombatAttack(target CombatUnit, now time.Time) {
	if target == nil || !target.IsAlive() {
		gc.playerAttackStop()
		return
	}

	// Calculate melee damage
	damageInfo := CalculateMeleeDamage(gc.player, target, BaseAttack)

	// Apply damage modifiers
	for i := range damageInfo.Damages {
		DealDamageMods(target, &damageInfo.Damages[i].Damage, &damageInfo.Damages[i].Absorb)
	}

	// Send attack state update to client
	gc.sendAttackerStateUpdateFromCalc(damageInfo)

	// Deal the damage
	DealDamage(gc.player, target, damageInfo)

	// Schedule next swing
	gc.scheduleNextSwing(now)

	gc.log.Debug().
		Str("target", targetName(target)).
		Uint8("outcome", uint8(damageInfo.HitOutcome)).
		Uint32("damage", damageInfo.Damages[0].Damage).
		Msg("melee swing")
}

// scheduleNextSwing schedules the next auto-attack swing.
func (gc *WorldSession) scheduleNextSwing(now time.Time) {
	if gc.player == nil || gc.player.AttackState != AttackStateSwinging {
		return
	}

	swingDelay := time.Duration(SwingTimerMS) * time.Millisecond
	if gc.player.BaseAttackSpeed > 0 {
		swingDelay = gc.player.BaseAttackSpeed
	}

	gc.player.NextAttackTime = now.UnixMilli() + swingDelay.Milliseconds()
}

// playerAttack initiates combat (session-level wrapper).
func (gc *WorldSession) playerAttack(victim CombatUnit, meleeAttack bool) {
	if victim == nil || victim.GetGUID() == gc.player.GUID() {
		return
	}

	// Dead units can neither attack nor be attacked
	if !gc.player.IsAlive() || !victim.IsAlive() {
		return
	}

	// Check map
	if !gc.player.IsInSameMap(victim) {
		return
	}

	// Player cannot attack while mounted
	if gc.player.Mounted {
		return
	}

	// If already attacking same target, toggle melee state
	if gc.player.AttackTarget == uint64(victim.GetGUID()) {
		if meleeAttack {
			if gc.player.AttackState != AttackStateSwinging {
				gc.player.AttackState = AttackStateSwinging
				gc.player.NextAttackTime = time.Now().UnixMilli() + SwingTimerMS
				gc.sendAttackStart(gc.player.GUID(), victim.GetGUID())
			}
		} else {
			if gc.player.AttackState == AttackStateSwinging {
				gc.player.AttackState = AttackStateIdle
				gc.sendAttackStop(gc.player.GUID(), victim.GetGUID())
			}
		}
		return
	}

	// Switch target — stop old attack
	if gc.player.AttackTarget != 0 {
		gc.playerAttackStop()
	}

	// Setup new attack
	gc.player.AttackTarget = uint64(victim.GetGUID())
	gc.player.Victim = victim
	if meleeAttack {
		gc.player.AttackState = AttackStateSwinging
		gc.player.NextAttackTime = time.Now().UnixMilli() + SwingTimerMS
	}

	// Add this player as attacker on victim
	victim.AddAttacker(gc.player.GUID())

	// Enter combat state
	gc.player.SetInCombat()
	victim.SetInCombat()

	// Send SMSG_ATTACKSTART to all nearby
	gc.sendAttackStart(gc.player.GUID(), victim.GetGUID())

	// If victim is a creature, trigger AI aggro
	if npc, ok := victim.(*NPC); ok {
		npc.OnEnterCombat(gc.player)
	}
}

// playerAttackStop stops the current attack (session-level wrapper).
func (gc *WorldSession) playerAttackStop() {
	if gc.player.AttackTarget == 0 {
		return
	}

	targetGUID := gc.player.AttackTarget
	gc.player.AttackTarget = 0
	gc.player.AttackState = AttackStateIdle
	gc.player.NextAttackTime = 0
	gc.player.Victim = nil

	// Remove from victim's attacker set
	target := gc.resolveCombatUnit(wow.GUID(targetGUID))
	if target != nil {
		target.RemoveAttacker(gc.player.GUID())
	}

	// Send SMSG_ATTACKSTOP
	gc.sendAttackStop(gc.player.GUID(), wow.GUID(targetGUID))

	// Nobody left fighting us: leave combat (AC keeps a 5 s PvP timer; skipped)
	if len(gc.player.GetAttackers()) == 0 {
		gc.player.ClearInCombat()
	}
}

// resolveCombatUnit resolves a GUID to a CombatUnit.
func (gc *WorldSession) resolveCombatUnit(guid wow.GUID) CombatUnit {
	if gc.player != nil && gc.player.GUID() == guid {
		return gc.player
	}

	// Check online players
	for _, other := range gc.ws.(*Server).GetOnlineSessions() {
		if other.player != nil && other.player.GUID() == guid {
			return other.player
		}
	}

	// Check NPCs (guid.Counter() matches the spawn ID)
	server, ok := gc.ws.(*Server)
	if ok {
		if npc := server.spawns.GetNPC(uint64(guid.Counter())); npc != nil {
			return npc
		}
		if npc := server.spawns.GetNPC(uint64(guid)); npc != nil {
			return npc
		}
	}

	return nil
}

// isValidAttackTarget checks if the player can attack the target.
func (gc *WorldSession) isValidAttackTarget(target CombatUnit) bool {
	if gc.player == nil || target == nil {
		return false
	}
	if target.GetGUID() == gc.player.GUID() || !target.IsAlive() || gc.player.Mounted {
		return false
	}

	// Duel exception
	if gc.player.Duel != nil {
		if di, ok := gc.player.Duel.(*player.DuelInfo); ok && di.Opponent != nil {
			if di.Opponent.GUID() == target.GetGUID() && di.State == player.DuelStateInProgress {
				return true
			}
		}
	}

	// Check NPCs
	if npc, ok := target.(*NPC); ok {
		// Non-attackable flags (Non-attackable, Not-attackable, Not-selectable)
		const nonAttackableMask = 0x02 | 0x04 | 0x80
		if npc.UnitFlags&nonAttackableMask != 0 {
			return false
		}

		// If faction manager is loaded, check reaction
		fm := GetFactionManager()
		if fm.IsLoaded() {
			return !fm.IsFriendlyTo(gc.player.FactionID, npc.Faction)
		}

		// Mobs / beasts are attackable; NPCs with service flags (vendor, trainer, etc.) are friendly
		return npc.NpcFlags == 0
	}

	// Player PvP
	if otherPlayer, ok := target.(*player.Player); ok {
		return gc.player.IsPVP && otherPlayer.IsPVP
	}

	return true
}

// getGC returns the WorldSession for a player (avoids circular import).
func getGC(p *player.Player) *WorldSession {
	if p.Sender == nil {
		return nil
	}
	if gc, ok := p.Sender.(*WorldSession); ok {
		return gc
	}
	return nil
}

// --- Core Damage Calculation Pipeline ---

// CalculateMeleeDamage computes the full melee damage for a single swing.
func CalculateMeleeDamage(attacker, victim CombatUnit, attackType WeaponAttackType) *CalcDamageInfo {
	damageInfo := &CalcDamageInfo{
		Attacker:   attacker,
		Target:     victim,
		AttackType: attackType,
	}

	if !attacker.IsAlive() || !victim.IsAlive() {
		return damageInfo
	}

	// Roll hit outcome
	damageInfo.HitOutcome = RollMeleeOutcomeAgainst(attacker, victim, attackType)

	// Calculate base weapon damage
	weaponMin, weaponMax := attacker.GetWeaponDamage(uint8(attackType))
	if weaponMin == 0 && weaponMax == 0 {
		weaponMin = 1
		weaponMax = 2
	}

	// Add attack power contribution (1 damage per 14 AP)
	attackPower := attacker.GetAttackPower()
	apBonus := attackPower / 14

	baseDamage := uint32(0)
	if weaponMax > weaponMin {
		baseDamage = weaponMin + uint32(rand.Intn(int(weaponMax-weaponMin+1))) + apBonus
	} else {
		baseDamage = weaponMin + apBonus
	}

	// Apply hit outcome modifiers (Unit::CalculateMeleeDamage): misses and
	// the avoidances carry no damage and are told through HitInfo / TargetState
	damageInfo.HitInfo = HitInfoAffectsVictim
	damageInfo.TargetState = VictimStateHit

	switch damageInfo.HitOutcome {
	case MeleeHitMiss:
		damageInfo.Damages[0].Damage = 0
		damageInfo.HitInfo |= HitInfoMiss
		damageInfo.TargetState = VictimStateIntact
		return damageInfo

	case MeleeHitDodge:
		damageInfo.Damages[0].Damage = 0
		damageInfo.TargetState = VictimStateDodge
		return damageInfo

	case MeleeHitParry:
		damageInfo.Damages[0].Damage = 0
		damageInfo.TargetState = VictimStateParry
		return damageInfo

	case MeleeHitEvade:
		damageInfo.Damages[0].Damage = 0
		damageInfo.HitInfo |= HitInfoNoAnimation
		damageInfo.TargetState = VictimStateEvades
		return damageInfo

	case MeleeHitBlock:
		damageInfo.HitInfo |= HitInfoBlock
		damageInfo.TargetState = VictimStateBlocks

	case MeleeHitGlancing:
		glanceFactor := 0.7 + rand.Float64()*0.2
		baseDamage = uint32(float64(baseDamage) * glanceFactor)
		damageInfo.HitInfo |= HitInfoGlancing

	case MeleeHitCrit:
		baseDamage *= 2
		damageInfo.HitInfo |= HitInfoCriticalHit

	case MeleeHitCrushing:
		baseDamage = uint32(float64(baseDamage) * 1.5)
		damageInfo.HitInfo |= HitInfoCrushing
	}

	// Apply armor mitigation
	armor := victim.GetArmor()
	if armor > 0 {
		attackerLevel := attacker.GetLevel()
		reduction := CalcArmorReduction(armor, uint32(attackerLevel))
		mitigated := uint32(float64(baseDamage) * reduction)
		damageInfo.CleanDamage += mitigated
		baseDamage -= mitigated
	}

	damageInfo.Damages[0].Damage = baseDamage
	damageInfo.Damages[0].Clean = damageInfo.CleanDamage

	return damageInfo
}

// RollMeleeOutcomeAgainst determines the hit outcome using combat ratings.
func RollMeleeOutcomeAgainst(attacker, victim CombatUnit, attackType WeaponAttackType) MeleeHitOutcome {
	attackerLevel := int32(attacker.GetLevel())
	victimLevel := int32(victim.GetLevel())

	// Miss chance
	missChance := baseMissChance
	levelDiff := victimLevel - attackerLevel
	if levelDiff > 0 {
		missChance += float64(levelDiff) * 1.0
	} else if levelDiff < 0 {
		missChance += float64(levelDiff) * 0.5
	}

	hitRating := float64(attacker.GetCombatRating(uint8(CRHitMelee)))
	missChance -= hitRating * 0.01
	if missChance < 0 {
		missChance = 0
	}

	// Dodge chance
	dodgeChance := float64(victim.GetDodgeChance())
	expertiseRating := float64(attacker.GetCombatRating(uint8(CRExpertise)))
	dodgeChance -= expertiseRating * 0.01
	if dodgeChance < 0 {
		dodgeChance = 0
	}

	// Parry chance
	parryChance := float64(victim.GetParryChance())
	parryChance -= expertiseRating * 0.01
	if parryChance < 0 {
		parryChance = 0
	}

	// Block chance
	blockChance := float64(victim.GetBlockChance())
	if blockChance < 0 {
		blockChance = 0
	}

	// Glancing chance (attacker level < victim level)
	glancingChance := 0.0
	if attackerLevel < victimLevel {
		glancingChance = baseGlancingChance + float64(victimLevel-attackerLevel)*2.0
		if glancingChance > 40 {
			glancingChance = 40
		}
	}

	// Crit chance
	critChance := float64(attacker.GetCritChance())

	// Crushing chance (victim level >= attacker level + 3)
	crushingChance := 0.0
	if victimLevel >= attackerLevel+3 {
		crushingChance = baseCrushingChance * 0.75
	}

	// Roll the dice
	roll := rand.Float64() * 100.0
	cumulative := 0.0

	cumulative += missChance
	if roll < cumulative {
		return MeleeHitMiss
	}
	cumulative += dodgeChance
	if roll < cumulative {
		return MeleeHitDodge
	}
	cumulative += parryChance
	if roll < cumulative {
		return MeleeHitParry
	}
	cumulative += glancingChance
	if roll < cumulative {
		return MeleeHitGlancing
	}
	cumulative += blockChance
	if roll < cumulative {
		return MeleeHitBlock
	}
	cumulative += crushingChance
	if roll < cumulative {
		return MeleeHitCrushing
	}
	cumulative += critChance
	if roll < cumulative {
		return MeleeHitCrit
	}

	return MeleeHitNormal
}

// CalcArmorReduction returns the damage reduction factor from armor.
func CalcArmorReduction(armor, attackerLevel uint32) float64 {
	if armor == 0 || attackerLevel == 0 {
		return 0
	}
	return float64(armor) / (float64(armor) + 400.0 + 85.0*float64(attackerLevel))
}

// DealDamageMods adjusts damage for dead/immune targets.
func DealDamageMods(victim CombatUnit, damage *uint32, absorb *uint32) {
	if victim == nil || !victim.IsAlive() {
		if absorb != nil {
			*absorb += *damage
		}
		*damage = 0
	}
}

// DealDamage applies damage to victim and handles all side effects.
func DealDamage(attacker, victim CombatUnit, damageInfo *CalcDamageInfo) uint32 {
	if damageInfo == nil || victim == nil || !victim.IsAlive() {
		return 0
	}

	totalDamage := damageInfo.Damages[0].Damage
	if totalDamage == 0 {
		return 0
	}

	// AI hooks
	if victim.IsCreature() {
		victim.OnDamageTaken(attacker, totalDamage)
		victim.AddThreat(attacker, float32(totalDamage))
	}
	if attacker != nil && attacker.IsCreature() {
		attacker.OnDamageDealt(victim, totalDamage)
	}

	// Check if victim is in duel
	duelInfo := getDuelInfo(victim)
	if duelInfo != nil && duelInfo.State == DuelStateInProgress {
		health := victim.GetHealth()
		if totalDamage >= health {
			totalDamage = health - 1
			damageInfo.Damages[0].Damage = totalDamage
			completeDuel(duelInfo, DuelWon)
		}
	}

	// Grant rage for damage dealt
	if attacker != nil {
		grantRageOnDamageDealt(attacker, totalDamage, damageInfo)
	}

	// Apply damage to health
	currentHealth := victim.GetHealth()
	if totalDamage >= currentHealth {
		Kill(attacker, victim, damageInfo)
	} else {
		newHealth := currentHealth - totalDamage
		victim.SetHealth(newHealth)
		grantRageOnDamageTaken(victim, totalDamage)
	}

	return totalDamage
}

// DealSpellDamage applies spell damage to victim and handles all side effects.
// Mirrors AzerothCore Unit::DealSpellDamage / Unit::DealDamage.
func DealSpellDamage(attacker, victim CombatUnit, damage uint32) uint32 {
	if victim == nil || !victim.IsAlive() || damage == 0 {
		return 0
	}

	// AI hooks
	if victim.IsCreature() {
		victim.OnDamageTaken(attacker, damage)
		victim.AddThreat(attacker, float32(damage))
	}
	if attacker != nil && attacker.IsCreature() {
		attacker.OnDamageDealt(victim, damage)
	}

	// Check if victim is in duel
	duelInfo := getDuelInfo(victim)
	if duelInfo != nil && duelInfo.State == DuelStateInProgress {
		health := victim.GetHealth()
		if damage >= health {
			damage = health - 1
			completeDuel(duelInfo, DuelWon)
		}
	}

	// Apply damage to health
	currentHealth := victim.GetHealth()
	if damage >= currentHealth {
		Kill(attacker, victim, nil)
	} else {
		newHealth := currentHealth - damage
		victim.SetHealth(newHealth)
		grantRageOnDamageTaken(victim, damage)
	}

	return damage
}

// Kill handles the death of a unit.
func Kill(attacker, victim CombatUnit, damageInfo *CalcDamageInfo) {
	// Called from DealDamage before the health is written, so the victim is
	// still alive here; only a unit that is already dead is skipped.
	if victim == nil || !victim.IsAlive() {
		return
	}

	victim.SetHealth(0)
	victim.CombatStop()

	if victim.IsPlayer() {
		handlePlayerDeath(attacker, victim.(*player.Player), damageInfo)
	} else if victim.IsCreature() {
		handleCreatureDeath(attacker, victim, damageInfo)
	}
}

// handlePlayerDeath processes player death.
func handlePlayerDeath(attacker CombatUnit, victim *player.Player, damageInfo *CalcDamageInfo) {
	duelInfo := getDuelInfo(victim)
	if duelInfo != nil {
		completeDuel(duelInfo, DuelInterrupted)
	}

	victim.IsGhost = true
	victim.CharFlags |= 0x10
	victim.AttackState = 0
	victim.AttackTarget = 0
}

// handleCreatureDeath processes creature death: the corpse stays lootable
// for a while, then the respawn timer starts (Creature::setDeathState).
func handleCreatureDeath(attacker CombatUnit, victim CombatUnit, damageInfo *CalcDamageInfo) {
	npc, ok := victim.(*NPC)
	if !ok {
		return
	}

	npc.OnDeath()

	// Mark the corpse lootable only when it actually leaves something: the
	// client then shows the loot cursor and CMSG_LOOT succeeds, instead of
	// replying with an error for an empty corpse.
	// Source: AzerothCore Unit.cpp:13744-13754 (DealDamage death flow)
	if creatureHasLoot(npc, attacker) {
		npc.DynamicFlags |= UnitDynFlagLootable
		npc.Object.SetUInt32Value(object.UnitDynamicFlags, npc.DynamicFlags)
	}

	// Grant XP to the killing player
	// Source: AzerothCore KillRewarder.cpp, Player.cpp:GiveXP
	if p, ok := attacker.(*player.Player); ok {
		if gc := getGC(p); gc != nil {
			isElite := npc.Rank > 0

			// Determine content level from map (default to vanilla for now)
			content := Content1_60
			// TODO: proper map→content lookup using Map.dbc expansion field

			gain := BaseGain(p.Level, npc.Level, content)

			if gain > 0 && isElite {
				gain = uint32(float32(gain) * 2.0)
			}

			// TODO: apply creature.ModExperience multiplier when available

			if gain > 0 {
				gc.GiveXP(gain, npc.GetGUID(), 1.0)
			}

			// Grant quest kill credit
			// Source: AzerothCore KillRewarder.cpp (quest credit path)
			qm := gc.getQuestManager()
			if qm != nil {
				playerQuests := gc.getPlayerQuests()
				completedQuests := qm.OnCreatureKilled(playerQuests, npc.EntryID)
				for _, questID := range completedQuests {
					gc.log.Debug().
						Uint32("quest", questID).
						Uint32("creature", npc.EntryID).
						Msg("quest completed via kill credit")
				}
			}
		}
	}

	if p, ok := attacker.(*player.Player); ok {
		if gc := getGC(p); gc != nil {
			if server, ok := gc.ws.(*Server); ok && server.respawnMgr != nil {
				server.respawnMgr.ScheduleRespawn(npc)
			}
		}
	}
}

// creatureHasLoot reports whether a dead creature leaves anything to loot: a
// money range or a creature_loot_template for its lootid. Without a session
// (a pet or script kill) a non-zero lootid is trusted.
func creatureHasLoot(npc *NPC, attacker CombatUnit) bool {
	if npc.MinGold > 0 || npc.MaxGold > 0 {
		return true
	}

	if npc.LootID == 0 {
		return false
	}

	if p, ok := attacker.(*player.Player); ok {
		if gc := getGC(p); gc != nil {
			if server, ok := gc.ws.(*Server); ok && server.lootMgr != nil {
				return server.lootMgr.HasLoot(loot.StoreCreature, npc.LootID)
			}
		}
	}

	return true
}

// --- Duel Helpers ---

func getDuelInfo(unit CombatUnit) *DuelInfo {
	if unit == nil {
		return nil
	}
	if p, ok := unit.(*player.Player); ok {
		if di, ok := p.GetDuelInfo().(*DuelInfo); ok {
			return di
		}
	}
	return nil
}

func completeDuel(info *DuelInfo, result DuelCompleteType) {
	if info == nil || info.State == DuelStateCompleted {
		return
	}
	info.State = DuelStateCompleted
	if info.Opponent != nil {
		info.Opponent.CombatStop()
	}
	if info.Initiator != nil {
		info.Initiator.CombatStop()
	}
	if info.Opponent != nil {
		info.Opponent.SetDuelInfo(nil)
	}
	if info.Initiator != nil {
		info.Initiator.SetDuelInfo(nil)
	}
}

// --- Rage ---

func grantRageOnDamageDealt(attacker CombatUnit, damage uint32, damageInfo *CalcDamageInfo) {
	powerType := attacker.GetPrimaryPowerType()
	if powerType != wow.PowerTypeRage {
		return
	}
	rageGain := damage / 10
	if rageGain < 5 {
		rageGain = 5
	}
	if damageInfo != nil && damageInfo.HitOutcome == MeleeHitCrit {
		rageGain *= 2
	}
	currentRage := attacker.GetPower(powerType)
	newRage := currentRage + rageGain
	if newRage > 100 {
		newRage = 100
	}
	attacker.SetPower(powerType, newRage)
}

func grantRageOnDamageTaken(victim CombatUnit, damage uint32) {
	powerType := victim.GetPrimaryPowerType()
	if powerType != wow.PowerTypeRage {
		return
	}
	rageGain := damage / 60
	if rageGain < 1 && damage > 0 {
		rageGain = 1
	}
	currentRage := victim.GetPower(powerType)
	newRage := currentRage + rageGain
	if newRage > 100 {
		newRage = 100
	}
	victim.SetPower(powerType, newRage)
}

// --- NPC Aggro ---

// OnEnterCombat is called when an NPC is attacked by a player.
// Triggers aggro, sends AI reaction, and calls for assistance.
func (n *NPC) OnEnterCombat(attacker CombatUnit) {
	if attacker == nil {
		return
	}

	// Set victim
	n.victim = attacker

	// Add attacker
	n.AddAttacker(attacker.GetGUID())

	// Enter combat state
	n.SetInCombat()

	// TODO: SendAIReaction(AI_REACTION_HOSTILE)
	// TODO: CallAssistance() for nearby NPCs
	// TODO: UpdateLeashExtensionTime()
}

// --- Packet Sending ---

// sendAttackStart sends SMSG_ATTACKSTART to all nearby players.
func (gc *WorldSession) sendAttackStart(attacker, target wow.GUID) {
	pkt := wow.NewPacket(wow.ServerAttackstart)
	_ = pkt.Write(attacker)
	_ = pkt.Write(target)

	// Broadcast to all nearby
	server, ok := gc.ws.(*Server)
	if ok {
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && other.player.IsInWorld {
				other.socket.Send(pkt)
			}
		}
	}
}

// BuildAttackerStateUpdatePacket builds SMSG_ATTACKERSTATEUPDATE packet from CalcDamageInfo.
func BuildAttackerStateUpdatePacket(damageInfo *CalcDamageInfo) *wow.Packet {
	if damageInfo == nil || damageInfo.Target == nil || damageInfo.Attacker == nil {
		return nil
	}

	pkt := wow.NewPacket(wow.ServerAttackerstateupdate)

	_ = pkt.Write(uint32(damageInfo.HitInfo))
	pkt.WriteBytes(damageInfo.Attacker.GetGUID().Pack())
	pkt.WriteBytes(damageInfo.Target.GetGUID().Pack())

	totalDamage := damageInfo.Damages[0].Damage
	_ = pkt.Write(uint32(totalDamage))

	overkill := uint32(0)
	if totalDamage > damageInfo.Target.GetHealth() {
		overkill = totalDamage - damageInfo.Target.GetHealth()
	}
	_ = pkt.Write(uint32(overkill))

	// Sub damage count
	_ = pkt.Write(uint8(1))

	// Sub damage details
	_ = pkt.Write(uint32(SpellSchoolMaskPhysical)) // school mask
	_ = pkt.Write(float32(totalDamage))            // damage (float)
	_ = pkt.Write(uint32(totalDamage))             // damage (int)

	if damageInfo.HitInfo&(HitInfoFullAbsorb|HitInfoPartialAbsorb) != 0 {
		_ = pkt.Write(uint32(damageInfo.Damages[0].Absorb))
	}
	if damageInfo.HitInfo&(HitInfoFullResist|HitInfoPartialResist) != 0 {
		_ = pkt.Write(uint32(damageInfo.Damages[0].Resist))
	}

	_ = pkt.Write(damageInfo.TargetState)
	_ = pkt.Write(uint32(0)) // attacker state
	_ = pkt.Write(uint32(0)) // melee spell id

	if damageInfo.HitInfo&HitInfoBlock != 0 {
		_ = pkt.Write(uint32(damageInfo.Damages[0].Block))
	}

	return pkt
}

// sendAttackerStateUpdateFromCalc sends SMSG_ATTACKERSTATEUPDATE from CalcDamageInfo.
func (gc *WorldSession) sendAttackerStateUpdateFromCalc(damageInfo *CalcDamageInfo) {
	pkt := BuildAttackerStateUpdatePacket(damageInfo)
	if pkt == nil {
		return
	}

	// Send to all (including attacker)
	server, ok := gc.ws.(*Server)
	if ok {
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && other.player.IsInWorld {
				other.socket.Send(pkt)
			}
		}
	}
}

// sendAttackStop sends SMSG_ATTACKSTOP.
func (gc *WorldSession) sendAttackStop(attacker, target wow.GUID) {
	pkt := wow.NewPacket(wow.ServerAttackstop)
	pkt.WriteBytes(attacker.Pack())
	pkt.WriteBytes(target.Pack())
	_ = pkt.Write(uint32(0))

	server, ok := gc.ws.(*Server)
	if ok {
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && other.player.IsInWorld {
				other.socket.Send(pkt)
			}
		}
	}
}

// broadcastHealthUpdate sends health update for a target.
func (gc *WorldSession) broadcastHealthUpdate(target *player.Player) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}
	upd := &Updater{}
	pkt := upd.BuildValuesUpdateObject(target, target)
	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
}

// broadcastNPCHealthUpdate sends health update for an NPC.
func (gc *WorldSession) broadcastNPCHealthUpdate(npc *NPC) {
	server, ok := gc.ws.(*Server)
	if !ok {
		return
	}
	mask := &object.UpdateMask{}
	mask.SetCount(uint32(npc.Object.ValuesCount()))
	mask.SetBit(uint32(object.UnitFieldHealth))

	pkt := wow.NewPacket(wow.ServerUpdateObject)
	_ = pkt.WriteUint32(1)
	_ = pkt.WriteOne(0)
	_ = pkt.WriteOne(wow.UpdateTypeValues)
	_ = pkt.Write(npc.GetGUID())

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
	_ = pkt.Write(npc.GetHealth())

	for _, other := range server.GetOnlineSessions() {
		if other.player != nil && other.player.IsInWorld {
			other.socket.Send(pkt)
		}
	}
}

// --- Helper ---

func targetName(unit CombatUnit) string {
	if unit == nil {
		return "<nil>"
	}
	if p, ok := unit.(*player.Player); ok {
		return p.Name
	}
	if n, ok := unit.(*NPC); ok {
		return n.Name
	}
	return "Unknown"
}

// --- Faction-Aware Hostility ---

// IsHostileToCheck checks if self is hostile to target using faction templates.
// This is a world-package function that can access FactionManager.
func IsHostileToCheck(self, target CombatUnit) bool {
	if self == nil || target == nil {
		return false
	}

	// Duel opponents are always hostile
	if p, ok := self.(*player.Player); ok {
		if p.Duel != nil {
			if di, ok := p.Duel.(*player.DuelInfo); ok && di.Opponent != nil {
				if di.Opponent.GUID() == target.GetGUID() && di.State == player.DuelStateInProgress {
					return true
				}
			}
		}
	}

	fm := GetFactionManager()
	if !fm.IsLoaded() {
		// Fallback: use simple same-faction check
		return !IsFriendlyToCheck(self, target)
	}

	selfFaction := getFactionID(self)
	targetFaction := getFactionID(target)

	return fm.IsHostileTo(selfFaction, targetFaction)
}

// IsFriendlyToCheck checks if self is friendly to target using faction templates.
func IsFriendlyToCheck(self, target CombatUnit) bool {
	if self == nil || target == nil {
		return false
	}

	// Same GUID = always friendly
	if self.GetGUID() == target.GetGUID() {
		return true
	}

	selfFaction := getFactionID(self)
	targetFaction := getFactionID(target)

	fm := GetFactionManager()
	if !fm.IsLoaded() {
		// Fallback: simple same-faction check
		return selfFaction != 0 && selfFaction == targetFaction
	}

	return fm.IsFriendlyTo(selfFaction, targetFaction)
}

// getFactionID returns the faction template ID for a CombatUnit.
func getFactionID(unit CombatUnit) uint32 {
	if p, ok := unit.(*player.Player); ok {
		return p.FactionID
	}
	if n, ok := unit.(*NPC); ok {
		if n.FactionID != 0 {
			return n.FactionID
		}
		return n.Faction
	}
	return 0
}

// --- NPC Chase / Follow & Movement Generators ---

// ProcessNPCChase handles NPC movement toward a chase target.
// Called from the world update loop.
func ProcessNPCChase(npc *NPC, now time.Time, sendPacket func(pkt *wow.Packet)) {
	if npc == nil || !npc.InCombat {
		return
	}
	if npc.ChaseTarget == nil && npc.victim != nil {
		npc.ChaseTarget = npc.victim
	}
	if npc.ChaseTarget == nil {
		return
	}

	target, ok := npc.ChaseTarget.(CombatUnit)
	if !ok || !target.IsAlive() {
		npc.ExitCombat()
		return
	}

	// Calculate distance to target
	dx := target.GetPositionX() - npc.X
	dy := target.GetPositionY() - npc.Y
	dz := target.GetPositionZ() - npc.Z
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	// Check leash range — if too far, exit combat and return to spawn
	if distance > npc.ChaseRadius {
		npc.ExitCombat()
		npc.MoveTo(npc.SpawnX, npc.SpawnY, npc.SpawnZ, now, SplineFlagRunMode, sendPacket)
		return
	}

	// If in melee range, stop moving
	if distance < 3.0 {
		if npc.IsMoving {
			npc.StopMoving(sendPacket)
		}
		return
	}

	// If not currently moving, or destination is far from target (> 2.0 yards), generate new spline
	targetDestDiffX := target.GetPositionX() - npc.MoveTargetX
	targetDestDiffY := target.GetPositionY() - npc.MoveTargetY
	targetDestDistSq := targetDestDiffX*targetDestDiffX + targetDestDiffY*targetDestDiffY

	if !npc.IsMoving || targetDestDistSq > 2.0*2.0 {
		npc.MoveTo(target.GetPositionX(), target.GetPositionY(), target.GetPositionZ(), now, SplineFlagRunMode, sendPacket)
	}
}

// ProcessNPCWander handles idle random wandering for non-combat NPCs.
func ProcessNPCWander(npc *NPC, now time.Time, sendPacket func(pkt *wow.Packet)) {
	if npc == nil || npc.InCombat || !npc.IsAlive() || npc.IsMoving || npc.WanderRadius <= 0 {
		return
	}

	if npc.NextWanderTime == 0 {
		// Initialize first wander timer 5-15s in future
		npc.NextWanderTime = now.UnixMilli() + int64(5000+rand.Intn(10000))
		return
	}

	if now.UnixMilli() < npc.NextWanderTime {
		return
	}

	// Schedule next wander in 10-25s
	npc.NextWanderTime = now.UnixMilli() + int64(10000+rand.Intn(15000))

	// Pick random angle and distance within WanderRadius
	angle := rand.Float64() * 2 * math.Pi
	dist := 1.0 + rand.Float64()*float64(npc.WanderRadius-1.0)

	destX := npc.SpawnX + float32(dist*math.Cos(angle))
	destY := npc.SpawnY + float32(dist*math.Sin(angle))
	destZ := npc.SpawnZ

	npc.MoveTo(destX, destY, destZ, now, SplineFlagWalkMode, sendPacket)
}

// ProcessNPCWaypoint drives waypoint-path movement for creatures with
// MovementType == WAYPOINT_MOTION_TYPE (2).  Each tick it either waits
// for the current waypoint's delay to expire or advances to the next
// point and sends an SMSG_MONSTER_MOVE spline.
//
// Called from Server.updateNPCs for every alive, out-of-combat NPC that
// carries a WaypointPath.
func ProcessNPCWaypoint(npc *NPC, now time.Time, sendPacket func(pkt *wow.Packet)) {
	if npc == nil || npc.WaypointPath == nil || npc.InCombat || !npc.IsAlive() {
		return
	}

	points := npc.WaypointPath.Points
	if len(points) == 0 {
		return
	}

	// If currently moving along a spline, let it finish.
	if npc.IsMoving {
		return
	}

	// If waiting out a delay at a waypoint, check the timer.
	if npc.WaypointDelayEnd > 0 {
		if now.UnixMilli() < npc.WaypointDelayEnd {
			return // still waiting
		}
		npc.WaypointDelayEnd = 0 // delay expired, advance below
	}

	// Get current waypoint
	wp := points[npc.WaypointIndex]
	splineFlags := SplineFlagRunMode
	if wp.MoveType == 0 {
		splineFlags = SplineFlagWalkMode
	}

	// Handle smooth transition with catmull-rom spline
	if wp.SmoothTransition && len(points) >= 3 {
		// Generate a smooth spline path through multiple waypoints
		waypoints := make([][3]float32, 0, len(points))
		for _, p := range points {
			waypoints = append(waypoints, [3]float32{p.X, p.Y, p.Z})
		}

		// Generate spline points starting from current position
		splinePoints := GenerateCatmullRomSpline(waypoints, 10, true)

		// Find the starting index in the spline (closest to current position)
		startIdx := 0
		minDist := float32(math.MaxFloat32)
		for i, sp := range splinePoints {
			dist := Distance3D(npc.X, npc.Y, npc.Z, sp[0], sp[1], sp[2])
			if dist < minDist {
				minDist = dist
				startIdx = i
			}
		}

		// Take points from current position onwards
		splinePoints = splinePoints[startIdx:]
		if len(splinePoints) < 2 {
			splinePoints = [][3]float32{{npc.X, npc.Y, npc.Z}, {wp.X, wp.Y, wp.Z}}
		}

		// Calculate total distance and duration
		totalDist := float32(0)
		for i := 1; i < len(splinePoints); i++ {
			totalDist += Distance3D(
				splinePoints[i-1][0], splinePoints[i-1][1], splinePoints[i-1][2],
				splinePoints[i][0], splinePoints[i][1], splinePoints[i][2],
			)
		}

		speed := float32(7.0) // default run speed
		if wp.Velocity > 0 {
			speed = wp.Velocity
		} else if splineFlags&SplineFlagWalkMode != 0 {
			speed = 2.5
		}

		durationMs := uint32((totalDist / speed) * 1000)
		if durationMs < 100 {
			durationMs = 100
		}

		// Set NPC movement state
		npc.SplineID++
		npc.IsMoving = true
		npc.MoveStartX = npc.X
		npc.MoveStartY = npc.Y
		npc.MoveStartZ = npc.Z
		npc.MoveTargetX = wp.X
		npc.MoveTargetY = wp.Y
		npc.MoveTargetZ = wp.Z
		npc.MoveStartTime = now.UnixMilli()
		npc.MoveDurationMs = durationMs

		// Update orientation towards first spline point
		if len(splinePoints) > 1 {
			dx := splinePoints[1][0] - npc.X
			dy := splinePoints[1][1] - npc.Y
			npc.O = float32(math.Atan2(float64(dy), float64(dx)))
		}

		// Send spline packet with catmull-rom flag
		if sendPacket != nil {
			pkt := BuildMonsterMoveSplinePacket(
				npc.GUID(),
				npc.X, npc.Y, npc.Z,
				splinePoints,
				npc.SplineID,
				durationMs,
				splineFlags|SplineFlagCatmullRom,
			)
			sendPacket(pkt)
		}
	} else {
		// Standard point-to-point movement
		npc.MoveToWithVelocity(wp.X, wp.Y, wp.Z, wp.Velocity, now, splineFlags, sendPacket)
	}

	// Set delay if waypoint has one
	if wp.Delay > 0 {
		npc.WaypointDelayEnd = now.UnixMilli() + int64(wp.Delay)
	}

	// Advance index, wrapping around.
	npc.WaypointIndex = (npc.WaypointIndex + 1) % len(points)
}

// ProcessNPCCombatTick executes melee attack swings from an NPC against its victim.
func ProcessNPCCombatTick(npc *NPC, now time.Time, sendPacket func(pkt *wow.Packet)) {
	if npc == nil || !npc.InCombat || npc.victim == nil || !npc.IsAlive() {
		return
	}

	target, ok := npc.victim.(CombatUnit)
	if !ok || !target.IsAlive() {
		npc.victim = nil
		npc.ChaseTarget = nil
		return
	}

	// Check distance
	dx := target.GetPositionX() - npc.X
	dy := target.GetPositionY() - npc.Y
	dz := target.GetPositionZ() - npc.Z
	distSq := dx*dx + dy*dy + dz*dz
	// Melee range ~ 3.5 yards (or combat reach)
	if distSq > 3.5*3.5 {
		return
	}

	// Check swing timer
	if now.UnixMilli() < npc.NextAttackTime {
		return
	}

	// Calculate melee damage
	damageInfo := CalculateMeleeDamage(npc, target, BaseAttack)

	// Apply damage modifiers
	for i := range damageInfo.Damages {
		DealDamageMods(target, &damageInfo.Damages[i].Damage, &damageInfo.Damages[i].Absorb)
	}

	// Send attacker state update packet
	if sendPacket != nil {
		pkt := BuildAttackerStateUpdatePacket(damageInfo)
		if pkt != nil {
			sendPacket(pkt)
		}
	}

	// Deal the damage
	DealDamage(npc, target, damageInfo)

	// Schedule next swing
	swingDelay := time.Duration(SwingTimerMS) * time.Millisecond
	if npc.BaseAttackSpeed > 0 {
		swingDelay = npc.BaseAttackSpeed
	}
	npc.NextAttackTime = now.UnixMilli() + swingDelay.Milliseconds()
}

// --- 5-Second Combat Timer ---

// ProcessCombatTimer checks if combat should end (5s after last damage).
func ProcessCombatTimer(unit CombatUnit, now time.Time) {
	if unit == nil || !unit.IsInCombatState() {
		return
	}

	// Check if combat should end
	if p, ok := unit.(*player.Player); ok {
		if p.CombatEnd > 0 && now.UnixMilli() >= p.CombatEnd {
			p.ClearInCombat()
			p.CombatEnd = 0
		}
	} else if n, ok := unit.(*NPC); ok {
		if n.InCombat && len(n.Attackers) == 0 {
			// No attackers = exit combat
			n.ExitCombat()
		}
	}
}

// UpdateCombatEnd updates the combat end timer (called when damage is dealt).
func UpdateCombatEnd(unit CombatUnit) {
	if unit == nil {
		return
	}
	if p, ok := unit.(*player.Player); ok {
		p.CombatEnd = time.Now().UnixMilli() + 5000 // 5 seconds
	}
}

// ExitCombat forces an NPC out of combat and initiates evade mode.
func (n *NPC) ExitCombat() {
	n.InCombat = false
	n.victim = nil
	n.ChaseTarget = nil
	n.Attackers = nil
	n.ClearThreat()
	n.Object.RemoveFlag(object.UnitFieldFlags, UnitFlagInCombat)

	// In WoW / AC, evading creatures regenerate to full health
	if n.IsAlive() && n.Health < n.MaxHealth {
		n.SetHealth(n.MaxHealth)
	}

	if n.AI != nil {
		n.AI.ExitCombat()
	} else if n.MotionMaster != nil {
		n.MotionMaster.MoveTargetedHome()
	}
}
