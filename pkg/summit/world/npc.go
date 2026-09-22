package world

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

// CreatureFlagExtraTrigger marks creature_template.flags_extra trigger creatures (invisible to players).
const CreatureFlagExtraTrigger = 0x80

// NPC represents a spawned creature in the world.
type NPC struct {
	*object.Object
	*object.Unit

	SpawnID       uint64
	EntryID       uint32
	Name          string
	DisplayID     uint32
	Faction       uint32
	Level         uint8
	Health        uint32
	MaxHealth     uint32
	BaseDamage    float32
	X, Y, Z, O    float32
	Map           uint32
	NpcFlags      uint32
	UnitFlags     uint32
	DynamicFlags  uint32
	SpawnTimeSecs uint32 // respawn delay from creature table
	// Scale is the model scale of the template (1.0 when unknown).
	Scale float32
	// Rank is the creature template rank (0=Normal, 1=Elite, 2=Rare Elite, 3=WorldBoss, 4=Rare).
	Rank uint32

	// Combat state
	FactionID uint32          // faction template ID (from creature_template)
	victim    interface{}     // current target (CombatUnit)
	InCombat  bool            // NPC is in combat
	Attackers map[uint64]bool // GUIDs of units attacking this NPC

	// Combat attack timing
	NextAttackTime  int64         // when next attack swing happens (Unix ms)
	BaseAttackSpeed time.Duration // base attack speed duration

	// Threat table
	threatMu   sync.RWMutex
	ThreatList map[uint64]float32 // GUID -> threat value

	// Chase state
	ChaseTarget interface{} // target being chased
	ChaseRadius float32     // max chase distance (leash)
	AggroRadius float32     // aggro detection range

	// Spawn coordinates (for returning home / wandering)
	SpawnX, SpawnY, SpawnZ, SpawnO float32

	// Movement type from creature spawn (0=idle, 1=random, 2=waypoint)
	MovementType uint8

	// Waypoint movement state (when MovementType == 2)
	WaypointPath     *store.WaypointPath
	WaypointIndex    int    // current waypoint index into WaypointPath.Points
	WaypointDelayEnd int64  // Unix ms when the current delay at a waypoint expires
	WaypointWalkMode bool   // true = walk, false = run for current path

	// Movement / Spline state
	SplineID       uint32
	IsMoving       bool
	MoveStartX     float32
	MoveStartY     float32
	MoveStartZ     float32
	MoveTargetX    float32
	MoveTargetY    float32
	MoveTargetZ    float32
	MoveStartTime  int64  // Unix ms
	MoveDurationMs uint32 // duration in ms
	NextWanderTime int64  // Unix ms
	WanderRadius   float32
}

// NewNPC creates a new NPC with the given parameters.
func NewNPC(entryID uint32, name string, displayID, faction uint32, level uint8, health uint32, x, y, z, o float32, mapID uint32, npcFlags uint32) *NPC {
	n := &NPC{
		Object:          object.NewObject(),
		Unit:            object.NewUnit(),
		SpawnID:         uint64(entryID),
		EntryID:         entryID,
		Name:            name,
		DisplayID:       displayID,
		Faction:         faction,
		Level:           level,
		Health:          health,
		MaxHealth:       health,
		X:               x,
		Y:               y,
		Z:               z,
		O:               o,
		SpawnX:          x,
		SpawnY:          y,
		SpawnZ:          z,
		SpawnO:          o,
		Map:             mapID,
		NpcFlags:        npcFlags,
		ThreatList:      make(map[uint64]float32),
		Attackers:       make(map[uint64]bool),
		BaseAttackSpeed: 2000 * time.Millisecond,
		AggroRadius:     20.0,
		ChaseRadius:     50.0,
		WanderRadius:    5.0,
	}

	n.init()

	return n
}

// pickDisplayID chooses one of the template's display ids for a spawn; the
// spawn id keeps the choice stable across restarts while spreading variants.
func pickDisplayID(modelIDs []uint32, spawnID uint64) uint32 {
	if len(modelIDs) == 0 {
		return 0
	}

	return modelIDs[spawnID%uint64(len(modelIDs))]
}

