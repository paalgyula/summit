package world

import (
	"strings"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/object"
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

	// Combat state
	FactionID uint32    // faction template ID (from creature_template)
	victim    interface{} // current target (CombatUnit)
	InCombat  bool      // NPC is in combat
	Attackers map[uint64]bool // GUIDs of units attacking this NPC
	
	// Chase state
	ChaseTarget interface{} // target being chased
	ChaseRadius float32     // max chase distance (leash)
	AggroRadius float32     // aggro detection range
}

// NewNPC creates a new NPC with the given parameters.
func NewNPC(entryID uint32, name string, displayID, faction uint32, level uint8, health uint32, x, y, z, o float32, mapID uint32, npcFlags uint32) *NPC {
	n := &NPC{
		Object:    object.NewObject(),
		Unit:      object.NewUnit(),
		SpawnID:   uint64(entryID),
		EntryID:   entryID,
		Name:      name,
		DisplayID: displayID,
		Faction:   faction,
		Level:     level,
		Health:    health,
		MaxHealth: health,
		X:         x,
		Y:         y,
		Z:         z,
		O:         o,
		Map:       mapID,
		NpcFlags:  npcFlags,
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
		X:             spawn.PosX,
		Y:             spawn.PosY,
		Z:             spawn.PosZ,
		O:             spawn.Orientation,
		Map:           spawn.MapID,
		NpcFlags:      npcFlags,
		UnitFlags:     tmpl.UnitFlags,
		DynamicFlags:  tmpl.DynamicFlags,
		SpawnTimeSecs: spawn.SpawnTimeSecs,
		Scale:         tmpl.Scale,
		AggroRadius:   20.0, // default 20 yard aggro range
		ChaseRadius:   50.0, // default 50 yard leash range
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

	// GUID — uses SpawnID as the counter so each spawn gets a unique GUID
	guid := wow.NewGUID(wow.UnitGUID, uint32(n.SpawnID))
	n.Object.SetGUID(guid)

	// Object fields
	n.Object.SetUInt32Value(object.ObjectFieldGuid, uint32(guid))
	n.Object.SetUInt32Value(object.ObjectFieldGuid+1, uint32(uint64(guid)>>32))
	n.Object.SetUInt32Value(object.ObjectFieldType, uint32(wow.TypeIDUnit))
	n.Object.SetUInt32Value(object.ObjectFieldEntry, n.EntryID)
	scale := n.Scale
	if scale <= 0 {
		scale = 1.0
	}

	n.Object.SetFloatValue(object.ObjectFieldScaleX, scale)

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

// AddThreat adds threat from an attacker (placeholder).
func (n *NPC) AddThreat(_ interface{}, _ float32) {}

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
}

// ClearInCombat removes the NPC from combat state.
func (n *NPC) ClearInCombat() {
	n.InCombat = false
	n.Attackers = nil
	n.victim = nil
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

// SpawnManager holds all spawned NPCs keyed by spawn ID.
type SpawnManager struct {
	npcs map[uint64]*NPC
	// templates by entry, for creature queries (nil without a world store)
	templates map[uint32]*store.CreatureTemplate
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

	sm.templates = templates
	spawned := 0

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
		sm.npcs[npc.SpawnID] = npc
		spawned++
	}

	log.Info().
		Int("spawns", spawned).
		Int("templates", len(templates)).
		Msg("loaded creature spawns from database")

	return sm
}

// SpawnNPC adds an NPC to the world.
func (sm *SpawnManager) SpawnNPC(npc *NPC) {
	sm.npcs[npc.SpawnID] = npc
}

// RemoveNPC removes an NPC from the world by spawn ID.
func (sm *SpawnManager) RemoveNPC(spawnID uint64) {
	delete(sm.npcs, spawnID)
}

// GetNPC returns an NPC by spawn ID.
func (sm *SpawnManager) GetNPC(spawnID uint64) *NPC {
	return sm.npcs[spawnID]
}

// GetNPCsInMap returns all NPCs in the given map.
func (sm *SpawnManager) GetNPCsInMap(mapID uint32) []*NPC {
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
	return len(sm.npcs)
}
