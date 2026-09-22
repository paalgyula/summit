package player

import (
	"math"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

// UnitFlagInCombat is UNIT_FLAG_IN_COMBAT of UNIT_FIELD_FLAGS.
const UnitFlagInCombat uint32 = 0x00080000

// DuelState mirrors AC's DuelState enum (Player.h:355).
type DuelState uint8

const (
	DuelStateChallenged DuelState = 0
	DuelStateCountdown  DuelState = 1
	DuelStateInProgress DuelState = 2
	DuelStateCompleted  DuelState = 3
)

// Attack states.
const (
	AttackStateIdle     = 0
	AttackStateSwinging = 1
)

// DuelInfo holds duel state for a player.
type DuelInfo struct {
	Opponent        *Player
	Initiator       *Player
	IsMounted       bool
	State           DuelState
	StartTime       time.Time
	OutOfBoundsTime time.Time
	FlagGUID        wow.GUID
}

// PacketSender is the interface used by the map update system to send
// packets to a player's session.
type PacketSender interface {
	Send(pkt *wow.Packet)
}

type WorldLocation struct {
	X, Y, Z, O float32
	Map        uint32
	Zone       uint32
}

// Location returns the X, Y, Z coordinates and map ID of a WorldLocation.
//
// No parameters are required.
// Returns a float32 for X, Y, and Z coordinates, and a uint32 for the map ID.
func (l WorldLocation) Location() (float32, float32, float32, uint32) {
	return l.X, l.Y, l.Z, l.Map
}

// Distance calculates the distance between two locations.
func (l *WorldLocation) Distance(point *WorldLocation) float64 {
	dx := l.X - point.X
	dy := l.Y - point.Y
	dz := l.Z - point.Z

	return math.Sqrt(float64(dx*dx + dy*dy + dz*dz))
}

// FactionTemplateFromRace maps playable races to their default faction template from ChrRaces.dbc.
func FactionTemplateFromRace(race wow.PlayerRace) uint32 {
	switch race {
	case wow.RaceHuman:
		return 1
	case wow.RaceOrc:
		return 2
	case wow.RaceDwarf:
		return 3
	case wow.RaceNightElf:
		return 4
	case wow.RaceUndead:
		return 5
	case wow.RaceTauren:
		return 6
	case wow.RaceGnome:
		return 115
	case wow.RaceTroll:
		return 116
	case wow.RaceBloodElf:
		return 1610
	case wow.RaceDraenei:
		return 1629
	default:
		return 1
	}
}

func NewPlayer() *Player {
	p := &Player{
		Object: object.NewObject(),
		Unit:   object.NewUnit(),
	}

	return p
}

//nolint:funlen
func CreatePlayer() {
	// p := NewPlayer()
	// // uint8 powertype = cEntry->powerType;

	// var unitfield uint32

	// powertype := wow.PowerTypeRage

	// _, _, _ = p, unitfield, powertype

	// switch powertype {
	// case wow.PowerTypeEnergy,wow.PowerTypeMana:
	// 	unitfield = 0x00000000
	// case wow.PowerTypeRage:
	// 	unitfield = 0x00110000
	// default:
	// 	log.Warn().Msgf("Invalid default powertype %s for player (class %T)", powertype)
	// 	return
	// }

	// p.Object.SetFloatValue(object.UnitFieldBoundingradius), DEFAULT_WORLD_OBJECT_SIZE);
	// p.Object.SetFloatValue(UNIT_FIELD_COMBATREACH, DEFAULT_COMBAT_REACH);

	// switch (gender)
	// {
	//     case GENDER_FEMALE:
	//         SetDisplayId(info->displayId_f);
	//         SetNativeDisplayId(info->displayId_f);
	//         break;
	//     case GENDER_MALE:
	//         SetDisplayId(info->displayId_m);
	//         SetNativeDisplayId(info->displayId_m);
	//         break;
	//     default:
	//         sLog.outError("Invalid gender %u for player", gender);
	//         return false;
	//         break;
	// }

	// setFactionForRace(race);

	// RaceClassGender uint32 = (race) | (class_ << 8) | (gender << 16);

	// p.SetUInt32Value(UNIT_FIELD_BYTES_0, (RaceClassGender | (powertype << 24)));
	// SetUInt32Value(UNIT_FIELD_BYTES_1, unitfield);
	// SetByteValue(UNIT_FIELD_BYTES_2, 1, UNIT_BYTE2_FLAG_SANCTUARY | UNIT_BYTE2_FLAG_UNK5);
	// SetUInt32Value(UNIT_FIELD_FLAGS, UNIT_FLAG_PVP_ATTACKABLE);
	// SetFloatValue(UNIT_MOD_CAST_SPEED, 1.0f);               // fix cast time showed in spell tooltip on client

	// // -1 is default value
	// SetInt32Value(PLAYER_FIELD_WATCHED_FACTION_INDEX, uint32(-1));

	// SetUInt32Value(PLAYER_BYTES, (skin | (face << 8) | (hairStyle << 16) | (hairColor << 24)));
	// SetUInt32Value(PLAYER_BYTES_2, (facialHair | (0x00 << 8) | (0x00 << 16) | (0x02 << 24)));
	// SetByteValue(PLAYER_BYTES_3, 0, gender);

	// SetUInt32Value(PLAYER_GUILDID, 0);
	// SetUInt32Value(PLAYER_GUILDRANK, 0);
	// SetUInt32Value(PLAYER_GUILD_TIMESTAMP, 0);

	// for (int i = 0; i < KNOWN_TITLES_SIZE; ++i)
	//     SetUInt64Value(PLAYER__FIELD_KNOWN_TITLES + i, 0);  // 0=disabled
	// SetUInt32Value(PLAYER_CHOSEN_TITLE, 0);

	// SetUInt32Value(PLAYER_FIELD_KILLS, 0);
	// SetUInt32Value(PLAYER_FIELD_LIFETIME_HONORABLE_KILLS, 0);
	// SetUInt32Value(PLAYER_FIELD_TODAY_CONTRIBUTION, 0);
	// SetUInt32Value(PLAYER_FIELD_YESTERDAY_CONTRIBUTION, 0);

	// // set starting level
	// uint32 start_level = sWorld.getConfig(CONFIG_START_PLAYER_LEVEL);

	// if (GetSession()->GetSecurity() >= SEC_MODERATOR)
	// {
	//     uint32 gm_level = sWorld.getConfig(CONFIG_START_GM_LEVEL);
	//     if (gm_level > start_level)
	//         start_level = gm_level;
	// }

	// SetUInt32Value(UNIT_FIELD_LEVEL, start_level);
	// SetUInt32Value (PLAYER_FIELD_COINAGE, sWorld.getConfig(CONFIG_START_PLAYER_MONEY));
	// SetUInt32Value (PLAYER_FIELD_HONOR_CURRENCY, sWorld.getConfig(CONFIG_START_HONOR_POINTS));
	// SetUInt32Value (PLAYER_FIELD_ARENA_CURRENCY, sWorld.getConfig(CONFIG_START_ARENA_POINTS));

	// // start with every map explored
	// if (sWorld.getConfig(CONFIG_START_ALL_EXPLORED))
	// {
	//     for (uint8 i = 0; i < 64; i++)
	//         SetFlag(PLAYER_EXPLORED_ZONES_1 + i, 0xFFFFFFFF);
	// }
}

type Player struct {
	*object.Object
	*object.Unit

	ID     uint32
	Name   string
	Race   wow.PlayerRace
	Class  wow.PlayerClass
	Gender wow.PlayerGender

	Skin       uint8
	Face       uint8
	HairStyle  uint8
	HairColor  uint8
	FacialHair uint8
	OutfitID   uint8

	Location     WorldLocation
	BindLocation WorldLocation

	Level uint8
	XP    uint32
	Money uint32

	Health    uint32
	MaxHealth uint32
	Power     [wow.MaxPowerTypes]uint32
	MaxPower  [wow.MaxPowerTypes]uint32

	DisplayID       uint32
	NativeDisplayID uint32
	MountDisplayID  uint32

	Inventory *Inventory
	GuildID   uint32

	// CharFlags for example dead, and display ghost
	CharFlags uint32

	// PlayerFlags (AFK, DND, ghost, resting, etc)
	PlayerFlags wow.PlayerFlag

	// Recustomization flags (change name, look, etc)
	Recustomization uint32

	FirstLogin uint8

	Pet Pet

	// IsInWorld indicates if the player is currently in the world
	IsInWorld bool

	// GroupID is the ID of the group the player is in (0 if not in a group)
	GroupID uint32

	// MoveFlags stores the current movement flags for the player
	MoveFlags wow.MovementFlag

	// Castable spells (spell IDs known by the player)
	KnownSpells []uint32

	// Active spell being cast (interface to avoid circular import)
	ActiveSpell interface{}

	// Active auras (buffs/debuffs) - interface to avoid circular import
	Auras []interface{}

	// SpellModifiers from talents/gear - ModifierList stored as interface to avoid circular import
	SpellModifiers interface{} `json:"-"`

	// Spell cooldowns (spell ID or category → expiry time)
	SpellCooldowns map[uint32]time.Time

	// Action bar (12 buttons per bar, 3 bars + stance bar)
	Actions [48]uint32

	// Attack state
	// Target is the selected unit (CMSG_SET_SELECTION), 0 for none
	Target uint64

	AttackTarget   uint64 // GUID of current attack target
	AttackState    int    // 0 = idle, 1 = swinging
	NextAttackTime int64  // when next swing happens (Unix ms)
	BaseDamage     float32

	// Loot state
	LootGUID   uint64     // GUID of the object being looted
	ActiveLoot interface{} // *lootSource from loot_handler.go (avoids import cycle)

	// Regen state
	NextRegenTime int64 // when next regen tick happens (Unix ms)

	// Combat state
	FactionID   uint32   // faction template ID (from ChrRaces.dbc)
	Mounted     bool     // player is mounted (can't attack)
	InCombat    bool     // player is in combat
	CombatTime  int64    // when combat started (Unix ms), for 5s rule
	CombatEnd   int64    // when combat should end (Unix ms)
	Attackers   map[uint64]bool // GUIDs of units attacking this player
	IsPVP       bool     // PvP flag enabled
	IsFFAPVP    bool     // FFA PvP zone
	IsSanctuary bool     // sanctuary zone (can't PvP)
	IsGhost     bool     // dead/ghost state
	DeathTime   int64    // when player died (Unix ms)

	// Chat flood throttle
	ChatFloodCount   int
	ChatFloodResetAt time.Time

	// CurrentMap is the map the player is currently on (interface to avoid circular import)
	CurrentMap interface{}

	// Sender is set when the player is added to a session. The map uses
	// it to send update packets back to the client.
	Sender PacketSender

	// Quest state — map[uint32]*quest.QuestStatusData stored as interface{}
	// to avoid circular import with the quest package.
	QuestStatus interface{}

	// Rewarded quest IDs — map[uint32]bool stored as interface{}
	RewardedQuests interface{}

	// CurrentMapID is the map ID the player is on
	CurrentMapID uint32

	// CurrentInstanceID is the instance ID (0 for non-instanced maps)
	CurrentInstanceID uint32

	// Duel state — stored as interface{} to avoid circular import with world package.
	// The world package casts this to *world.DuelInfo when needed.
	Duel interface{}

	// Victim is the current attack target (set when entering combat).
	// Stored as interface{} to avoid circular import.
	Victim interface{}

	// Combat stats (populated on login from character data)
	Strength     uint32
	Agility      uint32
	Stamina      uint32
	Intellect    uint32
	Spirit       uint32
	Armor        uint32
	AttackPower  uint32
	BaseAttackSpeed time.Duration

	// Weapon damage ranges (min, max) for mainhand and offhand
	MainHandDamageMin, MainHandDamageMax uint32
	OffHandDamageMin, OffHandDamageMax   uint32
}

// InitInventory fills a new character's inventory with its starting outfit
// (CharStartOutfit.dbc): equippable items go to the slot their inventory type
// selects, the rest (food, drink, ammo) to the backpack. The slots can be
// nil, in this case the inventory is left empty.
func (p *Player) InitInventory(slots []*basedata.InventorySlot) {
	if p.Inventory != nil {
		return
	}

	p.Inventory = NewInventory()

	for _, slot := range slots {
		if slot == nil || slot.ItemID <= 0 {
			continue
		}

		item := NewItem(uint32(slot.ItemID), p.GUID())
		item.SetEnchant(0, 0)

		equipSlot := FindEquipSlot(slot.InventoryType)
		if equipSlot == EquipmentSlotMainHand && p.Inventory.GetEquipment(equipSlot) != nil &&
			slot.InventoryType == wow.InventoryTypeWeapon {
			equipSlot = EquipmentSlotOffHand // second one-hander: dual wield
		}

		if equipSlot >= 0 && p.Inventory.GetEquipment(equipSlot) == nil {
			p.Inventory.SetEquipment(equipSlot, item)

			continue
		}

		for i := InventorySlotItemStart; i < InventorySlotItemEnd; i++ {
			if p.Inventory.GetItem(i) == nil {
				p.Inventory.SetItem(i, item)

				break
			}
		}
	}
}

func (p *Player) GUID() wow.GUID {
	return wow.NewPlayerGUID(p.ID)
}

// GetGUID returns the player's GUID (satisfies the world.Unit interface).
func (p *Player) GetGUID() wow.GUID {
	return p.GUID()
}

func (p *Player) Init() {
	if p.Object == nil {
		p.Object = object.NewObject()
	}
	if p.Unit == nil {
		p.Unit = object.NewUnit()
	}

	// The update system reads the type and update flags off p.Object, which
	// is a plain object: make it describe a player (living unit with a
	// stationary position, like Unit::Unit in AzerothCore).
	p.Object.SetGUID(p.GUID())
	p.Object.SetObjectTypeID(wow.TypeIDPlayer)
	p.Object.SetObjectType(wow.TypeMaskObject | wow.TypeMaskPlayer)
	p.Object.AddUpdateFlags(wow.UpdateFlagLiving | wow.UpdateFlagStationaryPosition)

	// Set default health/mana based on class
	p.MaxHealth = 100
	p.Health = 100

	// Default power type by class
	powerType := p.primaryPowerType()
	if powerType >= 0 && int(powerType) < wow.MaxPowerTypes {
		p.MaxPower[powerType] = 100
		p.Power[powerType] = 100
	}

	// Default display ID based on race/gender (creature display ID)
	if p.DisplayID == 0 {
		p.DisplayID = p.defaultDisplayID()
	}

	if p.NativeDisplayID == 0 {
		p.NativeDisplayID = p.DisplayID
	}

	// Set default movement speeds
	p.Unit.Speed[wow.MoveTypeWalk] = 2.5
	p.Unit.Speed[wow.MoveTypeRun] = 7.0
	p.Unit.Speed[wow.MoveTypeRunBack] = 4.5
	p.Unit.Speed[wow.MoveTypeSwim] = 4.722222
	p.Unit.Speed[wow.MoveTypeSwimBack] = 2.5
	p.Unit.Speed[wow.MoveTypeFlight] = 7.0
	p.Unit.Speed[wow.MoveTypeFlightBack] = 4.5
	p.Unit.Speed[wow.MoveTypeTurnRate] = 7.0

	if p.Inventory == nil {
		p.Inventory = NewInventory()
	}

	// Initialize quest state maps
	if p.QuestStatus == nil {
		p.QuestStatus = make(map[uint32]interface{})
	}

	if p.RewardedQuests == nil {
		p.RewardedQuests = make(map[uint32]bool)
	}

	// Initialize update field values array
	p.Object.InitValues(int(object.PlayerEnd))

	// Set Object fields
	p.Object.SetUInt32Value(object.ObjectFieldGuid, uint32(p.GUID()))
	p.Object.SetUInt32Value(object.ObjectFieldGuid+1, uint32(uint64(p.GUID())>>32))
	p.Object.SetUInt32Value(object.ObjectFieldType, uint32(wow.TypeIDPlayer))
	p.Object.SetFloatValue(object.ObjectFieldScaleX, 1.0)

	// Set Unit fields
	// UNIT_FIELD_BYTES_0: race | (class << 8) | (gender << 16) | (powerType << 24)
	raceClassGender := uint32(p.Race) | (uint32(p.Class) << 8) | (uint32(p.Gender) << 16)
	powerTypeVal := uint32(powerType) << 24
	p.Object.SetUInt32Value(object.UnitFieldBytes_0, raceClassGender|powerTypeVal)

	// Health / MaxHealth
	p.Object.SetUInt32Value(object.UnitFieldHealth, p.Health)
	p.Object.SetUInt32Value(object.UnitFieldMaxhealth, p.MaxHealth)

	// Power / MaxPower (only primary power type for now)
	for i := 0; i < wow.MaxPowerTypes; i++ {
		p.Object.SetUInt32Value(object.UpdateField(int(object.UnitFieldPower1)+i), p.Power[i])
		p.Object.SetUInt32Value(object.UpdateField(int(object.UnitFieldMaxpower1)+i), p.MaxPower[i])
	}

	// Level
	p.Object.SetUInt32Value(object.UnitFieldLevel, uint32(p.Level))

	// Faction template from ChrRaces.dbc
	if p.FactionID == 0 {
		p.FactionID = FactionTemplateFromRace(p.Race)
	}
	p.Object.SetUInt32Value(object.UnitFieldFactiontemplate, p.FactionID)

	// Display ID
	p.Object.SetUInt32Value(object.UnitFieldDisplayid, p.DisplayID)
	p.Object.SetUInt32Value(object.UnitFieldNativedisplayid, p.NativeDisplayID)

	// Mount display (0 = not mounted)
	p.Object.SetUInt32Value(object.UnitFieldMountdisplayid, 0)

	// Unit flags: UNIT_FLAG_PVP_ATTACKABLE (0x08)
	p.Object.SetUInt32Value(object.UnitFieldFlags, 0x08)

	// Bounding radius / combat reach
	p.Object.SetFloatValue(object.UnitFieldBoundingradius, 0.388999998569489)
	p.Object.SetFloatValue(object.UnitFieldCombatreach, 1.5)

	// Base attack time (2000ms)
	p.Object.SetUInt32Value(object.UnitFieldBaseattacktime, 2000)
	p.Object.SetUInt32Value(object.UnitFieldBaseattacktime+1, 0)

	// Mod cast speed (1.0)
	p.Object.SetFloatValue(object.UnitModCastSpeed, 1.0)

	// UNIT_FIELD_BYTES_2: (0x02 << 24) = sanctuary flag
	p.Object.SetUInt32Value(object.UnitFieldBytes_2, 0x02000000)

	// Player fields
	p.Object.SetUInt32Value(object.PlayerBytes,
		uint32(p.Skin)|(uint32(p.Face)<<8)|(uint32(p.HairStyle)<<16)|(uint32(p.HairColor)<<24))
	p.Object.SetUInt32Value(object.PlayerBytes_2,
		uint32(p.FacialHair)|(0x00<<8)|(0x00<<16)|(0x02<<24))
	p.Object.SetUInt32Value(object.PlayerBytes_3, uint32(p.Gender))

	// Guild
	p.Object.SetUInt32Value(object.PlayerGuildid, 0)
	p.Object.SetUInt32Value(object.PlayerGuildrank, 0)

	// Money
	p.Object.SetUInt32Value(object.PlayerFieldCoinage, p.Money)

	// Watched faction index (-1 = none)
	p.Object.SetInt32Value(object.PlayerFieldWatchedFactionIndex, -1)

	// Base auto-attack damage (placeholder)
	p.BaseDamage = 5.0
}

// primaryPowerType returns the primary power type for this class.
func (p *Player) primaryPowerType() wow.PowerType {
	switch p.Class {
	case wow.ClassWarior:
		return wow.PowerTypeRage
	case wow.ClassPaladin, wow.ClassPriest, wow.ClassShaman,
		wow.ClassMage, wow.ClassWarlock, wow.ClassDruid:
		return wow.PowerTypeMana
	case wow.ClassHunter:
		return wow.PowerTypeFocus
	case wow.ClassRogue, wow.ClassDeathKnight:
		return wow.PowerTypeEnergy
	default:
		return wow.PowerTypeMana
	}
}

// defaultDisplayID returns a default creature display ID for the race/gender combo.
// These are placeholder values — real data should come from CreatureDisplayInfo.dbc.
func (p *Player) defaultDisplayID() uint32 {
	switch p.Race {
	case wow.RaceHuman:
		if p.Gender == wow.GenderMale {
			return 49
		}
		return 50
	case wow.RaceOrc:
		if p.Gender == wow.GenderMale {
			return 51
		}
		return 52
	case wow.RaceDwarf:
		if p.Gender == wow.GenderMale {
			return 53
		}
		return 54
	case wow.RaceNightElf:
		if p.Gender == wow.GenderMale {
			return 55
		}
		return 56
	case wow.RaceUndead:
		if p.Gender == wow.GenderMale {
			return 57
		}
		return 58
	case wow.RaceTauren:
		if p.Gender == wow.GenderMale {
			return 59
		}
		return 60
	case wow.RaceGnome:
		if p.Gender == wow.GenderMale {
			return 1563
		}
		return 1564
	case wow.RaceTroll:
		if p.Gender == wow.GenderMale {
			return 1478
		}
		return 1479
	case wow.RaceGoblin:
		if p.Gender == wow.GenderMale {
			return 1563
		}
		return 1564
	case wow.RaceBloodElf:
		if p.Gender == wow.GenderMale {
			return 15553
		}
		return 15554
	case wow.RaceDraenei:
		if p.Gender == wow.GenderMale {
			return 16125
		}
		return 16126
	default:
		return 49
	}
}

func (p *Player) Transport() *object.Transport {
	return nil
}

func (p *Player) BuildCreateUpdateForPlayer(target *Player) {
	updatetype := wow.UpdateTypeCreateObject
	flags := p.Object.UpdateFlags()

	_ = updatetype

	/** lower flag1 **/
	if target != nil { // building packet for oneself
		flags |= wow.UpdateFlagSelf
	}

	//nolint:revive,staticcheck
	if flags&wow.UpdateFlagStationaryPosition != 0 {
		// UPDATETYPE_CREATE_OBJECT2 dynamic objects, corpses...
		// if isType(TYPEMASK_DYNAMICOBJECT) || isType(TYPEMASK_CORPSE) || isType(TYPEMASK_PLAYER) {
		// 	updatetype = wow.UpdateTypeCreateObject2
		// }

		// UPDATETYPE_CREATE_OBJECT2 for pets...
		// if target.GetPetGUID() == p.GetGUID() {
		//     updatetype = UPDATETYPE_CREATE_OBJECT2
		// }

		// UPDATETYPE_CREATE_OBJECT2 for some gameobject types...
		// if (isType(TYPEMASK_GAMEOBJECT))
		// {
		//     switch (((GameObject*)this)->GetGoType())
		//     {
		//     case GAMEOBJECT_TYPE_TRAP:
		//     case GAMEOBJECT_TYPE_DUEL_ARBITER:
		//     case GAMEOBJECT_TYPE_FLAGSTAND:
		//     case GAMEOBJECT_TYPE_FLAGDROP:
		//         updatetype = UPDATETYPE_CREATE_OBJECT2;
		//         break;
		//     case GAMEOBJECT_TYPE_TRANSPORT:
		//         flags |= UPDATEFLAG_TRANSPORT;
		//         break;
		//     default:
		//         break;
		//     }
		// }
	}
}

//nolint:errcheck
func (p *Player) ToCharacterEnum(w *wow.Packet) {
	w.Write(p.GUID())
	w.WriteString(p.Name)
	w.Write(p.Race)
	w.Write(p.Class)
	w.Write(p.Gender)

	w.Write(p.Skin)
	w.Write(p.Face)
	w.Write(p.HairStyle)
	w.Write(p.HairColor)
	w.Write(p.FacialHair)

	w.Write(p.Level)

	w.Write(p.Location.Zone)
	w.Write(p.Location.Map)

	w.Write(p.Location.X)
	w.Write(p.Location.Y)
	w.Write(p.Location.Z)

	w.Write(p.GuildID)

	// Character flags
	w.Write(p.CharFlags)
	w.Write(p.Recustomization)

	// First login
	// *data << uint8(atLoginFlags & AT_LOGIN_FIRST ? 1 : 0);
	w.Write(p.FirstLogin)

	// Player Pet section
	w.Write(p.Pet.DisplayID)
	w.Write(p.Pet.PetLevel)
	w.Write(p.Pet.PetFamilly)

	// Inventory display items
	p.Inventory.ToCharacterEnum(w)
}

// GetHealth returns the current health of the player.
func (p *Player) GetHealth() uint32 {
	return p.Health
}

// GetLevel returns the player's level as uint32.
func (p *Player) GetLevel() uint32 {
	return uint32(p.Level)
}

// SetHealth sets the current health, clamping to [0, MaxHealth].
func (p *Player) SetHealth(v uint32) {
	if v > p.MaxHealth {
		v = p.MaxHealth
	}

	p.Health = v
}

// GetMaxHealth returns the maximum health of the player.
func (p *Player) GetMaxHealth() uint32 {
	return p.MaxHealth
}

// SetMaxHealth sets the maximum health and clamps current health.
func (p *Player) SetMaxHealth(v uint32) {
	p.MaxHealth = v

	if p.Health > v {
		p.Health = v
	}
}

// GetPower returns the current power for the given power type.
func (p *Player) GetPower(pt wow.PowerType) uint32 {
	if int(pt) < 0 || int(pt) >= wow.MaxPowerTypes {
		return 0
	}

	return p.Power[pt]
}

// SetPower sets the current power for the given power type, clamping to max.
func (p *Player) SetPower(pt wow.PowerType, v uint32) {
	if int(pt) < 0 || int(pt) >= wow.MaxPowerTypes {
		return
	}

	if v > p.MaxPower[pt] {
		v = p.MaxPower[pt]
	}

	p.Power[pt] = v
}

// GetMaxPower returns the maximum power for the given power type.
func (p *Player) GetMaxPower(pt wow.PowerType) uint32 {
	if int(pt) < 0 || int(pt) >= wow.MaxPowerTypes {
		return 0
	}

	return p.MaxPower[pt]
}

// SetMaxPower sets the maximum power for the given power type and clamps current.
func (p *Player) SetMaxPower(pt wow.PowerType, v uint32) {
	if int(pt) < 0 || int(pt) >= wow.MaxPowerTypes {
		return
	}

	p.MaxPower[pt] = v

	if p.Power[pt] > v {
		p.Power[pt] = v
	}
}

// GetPrimaryPowerType returns the primary power type for this player's class.
func (p *Player) GetPrimaryPowerType() wow.PowerType {
	return p.primaryPowerType()
}

// IsAlive returns true if the player has more than 0 health.
func (p *Player) IsAlive() bool {
	return p.Health > 0
}

// IsDead returns true if the player has 0 health.
func (p *Player) IsDead() bool {
	return p.Health == 0
}

// --- CombatUnit interface implementation ---

// IsPlayer returns true (player is always a player).
func (p *Player) IsPlayer() bool { return true }

// IsCreature returns false for players.
func (p *Player) IsCreature() bool { return false }

// IsPet returns false for players.
func (p *Player) IsPet() bool { return false }

// IsTotem returns false for players.
func (p *Player) IsTotem() bool { return false }

// GetPositionX returns the player's X coordinate.
func (p *Player) GetPositionX() float32 { return p.Location.X }

// GetPositionY returns the player's Y coordinate.
func (p *Player) GetPositionY() float32 { return p.Location.Y }

// GetPositionZ returns the player's Z coordinate.
func (p *Player) GetPositionZ() float32 { return p.Location.Z }

// GetAttackPower returns the player's melee attack power.
func (p *Player) GetAttackPower() uint32 { return p.AttackPower }

// GetWeaponDamage returns (min, max) damage for the given attack type.
// attackType: 0=BaseAttack, 1=OffAttack, 2=RangedAttack
func (p *Player) GetWeaponDamage(attackType uint8) (uint32, uint32) {
	switch attackType {
	case 1: // OffAttack
		return p.OffHandDamageMin, p.OffHandDamageMax
	default: // BaseAttack
		return p.MainHandDamageMin, p.MainHandDamageMax
	}
}

// GetWeaponSpeed returns the base attack speed for the given attack type.
func (p *Player) GetWeaponSpeed(_ uint8) time.Duration {
	if p.BaseAttackSpeed > 0 {
		return p.BaseAttackSpeed
	}
	return 2000 * time.Millisecond // default 2.0s
}

// GetCombatRating returns the player's combat rating value.
func (p *Player) GetCombatRating(_ uint8) uint32 {
	// TODO: load from player ratings table
	return 0
}

// GetArmor returns the player's total armor value.
func (p *Player) GetArmor() uint32 { return p.Armor }

// GetBlockChance returns the player's block chance (0-100).
func (p *Player) GetBlockChance() float32 {
	// TODO: calculate from block skill + rating
	return 0
}

// GetDodgeChance returns the player's dodge chance (0-100).
func (p *Player) GetDodgeChance() float32 {
	// TODO: calculate from agility + defense skill
	return 0
}

// GetParryChance returns the player's parry chance (0-100).
func (p *Player) GetParryChance() float32 {
	// TODO: calculate from defense skill + rating
	return 0
}

// GetCritChance returns the player's melee crit chance (0-100).
func (p *Player) GetCritChance() float32 {
	// TODO: calculate from agility + rating
	return 5.0 // base 5% crit
}

// GetVictim returns the current combat target.
func (p *Player) GetVictim() interface{} { return p.Victim }

// SetVictim sets the current combat target.
func (p *Player) SetVictim(v interface{}) { p.Victim = v }

// AddThreat adds threat from an attacker (placeholder for future threat system).
func (p *Player) AddThreat(_ interface{}, _ float32) {}

// CombatStop stops all combat actions.
func (p *Player) CombatStop() {
	p.AttackState = 0
	p.AttackTarget = 0
	p.Victim = nil
}

// OnDamageTaken is called when this player takes damage.
func (p *Player) OnDamageTaken(_ interface{}, _ uint32) {}

// OnDamageDealt is called when this player deals damage.
func (p *Player) OnDamageDealt(_ interface{}, _ uint32) {}

// GetDuelInfo returns the player's duel info.
func (p *Player) GetDuelInfo() interface{} { return p.Duel }

// SetDuelInfo sets the player's duel info.
func (p *Player) SetDuelInfo(info interface{}) { p.Duel = info }

// IsMounted returns true if the player is mounted.
func (p *Player) IsMounted() bool { return p.Mounted }

// IsInSameMap returns true if the other unit is on the same map.
func (p *Player) IsInSameMap(other interface{}) bool {
	if other == nil {
		return false
	}
	type mapGetter interface {
		GetMapID() uint32
	}
	if mg, ok := other.(mapGetter); ok {
		return p.CurrentMapID == mg.GetMapID()
	}
	return false
}

// IsHostileTo returns true if the player is hostile to the other unit.
func (p *Player) IsHostileTo(other interface{}) bool {
	if other == nil {
		return false
	}
	// Duel opponents are always hostile
	if p.Duel != nil {
		if di, ok := p.Duel.(*DuelInfo); ok && di.Opponent != nil {
			type guidGetter interface {
				GetGUID() wow.GUID
			}
			if gg, ok := other.(guidGetter); ok {
				if di.Opponent.GUID() == gg.GetGUID() {
					return true
				}
			}
		}
	}
	// TODO: proper faction template system
	return false
}

// IsFriendlyTo returns true if the player is friendly to the other unit.
func (p *Player) IsFriendlyTo(other interface{}) bool {
	return !p.IsHostileTo(other)
}

// IsValidAttackTarget checks if this player can attack the target.
func (p *Player) IsValidAttackTarget(target interface{}) bool {
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
	if p.GUID() == gg.GetGUID() {
		return false
	}

	// Can't attack dead targets
	if !ac.IsAlive() {
		return false
	}

	// Can't attack while mounted
	if p.Mounted {
		return false
	}

	// Can't attack in sanctuary (PvP)
	if p.IsSanctuary {
		return false
	}

	// Duel exception — always allow attacking duel opponent
	if p.Duel != nil {
		if di, ok := p.Duel.(*DuelInfo); ok && di.Opponent != nil {
			if di.Opponent.GUID() == gg.GetGUID() && di.State == DuelStateInProgress {
				return true
			}
		}
	}

	// PvP checks
	if p.IsPVP {
		if pvper, ok := target.(interface{ IsPvP() bool }); ok && pvper.IsPvP() {
			return true
		}
	}

	// TODO: faction reputation checks
	return !p.IsFriendlyTo(target)
}

// AddAttacker adds a GUID to the attacker set.
func (p *Player) AddAttacker(guid wow.GUID) {
	if p.Attackers == nil {
		p.Attackers = make(map[uint64]bool)
	}
	p.Attackers[uint64(guid)] = true
}

// RemoveAttacker removes a GUID from the attacker set.
func (p *Player) RemoveAttacker(guid wow.GUID) {
	if p.Attackers != nil {
		delete(p.Attackers, uint64(guid))
	}
}

// HasAttacker returns true if the given GUID is in the attacker set.
func (p *Player) HasAttacker(guid wow.GUID) bool {
	if p.Attackers == nil {
		return false
	}
	return p.Attackers[uint64(guid)]
}

// GetAttackers returns the attacker set.
func (p *Player) GetAttackers() map[uint64]bool {
	return p.Attackers
}

// SetInCombat puts the player into combat state.
func (p *Player) SetInCombat() {
	p.InCombat = true
	p.Object.SetFlag(object.UnitFieldFlags, UnitFlagInCombat)
	// TODO: set combat timer (5s rule)
}

// ClearInCombat removes the player from combat state.
func (p *Player) ClearInCombat() {
	p.InCombat = false
	p.Attackers = nil
	p.Object.RemoveFlag(object.UnitFieldFlags, UnitFlagInCombat)
}

// IsInCombatState returns true if the player is in combat.
func (p *Player) IsInCombatState() bool {
	return p.InCombat
}

// Attack initiates combat against a target.
// The actual implementation is in the world package to avoid circular imports.
// This is a placeholder that will be called by the world package.
func (p *Player) Attack(_ interface{}, _ bool) {}

// AttackStop stops the current attack.
// The actual implementation is in the world package to avoid circular imports.
func (p *Player) AttackStop() {
	p.AttackTarget = 0
	p.AttackState = AttackStateIdle
	p.NextAttackTime = 0
	p.Victim = nil
}

// KnowsSpell returns true if the player knows the given spell.
func (p *Player) KnowsSpell(spellID uint32) bool {
	for _, id := range p.KnownSpells {
		if id == spellID {
			return true
		}
	}

	return false
}

// LearnSpell adds a spell to the player's known spells.
func (p *Player) LearnSpell(spellID uint32) {
	if !p.KnowsSpell(spellID) {
		p.KnownSpells = append(p.KnownSpells, spellID)
	}
}

// ForgetSpell removes a spell from the player's known spells.
func (p *Player) ForgetSpell(spellID uint32) {
	for i, id := range p.KnownSpells {
		if id == spellID {
			p.KnownSpells = append(p.KnownSpells[:i], p.KnownSpells[i+1:]...)

			return
		}
	}
}

// HasPowerForSpell checks if the player has enough power for a spell.
// powerType: 0=mana, 1=rage, 2=focus, 3=energy
// powerCost: amount of power needed
func (p *Player) HasPowerForSpell(powerType int, powerCost uint32) bool {
	if powerType < 0 || powerType >= wow.MaxPowerTypes {
		return false
	}

	return p.Power[powerType] >= powerCost
}

// ConsumePowerForSpell reduces power for a spell cast.
func (p *Player) ConsumePowerForSpell(powerType int, powerCost uint32) {
	if powerType < 0 || powerType >= wow.MaxPowerTypes {
		return
	}

	if p.Power[powerType] >= powerCost {
		p.Power[powerType] -= powerCost
	} else {
		p.Power[powerType] = 0
	}
}

// IsSpellOnCooldown checks if a spell is on cooldown.
func (p *Player) IsSpellOnCooldown(spellID uint32) bool {
	if p.SpellCooldowns == nil {
		return false
	}

	expiry, ok := p.SpellCooldowns[spellID]
	if !ok {
		return false
	}

	return time.Now().Before(expiry)
}

// AddCooldown sets a cooldown for a spell.
func (p *Player) AddCooldown(spellID uint32, durationMs int64) {
	if p.SpellCooldowns == nil {
		p.SpellCooldowns = make(map[uint32]time.Time)
	}

	p.SpellCooldowns[spellID] = time.Now().Add(time.Duration(durationMs) * time.Millisecond)
}

// IsWithinRange checks if the player is within range of a target.
func (p *Player) IsWithinRange(target *Player, maxRange float32) bool {
	if maxRange <= 0 {
		return true // melee or self
	}

	dx := p.Location.X - target.Location.X
	dy := p.Location.Y - target.Location.Y
	dz := p.Location.Z - target.Location.Z

	dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

	return dist <= maxRange
}

// XpToNextLevel returns the XP required for the next level.
// Uses the standard WotLK formula: (level * 1000) + (level^2 * 100)
func (p *Player) XpToNextLevel() uint32 {
	lvl := uint32(p.Level)
	return lvl*1000 + lvl*lvl*100
}

// GainXP grants experience points and checks for level up.
func (p *Player) GainXP(amount uint32) uint32 {
	if p.Level >= 80 {
		return 0 // max level
	}

	p.XP += amount

	// Check for level up
	levelsGained := uint32(0)
	for p.XP >= p.XpToNextLevel() && p.Level < 80 {
		p.XP -= p.XpToNextLevel()
		p.Level++
		levelsGained++

		// Apply stat gains per level
		p.applyLevelUpGains()
	}

	return levelsGained
}

// applyLevelUpGains applies stat increases when leveling up.
func (p *Player) applyLevelUpGains() {
	// Health gain per level (varies by class, simplified)
	var healthGain uint32
	switch p.Class {
	case wow.ClassWarior:
		healthGain = 20
	case wow.ClassPaladin, wow.ClassDruid:
		healthGain = 18
	case wow.ClassHunter, wow.ClassShaman:
		healthGain = 16
	case wow.ClassRogue, wow.ClassDeathKnight:
		healthGain = 14
	case wow.ClassPriest, wow.ClassMage, wow.ClassWarlock:
		healthGain = 12
	default:
		healthGain = 14
	}

	p.MaxHealth += healthGain
	p.Health = p.MaxHealth // Full heal on level up

	// Power gain per level (for mana users)
	powerType := p.primaryPowerType()
	if powerType == wow.PowerTypeMana {
		p.MaxPower[powerType] += 12
		p.Power[powerType] = p.MaxPower[powerType]
	}

	// Update update fields
	p.Object.SetUInt32Value(object.UnitFieldLevel, uint32(p.Level))
	p.Object.SetUInt32Value(object.UnitFieldMaxhealth, p.MaxHealth)
	p.Object.SetUInt32Value(object.UnitFieldHealth, p.Health)

	if powerType >= 0 && int(powerType) < wow.MaxPowerTypes {
		p.Object.SetUInt32Value(object.UpdateField(int(object.UnitFieldPower1)+int(powerType)), p.Power[powerType])
		p.Object.SetUInt32Value(object.UpdateField(int(object.UnitFieldMaxpower1)+int(powerType)), p.MaxPower[powerType])
	}
}

// Kill grants XP for killing a creature based on level difference.
func (p *Player) Kill(creatureLevel uint8) uint32 {
	// Base XP from creature level
	baseXP := uint32(creatureLevel) * 10

	// Level difference modifier
	diff := int32(p.Level) - int32(creatureLevel)
	var modifier float32
	switch {
	case diff <= -5:
		modifier = 2.0 // much higher level creature
	case diff <= -2:
		modifier = 1.5
	case diff <= 2:
		modifier = 1.0 // same level
	case diff <= 5:
		modifier = 0.8
	default:
		modifier = 0.5 // much lower level creature
	}

	xp := uint32(float32(baseXP) * modifier)
	if xp < 1 {
		xp = 1
	}

	return p.GainXP(xp)
}

// Die kills the player and sets ghost state.
func (p *Player) Die() {
	p.Health = 0
	p.IsGhost = true

	// Clear attack state
	p.AttackState = 0
	p.AttackTarget = 0

	// Update health in update fields
	p.Object.SetUInt32Value(object.UnitFieldHealth, 0)

	// Set ghost flag in CharFlags
	// CHAR_FLAG_GHOST = 0x00000010
	p.CharFlags |= 0x10
}

// Resurrect复活 the player with a percentage of health/mana.
func (p *Player) Resurrect() {
	p.IsGhost = false

	// Resurrect with 100% health
	p.Health = p.MaxHealth

	// Restore some power
	powerType := p.primaryPowerType()
	if powerType >= 0 && int(powerType) < wow.MaxPowerTypes {
		p.Power[powerType] = p.MaxPower[powerType]
	}

	// Clear ghost flag
	p.CharFlags &^= 0x10

	// Update update fields
	p.Object.SetUInt32Value(object.UnitFieldHealth, p.Health)

	if powerType >= 0 && int(powerType) < wow.MaxPowerTypes {
		p.Object.SetUInt32Value(object.UpdateField(int(object.UnitFieldPower1)+int(powerType)), p.Power[powerType])
	}
}

// GetMap returns the current map (interface to avoid circular import).
func (p *Player) GetMap() interface{} {
	return p.CurrentMap
}

// SetMap sets the current map.
func (p *Player) SetMap(m interface{}) {
	p.CurrentMap = m
}

// GetMapID returns the current map ID.
func (p *Player) GetMapID() uint32 {
	return p.CurrentMapID
}

// SetMapID sets the current map ID.
func (p *Player) SetMapID(mapID uint32) {
	p.CurrentMapID = mapID
}

// GetInstanceID returns the current instance ID.
func (p *Player) GetInstanceID() uint32 {
	return p.CurrentInstanceID
}

// SetInstanceID sets the current instance ID.
func (p *Player) SetInstanceID(instanceID uint32) {
	p.CurrentInstanceID = instanceID
}

// TeleportTo teleports the player to the given location.
func (p *Player) TeleportTo(mapID uint32, x, y, z, o float32) {
	p.Location.Map = mapID
	p.Location.X = x
	p.Location.Y = y
	p.Location.Z = z
	p.Location.O = o

	p.CurrentMapID = mapID
}

// UpdateInventoryFields updates the player's inventory update fields.
// This sets PlayerFieldInvSlotHead and PlayerFieldPackSlot_1 with item GUIDs.
func (p *Player) UpdateInventoryFields() {
	if p.Inventory == nil {
		return
	}

	// Set equipment and bag slot GUIDs (PlayerFieldInvSlotHead covers slots 0-22)
	// Each slot takes 2 uint32 values (low + high GUID)
	for i := 0; i < InventorySlotTotal && i < 23; i++ {
		item := p.Inventory.GetItem(i)
		slotField := object.UpdateField(int(object.PlayerFieldInvSlotHead) + i*2)

		if item != nil {
			guid := item.GUID()
			p.Object.SetUInt32Value(slotField, uint32(guid))
			p.Object.SetUInt32Value(slotField+1, uint32(uint64(guid)>>32))
		} else {
			p.Object.SetUInt32Value(slotField, 0)
			p.Object.SetUInt32Value(slotField+1, 0)
		}
	}

	// Set backpack slot GUIDs (PlayerFieldPackSlot_1 covers slots 27-38, 12 slots)
	for i := 0; i < 12; i++ {
		absSlot := InventorySlotItemStart + i
		item := p.Inventory.GetItem(absSlot)
		slotField := object.UpdateField(int(object.PlayerFieldPackSlot_1) + i*2)

		if item != nil {
			guid := item.GUID()
			p.Object.SetUInt32Value(slotField, uint32(guid))
			p.Object.SetUInt32Value(slotField+1, uint32(uint64(guid)>>32))
		} else {
			p.Object.SetUInt32Value(slotField, 0)
			p.Object.SetUInt32Value(slotField+1, 0)
		}
	}
}

// EquipItem equips an item to the appropriate slot.
// Returns the previously equipped item if the slot was occupied, or nil.
func (p *Player) EquipItem(item *Item) *Item {
	if item == nil || p.Inventory == nil {
		return nil
	}

	// Determine the equipment slot based on item's inventory type
	inventoryType := GetItemInventoryType(item.ItemEntry)
	slot := FindEquipSlot(inventoryType)
	if slot < 0 {
		return nil
	}

	// Get existing item in that slot
	prev := p.Inventory.SetEquipment(slot, item)

	// Update item's owner and contained
	item.Owner = p.GUID()
	item.Contained = p.GUID()
	item.SlotIndex = slot

	// Update item update fields
	item.initUpdateFields()

	// Update player's inventory update fields
	p.UpdateInventoryFields()

	return prev
}

// UnequipItem removes an item from an equipment slot.
func (p *Player) UnequipItem(slot int) *Item {
	if p.Inventory == nil || slot < 0 || slot >= EquipmentSlotEnd {
		return nil
	}

	item := p.Inventory.RemoveItem(slot)
	if item != nil {
		// Update player's inventory update fields
		p.UpdateInventoryFields()
	}

	return item
}

// GetItemInventoryType returns the inventory type for an item entry: from
// the item templates of the base data, with a range-based fallback for the
// test items when they are not loaded.
func GetItemInventoryType(entry uint32) wow.InventoryType {
	if tpl := basedata.GetInstance().LookupItem(entry); tpl != nil {
		return tpl.InventoryType
	}

	switch {
	case entry >= 25000 && entry < 25010:
		return wow.InventoryTypeHead
	case entry >= 25010 && entry < 25020:
		return wow.InventoryTypeShoulders
	case entry >= 25020 && entry < 25030:
		return wow.InventoryTypeChest
	case entry >= 25030 && entry < 25040:
		return wow.InventoryTypeLegs
	case entry >= 25040 && entry < 25050:
		return wow.InventoryTypeFeet
	case entry >= 25050 && entry < 25060:
		return wow.InventoryTypeWeaponMainHand
	case entry >= 25060 && entry < 25070:
		return wow.InventoryTypeWeaponOffHand
	case entry >= 25070 && entry < 25080:
		return wow.InventoryTypeRanged
	default:
		return wow.InventoryType(0)
	}
}

// ModifyMoney adds (or subtracts if negative) money to the player's coinage.
func (p *Player) ModifyMoney(amount int32) {
	newMoney := int64(p.Money) + int64(amount)
	if newMoney < 0 {
		newMoney = 0
	}

	// Cap at 2^32 - 1 (max uint32)
	if newMoney > 0xFFFFFFFF {
		newMoney = 0xFFFFFFFF
	}

	p.Money = uint32(newMoney)
	p.Object.SetUInt32Value(object.PlayerFieldCoinage, p.Money)
}