// NewNPCFromSpawn creates an NPC from database spawn + template data.
func NewNPCFromSpawn(spawn *store.CreatureSpawn, tmpl *store.CreatureTemplate) *NPC {
	health := tmpl.HealthMultiplier * 100
	if spawn.CurrentHealth > 0 {
		health = float32(spawn.CurrentHealth)
	}

	level := tmpl.MaxLevel
	if level == 0 {
		level = tmpl.MinLevel
	}

	npcFlags := tmpl.NpcFlag
	if spawn.NpcFlag != 0 {
		npcFlags = spawn.NpcFlag
	}

	n := &NPC{
		Object:        object.NewObject(),
		Unit:          object.NewUnit(),
		SpawnID:       spawn.SpawnID,
		EntryID:       spawn.Entry,
		Name:          tmpl.Name,
		DisplayID:     pickDisplayID(tmpl.ModelIDs, spawn.SpawnID),
		Faction:       tmpl.Faction,
		FactionID:     tmpl.Faction, // faction template ID
		Level:         level,
		Health:        uint32(health),
		MaxHealth:     uint32(health),
		X:               spawn.PosX,
		Y:               spawn.PosY,
		Z:               spawn.PosZ,
		O:               spawn.Orientation,
		SpawnX:          spawn.PosX,
		SpawnY:          spawn.PosY,
		SpawnZ:          spawn.PosZ,
		SpawnO:          spawn.Orientation,
		Map:             spawn.MapID,
		NpcFlags:        npcFlags,
		UnitFlags:       tmpl.UnitFlags,
		DynamicFlags:    tmpl.DynamicFlags,
		SpawnTimeSecs:   spawn.SpawnTimeSecs,
		Scale:           tmpl.Scale,
		Rank:            tmpl.Rank,
		AggroRadius:     20.0, // default 20 yard aggro range
		ChaseRadius:     50.0, // default 50 yard leash range
		ThreatList:      make(map[uint64]float32),
		Attackers:       make(map[uint64]bool),
		BaseAttackSpeed: 2000 * time.Millisecond,
	}

	// Determine movement type: spawn overrides template (0 in spawn means use template).
	n.MovementType = uint8(tmpl.MovementType)
	if spawn.MovementType != 0 {
		n.MovementType = spawn.MovementType
	}

	// Wire wander distance from spawn data (random movement radius).
	if spawn.WanderDistance > 0 {
		n.WanderRadius = spawn.WanderDistance
	} else if n.MovementType == store.MotionTypeRandom {
		n.WanderRadius = 5.0 // sensible default for random-movement creatures without explicit distance
	} else {
		n.WanderRadius = 0 // idle/waypoint creatures don't wander
	}

	// Disable wander for waypoint creatures
	if n.MovementType == store.MotionTypeWaypoint {
		n.WanderRadius = 0
	}

	if tmpl.BaseAttackTime > 0 {
		n.BaseAttackSpeed = time.Duration(tmpl.BaseAttackTime) * time.Millisecond
	}

	// Template speeds are multipliers of the base 2.5 / 7.0 yards per second
	if tmpl.SpeedWalk > 0 {
		n.Unit.Speed[wow.MoveTypeWalk] = 2.5 * tmpl.SpeedWalk
	}

	if tmpl.SpeedRun > 0 {
		n.Unit.Speed[wow.MoveTypeRun] = 7.0 * tmpl.SpeedRun
	}

	if spawn.UnitFlags != 0 {
		n.UnitFlags = spawn.UnitFlags
	}

	if spawn.DynamicFlags != 0 {
		n.DynamicFlags = spawn.DynamicFlags
	}

	n.init()

	return n
}

