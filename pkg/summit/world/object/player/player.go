package player

import (
	"math"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/wow"
)

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

	Level  uint8
	XP     uint32
	Money  uint32

	Health    uint32
	MaxHealth uint32
	Power     [wow.MaxPowerTypes]uint32
	MaxPower  [wow.MaxPowerTypes]uint32

	DisplayID        uint32
	NativeDisplayID  uint32
	MountDisplayID   uint32

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

	// Action bar (12 buttons per bar, 3 bars + stance bar)
	Actions [48]uint32
}

// Initializes the inventory. The slots can be nil, in this case it will be initialized as
// an empty inventory.
func (p *Player) InitInventory(slots []*basedata.InventorySlot) {
	if p.Inventory != nil {
		return
	}

	p.Inventory = NewInventory()

	if slots == nil {
		return
	}

	for i, slot := range slots {
		if i >= EquipmentSlotEnd {
			continue
		}

		if slot.ItemID <= 0 {
			continue
		}

		item := NewItem(uint32(slot.ItemID), p.GUID())
		item.SetEnchant(0, 0)
		p.Inventory.SetEquipment(i, item)
	}
}

func (p *Player) GUID() wow.GUID {
	return wow.NewPlayerGUID(p.ID)
}

func (p *Player) Init() {
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
	if flags&wow.UpdateFlagHasPosition != 0 {
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
