# Character Creation — Design & Protocol Reference

This document describes the WoW 3.3.5a (WotLK) character creation flow as
implemented by AzerothCore, then specifies how Summit should implement it with
MongoDB persistence.

**Database**: A MySQL instance with the AzerothCore world database is running
on `localhost:3307` (Docker container `ac-mysql`, user `root`, password
`ac_password`, database `world`). Use this to inspect real data.

---

## Table of Contents

1. [Packet Flow](#1-packet-flow)
2. [CMSG_CHAR_CREATE — Client Request](#2-cmsg_char_create--client-request)
3. [Server-Side Validation](#3-server-side-validation)
4. [Character Object Initialization (Player::Create)](#4-character-object-initialization-playercreate)
5. [Starting Items](#5-starting-items)
6. [SMSG_CHAR_CREATE — Server Response](#6-smsg_char_create--server-response)
7. [SMSG_CHAR_ENUM — Character List](#7-smsg_char_enum--character-list)
8. [Data Sources](#8-data-sources)
9. [AzerothCore World Database Reference](#9-azerothcore-world-database-reference)
10. [Summit MongoDB Storage Design](#10-summit-mongodb-storage-design)
11. [Implementation Status](#11-implementation-status)

---

## 1. Packet Flow

```
Client                          Server
  |                                |
  |--- CMSG_CHAR_CREATE (0x036) -->|  Create character
  |                                |
  |    [validate name, race,       |
  |     class, expansion, limits]  |
  |                                |
  |    [create Player object]      |
  |    [assign starting items]     |
  |    [save to database]          |
  |                                |
  |<-- SMSG_CHAR_CREATE (0x03a) ---|  Result code
  |                                |
  |--- CMSG_CHAR_ENUM (0x037) --->|  Refresh character list
  |<-- SMSG_CHAR_ENUM (0x03b) ----|  Updated list
```

### Opcodes

| Opcode | Direction | Name | Purpose |
|--------|-----------|------|---------|
| `0x036` | Client → Server | `CMSG_CHAR_CREATE` | Request character creation |
| `0x03a` | Server → Client | `SMSG_CHAR_CREATE` | Creation result code |
| `0x037` | Client → Server | `CMSG_CHAR_ENUM` | Request character list |
| `0x03b` | Server → Client | `SMSG_CHAR_ENUM` | Character list with equipment |

---

## 2. CMSG_CHAR_CREATE — Client Request

The client sends this packet when the player clicks "Accept" on the character
creation screen.

### Packet Layout (little-endian)

| Offset | Type | Field | Description |
|--------|------|-------|-------------|
| 0 | string (null-terminated) | Name | Character name |
| var | uint8 | Race | `PlayerRace` enum |
| var | uint8 | Class | `PlayerClass` enum |
| var | uint8 | Gender | 0 = Male, 1 = Female |
| var | uint8 | Skin | Skin color index |
| var | uint8 | Face | Face shape index |
| var | uint8 | HairStyle | Hair style index |
| var | uint8 | HairColor | Hair color index |
| var | uint8 | FacialHair | Facial hair style index |
| var | uint8 | OutfitID | Starting outfit selection (0 = default) |

### Go Struct (Summit)

```go
// pkg/summit/world/character.go
type CharacterCreateRequest struct {
    Race       wow.PlayerRace
    Class      wow.PlayerClass
    Gender     wow.PlayerGender
    Skin       uint8
    Face       uint8
    HairStyle  uint8
    HairColor  uint8
    FacialHair uint8
    OutfitID   uint8
}
```

---

## 3. Server-Side Validation

Before creating the character, the server validates multiple conditions.
Failures send `SMSG_CHAR_CREATE` with the appropriate error code and abort.

### Validation Steps (in order)

1. **Faction disabled** — Check `CONFIG_CHARACTER_CREATING_DISABLED` mask for
   the player's team (Alliance/Horde).

2. **Race/Class DBC lookup** — Verify the race and class IDs exist in
   `ChrRaces.dbc` / `ChrClasses.dbc`.

3. **Expansion check** — Race and class must not require a higher expansion than
   the account owns.

4. **Race mask disabled** — Check `CONFIG_CHARACTER_CREATING_DISABLED_RACEMASK`.

5. **Class mask disabled** — Check `CONFIG_CHARACTER_CREATING_DISABLED_CLASSMASK`.

6. **Name validation** — `normalizePlayerName()` + `ObjectMgr::CheckPlayerName()`
   enforces length, allowed characters, reserved names.

7. **Death Knight restrictions** — If class is DK:
   - Server must allow DK creation (`CONFIG_HEROIC_CHARACTERS_PER_REALM > 0`)
   - Account must have a character at the required level
     (`CONFIG_CHARACTER_CREATING_MIN_LEVEL_FOR_HEROIC_CHARACTER`)
   - Account must not exceed the DK slot limit per realm

8. **Name uniqueness** — Query `characters` table + character cache.

9. **Account character limit** — `CONFIG_CHARACTERS_PER_ACCOUNT`

10. **Realm character limit** — `CONFIG_CHARACTERS_PER_REALM`

11. **Faction lock (PvP realms)** — If `!CONFIG_ALLOW_TWO_SIDE_ACCOUNTS`, first
    character determines the account's faction; subsequent characters must match.

### Response Codes

```go
type CharacterCreateResult uint8

const (
    CharCreateSuccess              CharacterCreateResult = 0x00
    CharCreateFailed               CharacterCreateResult = 0x2E
    CharCreateNameInUse            CharacterCreateResult = 0x31
    CharCreateDisabled             CharacterCreateResult = 0x3A
    CharCreateNoName               CharacterCreateResult = 0x33
    CharCreateLevelRequirement     CharacterCreateResult = 0x3E // DK only
    CharCreateUniqueClassLimit     CharacterCreateResult = 0x3F // DK only
    CharCreateAccountLimit         CharacterCreateResult = 0x36
    CharCreateServerLimit          CharacterCreateResult = 0x37
    CharCreatePvpTeamsViolation    CharacterCreateResult = 0x3C
    CharCreateExpansion            CharacterCreateResult = 0x39 // race needs expansion
    CharCreateExpansionClass       CharacterCreateResult = 0x3C // class needs expansion
)
```

---

## 4. Character Object Initialization (Player::Create)

After validation, the server creates a `Player` object and populates it.

### 4.1 Starting Position

Loaded from `playercreateinfo` table (Summit: `basedata.PlayerCreateInfo`):

| Column | Field | Description |
|--------|-------|-------------|
| `map` | `Map` | Starting map (0 = Eastern Kingdoms, 1 = Kalimdor, 530 = Outland) |
| `zone` | `Zone` | Starting zone |
| `position_x` | `X` | X coordinate |
| `position_y` | `Y` | Y coordinate |
| `position_z` | `Z` | Z coordinate |
| `orientation` | `O` | Facing direction |

### 4.2 Unit Fields Set

```
UNIT_FIELD_BYTES_0       = race | (class << 8) | (gender << 16) | (powerType << 24)
UNIT_FIELD_LEVEL         = CONFIG_START_PLAYER_LEVEL (default: 1)
PLAYER_FIELD_COINAGE     = CONFIG_START_PLAYER_MONEY (default: 10c for Alliance, 25c for Horde)
PLAYER_FIELD_HONOR_CURRENCY = CONFIG_START_HONOR_POINTS
PLAYER_FIELD_ARENA_CURRENCY = CONFIG_START_ARENA_POINTS
```

### 4.3 Player Bytes (Appearance)

```
PLAYER_BYTES      = skin | (face << 8) | (hairStyle << 16) | (hairColor << 24)
PLAYER_BYTES_2    = facialHair | (0x00 << 8) | (0x00 << 16) | (restState << 24)
PLAYER_BYTES_3    = gender | (0 << 8) | (0 << 16) | (0 << 24)
```

### 4.4 Initial Stats

- `InitStatsForLevel()` — sets base strength, agility, stamina, intellect, spirit
- `InitTaxiNodesForLevel()` — unlocks flight paths
- `InitGlyphsForLevel()` — empty at level 1
- `InitTalentForLevel()` — 0 talent points at level 1
- `InitPrimaryProfessions()` — no professions at start

### 4.5 Power by Class

| Class | Power Type |
|-------|-----------|
| Warrior | Rage (0) |
| Paladin | Mana (0) |
| Hunter | Focus (2) |
| Rogue | Energy (3) |
| Priest | Mana (0) |
| Death Knight | Energy (3) |
| Shaman | Mana (0) |
| Mage | Mana (0) |
| Warlock | Mana (0) |
| Druid | Mana (0) |

### 4.6 Spells & Skills

1. `LearnDefaultSkills()` — race/class-appropriate skills from
   `playercreateinfo_skills` table
2. `LearnCustomSpells()` — spells from `playercreateinfo_spell_custom` table
3. Cast spells from `playercreateinfo_cast_spell` (aura-type spells applied on
   create)

### 4.7 Action Bar

Populated from `playercreateinfo_action` table:

```
button | action | type
```

Each entry becomes a button on the player's action bar.

---

## 5. Starting Items

Starting items come from two sources, processed in order:

### 5.1 CharStartOutfit DBC (Primary Source)

`CharStartOutfit.dbc` contains up to 24 item IDs per race/class/gender
combination. These define the visual outfit shown on the character creation
screen and become the character's starting equipment.

```
GetCharStartOutfitEntry(race, class, gender) → CharStartOutfitEntry
```

The outfit entry is a flat array of `ItemId[24]`. For each non-zero item:

1. Look up `ItemTemplate` (DBC item info)
2. Determine quantity:
   - **Food** (`ITEM_CLASS_CONSUMABLE`, `ITEM_SUBCLASS_FOOD`):
     - Spell category `SPELL_CATEGORY_FOOD`: 4 items (10 for DK)
     - Spell category `SPELL_CATEGORY_DRINK`: 2 items
     - Clamped to `MaxStack`
   - **All other items**: `BuyCount` (usually 1)
3. Call `StoreNewItemInBestSlots(itemId, count)`

### 5.2 Custom Items (DB Override)

From `playercreateinfo_item` table:

```
SELECT race, class, itemid, amount FROM playercreateinfo_item
```

- If `race=0` or `class=0`: applies to all races/classes (wildcard)
- If `amount > 0`: adds the item
- If `amount == -1`: removes the item from the DBC outfit list
- If `amount < -1`: invalid, logged as error

### 5.3 Collector's Edition Voucher

If `ACCOUNT_FLAG_COLLECTOR` is set, a race-specific gift voucher is added:

| Race | Voucher Item ID | Name |
|------|----------------|------|
| Human | 14646 | Goldshire Gift Voucher |
| Dwarf/Gnome | 14647 | Kharanos Gift Voucher |
| Night Elf | 14648 | Dolanaar Gift Voucher |
| Orc/Troll | 14649 | Razor Hill Gift Voucher |
| Tauren | 14650 | Bloodhoof Village Gift Voucher |
| Undead | 14651 | Brill Gift Voucher |
| Blood Elf | 20938 | Falconwing Square Gift Voucher |
| Draenei | 22888 | Azure Watch Gift Voucher |
| DK (any) | 39713 | Ebon Hold Gift Voucher |

### 5.4 Auto-Equip

After all items are placed in inventory, a second pass auto-equips items:

```
for each item in INVENTORY_SLOT_ITEM_START..INVENTORY_SLOT_ITEM_END:
    if CanEquipItem():
        RemoveItem → EquipItem
    else:
        CanStoreItem → move to bag
        CanUseAmmo → SetAmmo
```

---

## 6. SMSG_CHAR_CREATE — Server Response

Single byte response after character creation attempt.

```
Packet: SMSG_CHAR_CREATE (0x03a)
Body:   uint8 result
```

| Code | Name | Meaning |
|------|------|---------|
| 0x00 | `CHAR_CREATE_SUCCESS` | Character created successfully |
| 0x2E | `CHAR_CREATE_FAILED` | General failure (bad race/class) |
| 0x31 | `CHAR_CREATE_NAME_IN_USE` | Name already taken |
| 0x33 | `CHAR_CREATE_NO_NAME` | Empty or invalid name |
| 0x36 | `CHAR_CREATE_ACCOUNT_LIMIT` | Too many chars on account |
| 0x37 | `CHAR_CREATE_SERVER_LIMIT` | Too many chars on realm |
| 0x39 | `CHAR_CREATE_EXPANSION` | Race requires expansion |
| 0x3A | `CHAR_CREATE_DISABLED` | Race/class/faction disabled |
| 0x3C | `CHAR_CREATE_PVP_TEAMS_VIOLATION` | Faction mismatch on PvP realm |
| 0x3E | `CHAR_CREATE_LEVEL_REQUIREMENT` | DK level requirement not met |
| 0x3F | `CHAR_CREATE_UNIQUE_CLASS_LIMIT` | Too many DKs on realm |

---

## 7. SMSG_CHAR_ENUM — Character List

Sent after login and after character creation/deletion to refresh the list.

### Server-side Query

```sql
-- Standard
SELECT characters.guid, name, race, class, gender, skin, face,
       hairStyle, hairColor, facialStyle, level, zone, map,
       position_x, position_y, position_z,
       guild_member.guildid, playerFlags, at_login,
       character_pet.entry, character_pet.modelid, character_pet.level,
       equipmentCache, character_banned.guid, extra_flags
FROM characters
LEFT JOIN guild_member ON ...
LEFT JOIN character_pet ON ...
LEFT JOIN character_banned ON ...
WHERE account = ?
```

### Packet Layout (per character)

| Field | Type | Description |
|-------|------|-------------|
| GUID | uint64 | Player GUID |
| Name | string | Character name |
| Race | uint8 | Race ID |
| Class | uint8 | Class ID |
| Gender | uint8 | Gender (0=M, 1=F) |
| Skin | uint8 | Skin color |
| Face | uint8 | Face shape |
| HairStyle | uint8 | Hair style |
| HairColor | uint8 | Hair color |
| FacialHair | uint8 | Facial hair style |
| Level | uint8 | Character level |
| Zone | uint32 | Current zone |
| Map | uint32 | Current map |
| X | float32 | Position X |
| Y | float32 | Position Y |
| Z | float32 | Position Z |
| GuildID | uint32 | Guild GUID |
| CharFlags | uint32 | Character flags (ghost, rename, etc.) |
| CustomizeFlags | uint32 | Recustomization flags |
| FirstLogin | uint8 | 1 if first login (show cinematic) |
| PetDisplayID | uint32 | Pet model display ID |
| PetLevel | uint32 | Pet level |
| PetFamily | uint32 | Pet family (creature_template.family) |
| **Equipment** | | 19 slots × (displayID + inventoryType + enchantAura) |

### Equipment Slot Layout (per slot)

```
uint32 displayId      // ItemTemplate.DisplayInfoID
uint8  inventoryType  // ItemTemplate.InventoryType
uint32 enchantAura    // SpellItemEnchantmentEntry.aura_id (or 0)
```

### Character Flags

```go
const (
    CharacterFlagNone              = 0x00000000
    CharacterFlagResting           = 0x00000002
    CharacterFlagLockedForTransfer = 0x00000004
    CharacterFlagHideHelm          = 0x00000400
    CharacterFlagHideCloak         = 0x00000800
    CharacterFlagGhost             = 0x00002000
    CharacterFlagRename            = 0x00004000
    CharacterFlagLockedByBilling   = 0x01000000
    CharacterFlagDeclined          = 0x02000000
)
```

---

## 8. Data Sources

### DBC Files (Client Data)

| File | Used For |
|------|----------|
| `ChrRaces.dbc` | Race info, display IDs, expansion requirement |
| `ChrClasses.dbc` | Class info, power type, expansion requirement |
| `CharStartOutfit.dbc` | Starting equipment items per race/class/gender |
| `CreatureDisplayInfo.dbc` | Display ID for character model |

### Database Tables (World DB)

| Table | Purpose |
|-------|---------|
| `playercreateinfo` | Starting position per race/class |
| `playercreateinfo_item` | Custom starting items (add/remove) |
| `playercreateinfo_skills` | Starting skills per race/class |
| `playercreateinfo_spell_custom` | Custom starting spells |
| `playercreateinfo_cast_spell` | Spells cast on character creation |
| `playercreateinfo_action` | Starting action bar layout |
| `player_race_stats` | Race stat modifiers |
| `player_class_stats` | Class base stats per level |

---

## 9. AzerothCore World Database Reference

A MySQL instance is running at `localhost:3307` with the full AzerothCore world
database. Use this to inspect real game data.

### Connection

```bash
docker exec -it ac-mysql mariadb -u root -pac_password world
```

### Key Tables for Character Creation

| Table | Rows | Purpose |
|-------|------|---------|
| `playercreateinfo` | 62 | Starting position per race/class |
| `playercreateinfo_item` | 1 | Custom starting items |
| `playercreateinfo_action` | 325 | Starting action bar per race/class |
| `playercreateinfo_spell` | 182 | Starting spells per race/class |
| `playercreateinfo_spell_custom` | 0 | Custom spell overrides (empty) |
| `item_template` | 38,610 | All item definitions |
| `creature_template` | 29,928 | NPC/creature definitions |
| `creature` | 145,946 | NPC spawn points |
| `quest_template` | 9,464 | Quest definitions |
| `gossip_menu` | 5,665 | NPC dialogue menus |
| `npc_vendor` | 37,449 | NPC vendor sell lists |
| `player_levelstats` | 4,960 | Base stats per race/class/level |
| `player_classlevelstats` | 800 | Base HP/mana per class/level |

### Sample Queries

```sql
-- Starting positions for all race/class combos
SELECT race, class, map, zone, position_x, position_y, position_z, orientation
FROM playercreateinfo;

-- Starting items (only 1 row in this DB — most items come from CharStartOutfit.dbc)
SELECT * FROM playercreateinfo_item;

-- Action bar for Human Paladin (race=1, class=2)
SELECT button, action, type FROM playercreateinfo_action
WHERE race=1 AND class=2;

-- Item template lookup
SELECT entry, name, class, subclass, displayid, InventoryType, BuyCount
FROM item_template WHERE entry=23322;

-- Base stats for Human Paladin at level 1
SELECT * FROM player_levelstats WHERE race=1 AND class=2 AND level=1;

-- Class base HP/mana
SELECT * FROM player_classlevelstats WHERE class=2 LIMIT 10;
```

### Table: `playercreateinfo`

```sql
CREATE TABLE playercreateinfo (
  race      tinyint(3) unsigned NOT NULL DEFAULT '0',
  class     tinyint(3) unsigned NOT NULL DEFAULT '0',
  map       smallint(5) unsigned NOT NULL DEFAULT '0',
  zone      mediumint(8) unsigned NOT NULL DEFAULT '0',
  position_x float NOT NULL DEFAULT '0',
  position_y float NOT NULL DEFAULT '0',
  position_z float NOT NULL DEFAULT '0',
  orientation float NOT NULL DEFAULT '0',
  PRIMARY KEY (race, class)
);
```

### Table: `item_template` (key columns)

```sql
CREATE TABLE item_template (
  entry          mediumint(8) unsigned NOT NULL,  -- item ID
  class          tinyint(3) unsigned NOT NULL,     -- ITEM_CLASS_*
  subclass       tinyint(3) unsigned NOT NULL,
  name           varchar(255) NOT NULL,
  displayid      mediumint(8) unsigned NOT NULL,  -- display model
  InventoryType  tinyint(3) unsigned NOT NULL,     -- equipment slot
  AllowableClass int(11) NOT NULL DEFAULT '-1',    -- class mask (-1 = all)
  AllowableRace  int(11) NOT NULL DEFAULT '-1',    -- race mask (-1 = all)
  ItemLevel      smallint(5) unsigned NOT NULL,
  RequiredLevel  tinyint(3) unsigned NOT NULL,
  BuyCount       tinyint(3) unsigned NOT NULL DEFAULT '1',
  BuyPrice       bigint(20) NOT NULL,
  SellPrice      int(10) unsigned NOT NULL,
  -- ... 10 stat columns, 5 dmg columns, etc.
  PRIMARY KEY (entry)
);
```

---

## 10. Summit MongoDB Storage Design

### Current State

Characters are stored as JSON blobs inside `summit-store.yaml`:

```yaml
accounts:
  - name: PLAYERONE
    data:
      characters: '[{"id":1,"name":"Test",...}]'  # JSON string in YAML
```

**Problems:**
- Characters are serialized as a JSON string inside a YAML `data` map
- All characters for all accounts in one file
- No individual character files — editing one character rewrites everything
- Not human-readable for debugging
- No proper indexes for lookups

### MongoDB Collections Design

#### `accounts` collection

```json
{
  "_id": ObjectId,
  "id": "01HXYZ...",              // XID string
  "name": "PLAYERONE",            // uppercase
  "email": "player@example.com",
  "salt": "9398c11e...",          // hex-encoded SRP6 salt
  "verifier": "3e3f49a5...",      // hex-encoded SRP6 verifier
  "createdAt": ISODate("2024-01-15T10:30:00Z"),
  "lastLogin": ISODate("2024-03-20T14:22:00Z"),
  "activated": true,
  "banInfo": null
}
```

**Indexes:**
- `{ name: 1 }` — unique, for login lookup

#### `characters` collection

```json
{
  "_id": ObjectId,
  "id": 1,                        // uint32 GUID (globally unique)
  "account": "PLAYERONE",         // uppercase account name

  // Identity
  "name": "Arthas",
  "race": 6,                      // Human
  "class": 2,                     // Paladin
  "gender": 0,                    // Male

  // Appearance
  "skin": 1,
  "face": 0,
  "hairStyle": 5,
  "hairColor": 3,
  "facialHair": 2,
  "outfitId": 0,

  // Position
  "location": {
    "map": 0,
    "zone": 12,
    "x": -8949.87,
    "y": -132.45,
    "z": 83.53,
    "o": 3.66
  },
  "bindLocation": {
    "map": 0,
    "zone": 12,
    "x": -8949.87,
    "y": -132.45,
    "z": 83.53,
    "o": 3.66
  },

  // Stats
  "level": 1,
  "xp": 0,
  "money": 10,
  "health": 100,
  "maxHealth": 100,
  "power": [0, 0, 0, 0, 0, 0, 0, 0],
  "maxPower": [0, 0, 0, 0, 0, 0, 0, 0],

  // Display
  "displayId": 49,
  "nativeDisplayId": 49,

  // Flags
  "guildId": 0,
  "charFlags": 0,
  "playerFlags": 0,
  "recustomization": 0,
  "firstLogin": 1,

  // Pet
  "pet": {
    "displayId": 0,
    "level": 0,
    "family": 0
  },

  // Inventory (embedded array)
  "inventory": [
    { "slot": 0,  "itemId": 23322, "enchant": 0, "count": 1 },
    { "slot": 4,  "itemId": 23344, "enchant": 0, "count": 1 },
    { "slot": 23, "itemId": 4540,  "enchant": 0, "count": 4 },
    { "slot": 24, "itemId": 4542,  "enchant": 0, "count": 2 }
  ],

  // Spells
  "knownSpells": [20154, 21084, 465],

  // Action bar (packed uint32 per button)
  "actions": [6603, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],

  // Timestamps
  "createdAt": ISODate("2024-01-15T10:30:00Z"),
  "updatedAt": ISODate("2024-03-20T14:22:00Z")
}
```

**Indexes:**
- `{ id: 1 }` — unique, for GUID lookups
- `{ account: 1 }` — for listing characters per account
- `{ name: 1 }` — unique, for name uniqueness checks

#### `world_data` collection (optional, for runtime world state)

```json
{
  "_id": ObjectId,
  "type": "spawn",
  "map": 0,
  "entry": 1234,                 // creature_template.entry
  "guid": 12345,                 // spawn GUID
  "position": { "x": 0, "y": 0, "z": 0, "o": 0 },
  "spawntimesecs": 300
}
```

### Interface Changes

The `CharacterRepo` interface needs an update method:

```go
type CharacterRepo interface {
    GetCharacters(account string) (player.Players, error)
    GetCharacter(guid uint32) (*player.Player, error)
    CreateCharacter(account string, character *player.Player) error
    UpdateCharacter(character *player.Player) error  // NEW: for periodic saves
    DeleteCharacter(characterID int) error
}
```

### MongoDB Store Implementation

```go
// internal/store/mongostore/mongostore.go

package mongostore

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
    client     *mongo.Client
    db         *mongo.Database
    accounts   *mongo.Collection
    characters *mongo.Collection
}

func New(ctx context.Context, uri, database string) (*MongoStore, error) {
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }

    db := client.Database(database)
    return &MongoStore{
        client:     client,
        db:         db,
        accounts:   db.Collection("accounts"),
        characters: db.Collection("characters"),
    }, nil
}
```

### GUID Assignment

Use MongoDB's `counters` collection for atomic GUID allocation:

```json
// counters collection
{ "_id": "character_guid", "seq": 42 }
```

```go
func (m *MongoStore) nextGUID(ctx context.Context) (uint32, error) {
    filter := bson.M{"_id": "character_guid"}
    update := bson.M{"$inc": bson.M{"seq": 1}}
    opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

    var result struct{ Seq int32 `bson:"seq"` }
    err := m.db.Collection("counters").FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
    return uint32(result.Seq), err
}
```

### Migration from YAML

To migrate existing `summit-store.yaml` data to MongoDB:

```bash
# 1. Start MongoDB
docker run -d --name ac-mongo -p 27017:27017 mongo:7

# 2. Write a migration script (cmd/migrate/main.go) that:
#    - Reads summit-store.yaml
#    - Parses the JSON characters from Account.Data["characters"]
#    - Inserts into MongoDB accounts + characters collections
```

---

## 11. Implementation Status

### Current Summit Implementation

| Feature | Status | Location |
|---------|--------|----------|
| `CMSG_CHAR_CREATE` handler | ✅ Done | `pkg/summit/world/character.go:50` |
| `CMSG_CHAR_ENUM` handler | ✅ Done | `pkg/summit/world/character.go:12` |
| `Player` struct | ✅ Done | `pkg/summit/world/object/player/player.go:137` |
| `Player.Init()` (update fields) | ✅ Done | `pkg/summit/world/object/player/player.go:273` |
| `Player.ToCharacterEnum()` | ✅ Done | `pkg/summit/world/object/player/player.go:521` |
| `Player.InitInventory()` | ✅ Done | `pkg/summit/world/object/player/player.go:238` |
| `basedata.PlayerCreateInfo` | ✅ Done | `pkg/summit/world/basedata/character.go:13` |
| YAML persistence | ✅ Done | `internal/store/localdb/localstore.go` |
| `SessionManager` interface | ✅ Done | `pkg/summit/world/sessionmanager.go:10` |

### Missing / To Implement

| Feature | Priority | Notes |
|---------|----------|-------|
| Name validation | High | Need name rules, reserved words, min/max length |
| Race/class validation | High | Validate against DBC data |
| Expansion check | Medium | Account expansion vs race/class requirement |
| Account/realm char limits | Medium | `CONFIG_CHARACTERS_PER_ACCOUNT`, `CONFIG_CHARACTERS_PER_REALM` |
| Faction lock (PvP) | Low | First-char faction determination |
| DK restrictions | Medium | Level requirement, slot limit |
| CharStartOutfit DBC parsing | High | Parse DBC to get starting items |
| Custom starting items | Medium | `playercreateinfo_item` equivalent |
| Starting spells | Medium | `LearnDefaultSkills()` + `LearnCustomSpells()` |
| Starting action bar | Medium | `playercreateinfo_action` equivalent |
| Collector's edition voucher | Low | Account flag check + race-specific voucher |
| Auto-equip after creation | Medium | Second pass to equip from bags |
| Character delete handler | Medium | `CMSG_CHAR_DELETE` (0x038) |
| Character rename/customize | Low | `CMSG_CHAR_CUSTOMIZE`, `CMSG_CHAR_RACE_CHANGE` |
| YAML file-per-character | High | Replace JSON-in-YAML with proper files |
| Character cache | Medium | In-memory cache for name↔GUID lookups |

### DBC Data Needed

For a complete implementation, Summit needs DBC data for:

1. **CharStartOutfit.dbc** — Starting equipment per race/class/gender
2. **ChrRaces.dbc** — Race names, display IDs, expansion
3. **ChrClasses.dbc** — Class names, power types, expansion

The `datagen` tool can convert these DBCs to Go-compatible data. See
`cmd/datagen/` for the existing DBC conversion infrastructure.

---

## Appendix A: AzerothCore Source References

| File | Key Functions |
|------|--------------|
| `src/server/game/Handlers/CharacterHandler.cpp` | `HandleCharCreateOpcode`, `HandleCharEnum`, `HandleCharEnumOpcode` |
| `src/server/game/Entities/Player/Player.cpp` | `Player::Create`, `Player::BuildEnumData` |
| `src/server/game/Globals/ObjectMgr.cpp` | `LoadPlayerInfo`, `PlayerCreateInfoAddItemHelper`, `GetPlayerInfo` |
| `src/server/game/DataStores/DBCStores.cpp` | `GetCharStartOutfitEntry` |
| `src/server/game/Server/WorldSession.h` | `CharacterCreateInfo` class |

## Appendix B: Summit Source References

| File | Key Functions |
|------|--------------|
| `pkg/summit/world/character.go` | `CreateCharacter`, `SendCharacterEnum` |
| `pkg/summit/world/object/player/player.go` | `Player.Init`, `Player.ToCharacterEnum`, `Player.InitInventory` |
| `pkg/summit/world/basedata/character.go` | `PlayerCreateInfo`, `LookupCharacterCreateInfo` |
| `pkg/summit/world/basedata/store.go` | `LoadFromFile` |
| `pkg/summit/world/sessionmanager.go` | `SessionManager` interface |
| `internal/store/localdb/localstore.go` | `LocalStore` (YAML persistence) |