// init sets up the NPC's update fields.
func (n *NPC) init() {
	n.Object.InitValues(int(object.UnitEnd))
	// The type id selects the per-field visibility table used by
	// BuildFilteredUpdateMask; without it every unit field is dropped
	// from create blocks.
	n.Object.SetObjectTypeID(wow.TypeIDUnit)

	// GUID — uses SpawnID as the counter so each spawn gets a unique GUID
	guid := wow.NewGUID(wow.UnitGUID, uint32(n.SpawnID))
	n.Object.SetGUID(guid)

	// Object fields
	n.Object.SetUInt32Value(object.ObjectFieldGuid, uint32(guid))
	n.Object.SetUInt32Value(object.ObjectFieldGuid+1, uint32(uint64(guid)>>32))
	n.Object.SetUInt32Value(object.ObjectFieldType, uint32(wow.TypeIDUnit))
	n.Object.SetUInt32Value(object.ObjectFieldEntry, n.EntryID)
	n.Object.SetFloatValue(object.ObjectFieldScaleX, ComputeCreatureScale(n.Scale, n.DisplayID))

	// Unit fields
	n.Object.SetUInt32Value(object.UnitFieldDisplayid, n.DisplayID)
	n.Object.SetUInt32Value(object.UnitFieldNativedisplayid, n.DisplayID)
	n.Object.SetUInt32Value(object.UnitFieldFactiontemplate, n.Faction)
	n.Object.SetUInt32Value(object.UnitFieldLevel, uint32(n.Level))
	n.Object.SetUInt32Value(object.UnitFieldHealth, n.Health)
	n.Object.SetUInt32Value(object.UnitFieldMaxhealth, n.MaxHealth)

	// Unit flags
	n.Object.SetUInt32Value(object.UnitFieldFlags, 0x08) // UNIT_FLAG_PVP_ATTACKABLE

	if n.UnitFlags != 0 {
		n.Object.SetUInt32Value(object.UnitFieldFlags, n.UnitFlags)
	}

	// Dynamic flags
	if n.DynamicFlags != 0 {
		n.Object.SetUInt32Value(object.UnitDynamicFlags, n.DynamicFlags)
	}

	// Bounding radius
	n.Object.SetFloatValue(object.UnitFieldBoundingradius, 0.388999998569489)
	n.Object.SetFloatValue(object.UnitFieldCombatreach, 1.5)

	// Base attack time (2000ms)
	n.Object.SetUInt32Value(object.UnitFieldBaseattacktime, 2000)

	// NPC flags (quest giver, vendor, etc.)
	if n.NpcFlags > 0 {
		n.Object.SetUInt32Value(object.UnitNpcFlags, n.NpcFlags)
	}
}

// GUID returns the NPC's GUID.
func (n *NPC) GUID() wow.GUID {
	return wow.NewGUID(wow.UnitGUID, uint32(n.SpawnID))
}

// GetGUID returns the NPC's GUID (satisfies world.Unit interface).
func (n *NPC) GetGUID() wow.GUID {
	return n.GUID()
}

// GetPosition returns the NPC's world location (satisfies Map.AddNPC positionProvider).
func (n *NPC) GetPosition() *player.WorldLocation {
	return &player.WorldLocation{
		X:   n.X,
		Y:   n.Y,
		Z:   n.Z,
		O:   n.O,
		Map: n.Map,
	}
}

// GetNPCObject returns the NPC's underlying Object (satisfies Map.AddNPC objectProvider).
func (n *NPC) GetNPCObject() *object.Object {
	return n.Object
}

// GetHealth returns the NPC's current health.
func (n *NPC) GetHealth() uint32 {
	return n.Health
}

// SetHealth sets the NPC's health, clamped to [0, MaxHealth].
func (n *NPC) SetHealth(v uint32) {
	if v > n.MaxHealth {
		v = n.MaxHealth
	}

	n.Health = v
	n.Object.SetUInt32Value(object.UnitFieldHealth, v)
}

// IsDead returns true if the NPC has 0 health.
func (n *NPC) IsDead() bool {
	return n.Health == 0
}

// --- CombatUnit interface implementation ---

// GetMaxHealth returns the NPC's maximum health.
func (n *NPC) GetMaxHealth() uint32 { return n.MaxHealth }

// IsAlive returns true if the NPC has health > 0.
func (n *NPC) IsAlive() bool { return n.Health > 0 }

// IsPlayer returns false for NPCs.
func (n *NPC) IsPlayer() bool { return false }

// IsCreature returns true for NPCs.
func (n *NPC) IsCreature() bool { return true }

// IsPet returns false for NPCs (override for Pet type).
func (n *NPC) IsPet() bool { return false }

// IsTotem returns false for NPCs.
func (n *NPC) IsTotem() bool { return false }

// GetPositionX returns the NPC's X coordinate.
func (n *NPC) GetPositionX() float32 { return n.X }

// GetPositionY returns the NPC's Y coordinate.
func (n *NPC) GetPositionY() float32 { return n.Y }

// GetPositionZ returns the NPC's Z coordinate.
func (n *NPC) GetPositionZ() float32 { return n.Z }

// GetMapID returns the NPC's map ID.
func (n *NPC) GetMapID() uint32 { return n.Map }

// GetLevel returns the NPC's level as uint32.
func (n *NPC) GetLevel() uint32 { return uint32(n.Level) }

// GetAttackPower returns the NPC's attack power (level-based).
func (n *NPC) GetAttackPower() uint32 {
	return uint32(n.Level) * 3 // simplified
}

// GetWeaponDamage returns (min, max) damage for the NPC.
func (n *NPC) GetWeaponDamage(_ uint8) (uint32, uint32) {
	dmg := uint32(n.BaseDamage)
	if dmg == 0 {
		dmg = uint32(n.Level) * 2
	}
	return dmg, dmg + uint32(n.Level)
}

// GetWeaponSpeed returns the base attack speed.
func (n *NPC) GetWeaponSpeed(_ uint8) time.Duration {
	if n.BaseAttackSpeed > 0 {
		return n.BaseAttackSpeed
	}
	return 2000 * time.Millisecond // default 2.0s
}

// GetCombatRating returns 0 for NPCs.
func (n *NPC) GetCombatRating(_ uint8) uint32 { return 0 }

// GetArmor returns the NPC's armor (level-based).
func (n *NPC) GetArmor() uint32 {
	return uint32(n.Level) * 30 // simplified
}

// GetBlockChance returns 0 for NPCs.
func (n *NPC) GetBlockChance() float32 { return 0 }

// GetDodgeChance returns 5% base dodge for NPCs.
func (n *NPC) GetDodgeChance() float32 { return 5.0 }

// GetParryChance returns 0 for NPCs.
func (n *NPC) GetParryChance() float32 { return 0 }

// GetCritChance returns 0 for NPCs.
func (n *NPC) GetCritChance() float32 { return 0 }

// GetPrimaryPowerType returns mana for NPCs.
func (n *NPC) GetPrimaryPowerType() wow.PowerType { return wow.PowerTypeMana }

// GetPower returns 0 for NPCs (no power pool).
func (n *NPC) GetPower(_ wow.PowerType) uint32 { return 0 }

// SetPower is a no-op for NPCs.
func (n *NPC) SetPower(_ wow.PowerType, _ uint32) {}

// GetMaxPower returns 0 for NPCs.
func (n *NPC) GetMaxPower(_ wow.PowerType) uint32 { return 0 }

// GetVictim returns the NPC's current target (stored as interface{}).
func (n *NPC) GetVictim() interface{} { return n.victim }

// SetVictim sets the NPC's current target.
func (n *NPC) SetVictim(v interface{}) { n.victim = v }

// AddThreat adds threat from an attacker.
func (n *NPC) AddThreat(attacker interface{}, threat float32) {
	if attacker == nil || !n.IsAlive() {
		return
	}
	type guidGetter interface {
		GetGUID() wow.GUID
	}
	type aliveChecker interface {
		IsAlive() bool
	}
	gg, ok := attacker.(guidGetter)
	if !ok {
		return
	}
	if ac, ok := attacker.(aliveChecker); ok && !ac.IsAlive() {
		return
	}
	attackerGUID := gg.GetGUID()
	if attackerGUID == n.GUID() {
		return
	}

	n.threatMu.Lock()
	if n.ThreatList == nil {
		n.ThreatList = make(map[uint64]float32)
	}
	n.ThreatList[uint64(attackerGUID)] += threat
	currentThreat := n.ThreatList[uint64(attackerGUID)]
	n.threatMu.Unlock()

	n.AddAttacker(attackerGUID)
	n.SetInCombat()

	// Switch victim if none, or if new threat exceeds current victim by >10%
	if n.victim == nil {
		n.victim = attacker
		n.ChaseTarget = attacker
	} else if currentVictim, ok := n.victim.(guidGetter); ok {
		n.threatMu.RLock()
		victimThreat := n.ThreatList[uint64(currentVictim.GetGUID())]
		n.threatMu.RUnlock()

		if currentThreat > victimThreat*1.1 || !n.victimIsAlive() {
			n.victim = attacker
			n.ChaseTarget = attacker
		}
	}
}

// GetThreat returns the accumulated threat for a given GUID.
func (n *NPC) GetThreat(guid wow.GUID) float32 {
	n.threatMu.RLock()
	defer n.threatMu.RUnlock()

	if n.ThreatList == nil {
		return 0
	}

	return n.ThreatList[uint64(guid)]
}

// ClearThreat clears all entries in the threat table.
func (n *NPC) ClearThreat() {
	n.threatMu.Lock()
	defer n.threatMu.Unlock()

	n.ThreatList = make(map[uint64]float32)
}

// victimIsAlive checks whether the current victim is alive.
func (n *NPC) victimIsAlive() bool {
	if n.victim == nil {
		return false
	}
	type aliveChecker interface {
		IsAlive() bool
	}
	if ac, ok := n.victim.(aliveChecker); ok {
		return ac.IsAlive()
	}
	return true
}

// CombatStop stops NPC combat.
func (n *NPC) CombatStop() {
	n.victim = nil
}

// OnDamageTaken is called when the NPC takes damage.
func (n *NPC) OnDamageTaken(_ interface{}, _ uint32) {}

// OnDamageDealt is called when the NPC deals damage.
func (n *NPC) OnDamageDealt(_ interface{}, _ uint32) {}

// GetDuelInfo returns nil for NPCs.
func (n *NPC) GetDuelInfo() interface{} { return nil }

// SetDuelInfo is a no-op for NPCs.
func (n *NPC) SetDuelInfo(_ interface{}) {}

// IsMounted returns false for NPCs (creatures aren't mounted).
func (n *NPC) IsMounted() bool { return false }

// IsInSameMap returns true if the other unit is on the same map.
func (n *NPC) IsInSameMap(other interface{}) bool {
	if other == nil {
		return false
	}
	type mapGetter interface {
		GetMapID() uint32
	}
	if mg, ok := other.(mapGetter); ok {
		return n.Map == mg.GetMapID()
	}
	return false
}

// IsHostileTo returns true if the NPC is hostile to the other unit.
func (n *NPC) IsHostileTo(other interface{}) bool {
	if other == nil {
		return false
	}
	// TODO: proper faction template system
	if n.Faction == 1 {
		return true
	}
	return false
}

// IsFriendlyTo returns true if the NPC is friendly to the other unit.
func (n *NPC) IsFriendlyTo(other interface{}) bool {
	return !n.IsHostileTo(other)
}

// IsValidAttackTarget checks if this NPC can attack the target.
func (n *NPC) IsValidAttackTarget(target interface{}) bool {
	if target == nil {
		return false
	}

	type guidGetter interface {
		GetGUID() wow.GUID
	}
	type aliveChecker interface {
		IsAlive() bool
	}

	gg, ok := target.(guidGetter)
	if !ok {
		return false
	}

	ac, ok := target.(aliveChecker)
	if !ok {
		return false
	}

	// Can't attack self
	if n.GUID() == gg.GetGUID() {
		return false
	}

	// Can't attack dead targets
	if !ac.IsAlive() {
		return false
	}

	return true
}

// AddAttacker adds a GUID to the attacker set.
func (n *NPC) AddAttacker(guid wow.GUID) {
	if n.Attackers == nil {
		n.Attackers = make(map[uint64]bool)
	}
	n.Attackers[uint64(guid)] = true
}

// RemoveAttacker removes a GUID from the attacker set.
func (n *NPC) RemoveAttacker(guid wow.GUID) {
	if n.Attackers != nil {
		delete(n.Attackers, uint64(guid))
	}
}

// HasAttacker returns true if the given GUID is in the attacker set.
func (n *NPC) HasAttacker(guid wow.GUID) bool {
	if n.Attackers == nil {
		return false
	}
	return n.Attackers[uint64(guid)]
}

// GetAttackers returns the attacker set.
func (n *NPC) GetAttackers() map[uint64]bool {
	return n.Attackers
}

// SetInCombat puts the NPC into combat state.
func (n *NPC) SetInCombat() {
	n.InCombat = true
	n.Object.SetFlag(object.UnitFieldFlags, UnitFlagInCombat)
}

// ClearInCombat removes the NPC from combat state.
func (n *NPC) ClearInCombat() {
	n.InCombat = false
	n.Attackers = nil
	n.victim = nil
	n.ClearThreat()
	n.Object.RemoveFlag(object.UnitFieldFlags, UnitFlagInCombat)
}

// OnDeath switches the NPC into its corpse state: no more services, no
// combat, and the client shows it dead through UNIT_FIELD_HEALTH = 0.
func (n *NPC) OnDeath() {
	n.ClearInCombat()
	n.ChaseTarget = nil
	n.Object.SetUInt32Value(object.UnitNpcFlags, 0)
}

// Reset brings a despawned NPC back to its spawn state for respawning.
func (n *NPC) Reset() {
	n.ClearInCombat()
	n.ChaseTarget = nil
	n.SetHealth(n.MaxHealth)
	n.Object.SetUInt32Value(object.UnitNpcFlags, n.NpcFlags)
}

// IsInCombatState returns true if the NPC is in combat.
func (n *NPC) IsInCombatState() bool {
	return n.InCombat
}

// Attack initiates combat against a target (for NPCs/creatures).
func (n *NPC) Attack(victim interface{}, meleeAttack bool) {
	if victim == nil {
		return
	}
	type guidGetter interface {
		GetGUID() wow.GUID
	}
	gg, ok := victim.(guidGetter)
	if !ok || gg.GetGUID() == n.GUID() {
		return
	}
	if !n.IsAlive() {
		return
	}

	n.victim = victim
	n.AddAttacker(gg.GetGUID())
	n.SetInCombat()
}

// AttackStop stops the current attack.
func (n *NPC) AttackStop() {
	n.victim = nil
	n.InCombat = false
}

// MoveTo sets a destination and begins movement towards it.
func (n *NPC) MoveTo(destX, destY, destZ float32, now time.Time, flags uint32, sendPacket func(pkt *wow.Packet)) {
	n.MoveToWithVelocity(destX, destY, destZ, 0, now, flags, sendPacket)
}

// MoveToWithVelocity moves the NPC to the destination with an optional custom velocity.
// If velocity is 0, uses the default speed based on flags.
func (n *NPC) MoveToWithVelocity(destX, destY, destZ, velocity float32, now time.Time, flags uint32, sendPacket func(pkt *wow.Packet)) {
	dx := destX - n.X
	dy := destY - n.Y
	dz := destZ - n.Z
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if dist < 0.2 {
		return
	}

	speed := float32(7.0) // default run speed
	if flags&SplineFlagWalkMode != 0 {
		speed = 2.5 // walk speed
	}

	// Use custom velocity if provided
	if velocity > 0 {
		speed = velocity
	} else if n.Unit != nil {
		if flags&SplineFlagWalkMode != 0 && n.Unit.Speed[wow.MoveTypeWalk] > 0 {
			speed = n.Unit.Speed[wow.MoveTypeWalk]
		} else if flags&SplineFlagWalkMode == 0 && n.Unit.Speed[wow.MoveTypeRun] > 0 {
			speed = n.Unit.Speed[wow.MoveTypeRun]
		}
	}

	durationMs := uint32((dist / speed) * 1000)
	if durationMs < 100 {
		durationMs = 100
	}

	n.SplineID++
	n.IsMoving = true
	n.MoveStartX = n.X
	n.MoveStartY = n.Y
	n.MoveStartZ = n.Z
	n.MoveTargetX = destX
	n.MoveTargetY = destY
	n.MoveTargetZ = destZ
	n.MoveStartTime = now.UnixMilli()
	n.MoveDurationMs = durationMs

	// Update orientation towards destination
	n.O = float32(math.Atan2(float64(dy), float64(dx)))

	if sendPacket != nil {
		pkt := BuildMonsterMovePacket(n.GUID(), n.X, n.Y, n.Z, destX, destY, destZ, n.SplineID, durationMs, flags)
		sendPacket(pkt)
	}
}

// StopMoving stops active spline movement and broadcasts a stop packet.
func (n *NPC) StopMoving(sendPacket func(pkt *wow.Packet)) {
	if !n.IsMoving {
		return
	}
	n.IsMoving = false
	n.SplineID++
	if sendPacket != nil {
		pkt := BuildMonsterMoveStopPacket(n.GUID(), n.X, n.Y, n.Z, n.SplineID)
		sendPacket(pkt)
	}
}

// UpdatePositionFromSpline updates the NPC's world coordinates along its active spline.
func (n *NPC) UpdatePositionFromSpline(now time.Time) {
	if !n.IsMoving {
		return
	}

	elapsed := now.UnixMilli() - n.MoveStartTime
	if elapsed >= int64(n.MoveDurationMs) {
		n.X = n.MoveTargetX
		n.Y = n.MoveTargetY
		n.Z = n.MoveTargetZ
		n.IsMoving = false

		// When a waypoint move completes, start the delay timer for the
		// waypoint we just arrived at (looked up by the index that was
		// advanced *before* MoveTo was called, i.e. one past the waypoint
		// we arrived at).
		if n.WaypointPath != nil && len(n.WaypointPath.Points) > 0 {
			// The waypoint we arrived at is (WaypointIndex - 1) mod len,
			// because WaypointIndex was advanced after the MoveTo call.
			prevIdx := n.WaypointIndex - 1
			if prevIdx < 0 {
				prevIdx = len(n.WaypointPath.Points) - 1
			}
			wp := n.WaypointPath.Points[prevIdx]
			if wp.Delay > 0 {
				n.WaypointDelayEnd = now.UnixMilli() + int64(wp.Delay)
			}
		}

		return
	}

	if n.MoveDurationMs > 0 {
		progress := float32(elapsed) / float32(n.MoveDurationMs)
		if progress > 1.0 {
			progress = 1.0
		}
		n.X = n.MoveStartX + (n.MoveTargetX-n.MoveStartX)*progress
		n.Y = n.MoveStartY + (n.MoveTargetY-n.MoveStartY)*progress
		n.Z = n.MoveStartZ + (n.MoveTargetZ-n.MoveStartZ)*progress
	}
}

// SpawnManager holds all spawned NPCs keyed by spawn ID. Sessions read it
// while respawn goroutines add and remove NPCs, hence the lock.
type SpawnManager struct {
	mu   sync.RWMutex
	npcs map[uint64]*NPC
	// templates by entry, for creature queries (nil without a world store)
	templates map[uint32]*store.CreatureTemplate
	// waypoint paths keyed by path_id (nil without a world store)
	waypointPaths map[uint32]*store.WaypointPath
}

// Template returns the creature template of an entry, or nil.
func (sm *SpawnManager) Template(entry uint32) *store.CreatureTemplate {
	return sm.templates[entry]
}

// NewSpawnManager creates a spawn manager with hardcoded test NPCs.
func NewSpawnManager() *SpawnManager {
	sm := &SpawnManager{
		npcs: make(map[uint64]*NPC),
	}

	// Spawn a training dummy at starting location
	sm.SpawnNPC(NewNPC(
		100001,           // entry ID
		"Training Dummy", // name
		49,               // displayID (human male)
		1,                // faction (hostile)
		1,                // level
		50,               // health
		-8949.95,         // X (Elwynn Forest starting area)
		-132.66,          // Y
		83.53,            // Z
		1.0,              // O
		0,                // map (Eastern Kingdoms)
		0,                // npcFlags
	))

	return sm
}

// NewSpawnManagerFromDB creates a spawn manager loaded from the world store.
func NewSpawnManagerFromDB(worldRepo store.WorldRepo) *SpawnManager {
	sm := &SpawnManager{
		npcs: make(map[uint64]*NPC),
	}

	if worldRepo == nil {
		log.Warn().Msg("no world repo, using hardcoded spawns only")

		return sm
	}

	// Load creature templates
	templates, err := worldRepo.GetCreatureTemplates()
	if err != nil {
		log.Warn().Err(err).Msg("failed to load creature templates for spawning")

		return sm
	}

	// Load all creature spawns (map 0 = Eastern Kingdoms, but we load all)
	spawns, err := worldRepo.GetCreatureSpawns(0)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load creature spawns")

		return sm
	}

	// Load creature addons (per-spawn path_id, mount, etc.)
	addons, err := worldRepo.GetAllCreatureAddons()
	if err != nil {
		log.Warn().Err(err).Msg("failed to load creature addons")
		addons = make(map[uint32]*store.CreatureAddon) // proceed without addons
	}

	// Load all waypoint paths so we can attach them to NPCs
	waypointPaths, err := worldRepo.GetAllWaypointPaths()
	if err != nil {
		log.Warn().Err(err).Msg("failed to load waypoint paths")
		waypointPaths = make(map[uint32]*store.WaypointPath) // proceed without waypoints
	}

	sm.templates = templates
	sm.waypointPaths = waypointPaths
	spawned := 0
	waypointNPCs := 0
	randomNPCs := 0

	for _, spawn := range spawns {
		tmpl, ok := templates[spawn.Entry]
		if !ok {
			log.Debug().Uint32("entry", spawn.Entry).Msg("creature template not found for spawn")

			continue
		}

		// Invisible trigger creatures and [DND] development placeholders are GM-only in the real game
		if tmpl.FlagsExtra&CreatureFlagExtraTrigger != 0 || strings.HasPrefix(tmpl.Name, "[DND]") {
			continue
		}

		npc := NewNPCFromSpawn(spawn, tmpl)

		// Attach creature_addon data (path_id → waypoint path)
		if addon, ok := addons[uint32(npc.SpawnID)]; ok && addon.PathID > 0 {
			if path, ok := waypointPaths[addon.PathID]; ok && len(path.Points) > 0 {
				npc.WaypointPath = path
				npc.MovementType = store.MotionTypeWaypoint
				npc.WanderRadius = 0
				waypointNPCs++
			}
		}

		// Track random-movement creatures for logging
		if npc.MovementType == store.MotionTypeRandom {
			randomNPCs++
		}

		sm.npcs[npc.SpawnID] = npc
		spawned++
	}

	log.Info().
		Int("spawns", spawned).
		Int("templates", len(templates)).
		Int("waypoint", waypointNPCs).
		Int("random", randomNPCs).
		Msg("loaded creature spawns from database")

	return sm
}

// SpawnNPC adds an NPC to the world.
func (sm *SpawnManager) SpawnNPC(npc *NPC) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.npcs[npc.SpawnID] = npc
}

// RemoveNPC removes an NPC from the world by spawn ID.
func (sm *SpawnManager) RemoveNPC(spawnID uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.npcs, spawnID)
}

// GetNPC returns an NPC by spawn ID.
func (sm *SpawnManager) GetNPC(spawnID uint64) *NPC {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.npcs[spawnID]
}

// GetNPCs returns all spawned NPCs.
func (sm *SpawnManager) GetNPCs() []*NPC {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*NPC, 0, len(sm.npcs))
	for _, npc := range sm.npcs {
		result = append(result, npc)
	}

	return result
}

// GetNPCsInMap returns all NPCs in the given map.
func (sm *SpawnManager) GetNPCsInMap(mapID uint32) []*NPC {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []*NPC

	for _, npc := range sm.npcs {
		if npc.Map == mapID {
			result = append(result, npc)
		}
	}

	return result
}

// Count returns the total number of spawned NPCs.
func (sm *SpawnManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.npcs)
}
