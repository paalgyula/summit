# Quest System Port: AzerothCore → Summit

## Overview

Port the quest system from AzerothCore (C++) to Summit (Go). This covers quest templates, quest state on players, quest giver NPCs, quest packets, completion tracking, and critter/NPC spawn infrastructure.

**Current state in Summit:** Quest data exists in MongoDB (via `datagen migrate`), but nothing consumes it at runtime. No quest handlers, no quest state on Player, no quest giver relations. NPC spawns are hardcoded (one Training Dummy).

---

## Phase 1: Data Layer — WorldRepo & Store

**Goal:** Define quest/creature data structures and load them from MongoDB.

### 1.1 Extend `WorldRepo` interface

`pkg/store/stores.go` — add quest and creature methods:

```go
type WorldRepo interface {
    // Creature templates
    GetCreatureTemplate(entry uint32) (*CreatureTemplate, error)
    GetCreatureTemplates() (map[uint32]*CreatureTemplate, error)

    // Creature spawns
    GetCreatureSpawns(mapID uint32) ([]*CreatureSpawn, error)
    GetCreatureSpawn(spawnID uint64) (*CreatureSpawn, error)

    // Quest templates
    GetQuestTemplate(id uint32) (*QuestTemplate, error)
    GetQuestTemplates() (map[uint32]*QuestTemplate, error)

    // Quest relations (quest giver → quests)
    GetCreatureQuestRelations(creatureEntry uint32) ([]uint32, error)
    GetCreatureQuestInvolvedRelations(creatureEntry uint32) ([]uint32, error)
}
```

### 1.2 Entity structs

`pkg/store/model/world_entities.go` — new file:

```go
type CreatureTemplateEntity struct {
    Entry        uint32 `bson:"entry"`
    Name         string `bson:"name"`
    SubName      string `bson:"subName"`
    MinLevel     uint8  `bson:"minLevel"`
    MaxLevel     uint8  `bson:"maxLevel"`
    Faction      uint32 `bson:"faction"`
    NpcFlag      uint32 `bson:"npcFlag"`
    UnitFlags    uint32 `bson:"unitFlags"`
    DynamicFlags uint32 `bson:"dynamicFlags"`
    Type         uint32 `bson:"type"`
    Family       uint32 `bson:"family"`
    Rank         uint32 `bson:"rank"`
    HealthMultiplier float32 `bson:"healthMultiplier"`
    DamageMultiplier float32 `bson:"damageMultiplier"`
    ArmorMultiplier  float32 `bson:"armorMultiplier"`
    MovementType uint32 `bson:"movementType"`
    // ... more fields as needed
}

type CreatureSpawnEntity struct {
    SpawnID       uint64  `bson:"guid"`
    Entry         uint32  `bson:"entry"`
    MapID         uint32  `bson:"map"`
    SpawnMask     uint8   `bson:"spawnMask"`
    PhaseMask     uint32  `bson:"phaseMask"`
    PosX          float32 `bson:"posX"`
    PosY          float32 `bson:"posY"`
    PosZ          float32 `bson:"posZ"`
    Orientation   float32 `bson:"orientation"`
    SpawnTimeSecs uint32  `bson:"spawnTime"`
    WanderDistance float32 `bson:"wander_distance"`
    MovementType  uint8   `bson:"movementType"`
    CurrentWaypoint uint32 `bson:"currentwaypoint"`
    CurrentHealth uint32  `bson:"curhealth"`
    CurrentMana   uint32  `bson:"curmana"`
    NpcFlag       uint32  `bson:"npcflag"`
    UnitFlags     uint32  `bson:"unit_flags"`
    DynamicFlags  uint32  `bson:"dynamicflags"`
}

type QuestTemplateEntity struct {
    ID             uint32 `bson:"id"`
    QuestLevel     int16  `bson:"questLevel"`
    MinLevel       uint8  `bson:"minLevel"`
    QuestType      uint8  `bson:"questType"`
    QuestSortID    int16  `bson:"questSortID"`
    RequiredClasses uint32 `bson:"requiredClasses"`
    RequiredRaces  uint32 `bson:"requiredRaces"`
    RequiredItemId [6]uint32  // parsed from bson
    RequiredItemCount [6]uint16
    RequiredNpcOrGo [4]int32
    RequiredNpcOrGoCount [4]uint16
    RewardItemId   [4]uint32
    RewardItemCount [4]uint16
    RewardChoiceItemId [6]uint32
    RewardChoiceItemCount [6]uint16
    RewardMoney    int32  `bson:"rewardMoney"`
    RewardXP       uint32 `bson:"rewardXP"`
    Flags          uint32 `bson:"flags"`
    // ... more fields as needed
}
```

### 1.3 Implement on MongoDB store

`internal/store/mongostore/worldstore.go` — new file implementing `WorldRepo`:

- Query `creatureTemplate` collection (already migrated by `datagen migrate`)
- Query `creature` collection (already migrated)
- Query `questTemplate` collection (already migrated)
- For quest relations: query `creature_questrelation` / `creature_involvedrelation` collections (need to add migration for these tables)

### 1.4 Add missing migration tables

`cmd/datagen/migrate.go` — add migration for:

- `creature_questrelation` → `creatureQuestRelation` collection
- `creature_involvedrelation` → `creatureInvolvedRelation` collection
- `quest_template_addon` → `questTemplateAddon` collection
- `creature_addon` → `creatureAddon` collection

**Effort:** ~2-3 days

---

## Phase 2: Quest Data Structures (Go types)

**Goal:** Define the in-memory quest representation and quest state types.

### 2.1 Quest template

`pkg/summit/world/quest/quest_def.go` — new package:

```go
type QuestFlags uint32

const (
    QuestFlagsStayAlive     QuestFlags = 0x0001
    QuestFlagsPartyAccept   QuestFlags = 0x0002
    QuestFlagsSharable      QuestFlags = 0x0008
    QuestFlagsDaily         QuestFlags = 0x1000
    QuestFlagsWeekly        QuestFlags = 0x8000
    QuestFlagsAutoComplete  QuestFlags = 0x10000
    QuestFlagsAutoAccept    QuestFlags = 0x80000
)

type QuestSpecialFlags uint32

const (
    QuestSpecialRepeatable            QuestSpecialFlags = 0x01
    QuestSpecialExplorationOrEvent    QuestSpecialFlags = 0x02
    QuestSpecialAutoAccept            QuestSpecialFlags = 0x04
    QuestSpecialDFQuest               QuestSpecialFlags = 0x08
    QuestSpecialCast                  QuestSpecialFlags = 0x20
)

type Quest struct {
    ID          uint32
    Method      uint32
    ZoneOrSort  int32
    MinLevel    uint32
    Level       int32
    Type        uint32
    AllowableRaces uint32
    Flags       QuestFlags
    SpecialFlags QuestSpecialFlags
    TimeAllowed uint32
    StartItem   uint32
    RequiredPlayerKills uint32
    RewardMoney int32
    RewardXP    uint32

    RequiredItemId     [6]uint32
    RequiredItemCount  [6]uint16
    RequiredNpcOrGo    [4]int32   // >0 = creature, <0 = GO (abs value)
    RequiredNpcOrGoCount [4]uint16

    RewardChoiceItemId    [6]uint32
    RewardChoiceItemCount [6]uint16
    RewardItemId          [4]uint32
    RewardItemCount       [4]uint16

    RewardFactionId   [5]uint32
    RewardFactionValue [5]int32

    // Text fields
    Title               string
    Description         string
    Objectives          string
    AreaDescription     string
    QuestCompletionLog  string
    ObjectiveText       [4]string

    // Addon fields
    PrevQuestId     int32
    NextQuestId     uint32
    ExclusiveGroup  int32
    BreadcrumbForQuestId uint32
    RequiredSkillId     uint16
    RequiredSkillPoints uint16
}
```

### 2.2 Quest status on player

`pkg/summit/world/quest/quest_status.go`:

```go
type QuestStatus uint32

const (
    QuestStatusNone        QuestStatus = 0
    QuestStatusUnavailable QuestStatus = 1
    QuestStatusComplete    QuestStatus = 2
    QuestStatusIncomplete  QuestStatus = 3
    QuestStatusRewarded    QuestStatus = 4
    QuestStatusFailed      QuestStatus = 5
)

type QuestStatusData struct {
    Status            QuestStatus
    Timer             uint32
    ItemCount         [6]uint16
    CreatureOrGOCount [4]uint16
    PlayerCount       uint16
    Explored          bool
}
```

### 2.3 Quest giver status enum

```go
type QuestGiverStatus uint32

const (
    DialogStatusNone                QuestGiverStatus = 0
    DialogStatusUnavailable         QuestGiverStatus = 1
    DialogStatusLowLevelAvailable   QuestGiverStatus = 2
    DialogStatusLowLevelRewardRep   QuestGiverStatus = 3
    DialogStatusLowLevelAvailableRep QuestGiverStatus = 4
    DialogStatusIncomplete          QuestGiverStatus = 5
    DialogStatusRewardRep           QuestGiverStatus = 6
    DialogStatusAvailable           QuestGiverStatus = 8
    DialogStatusReward              QuestGiverStatus = 10
)
```

### 2.4 Add quest state to Player

`pkg/summit/world/object/player/player.go` — add fields:

```go
// Quest state
QuestStatus    map[uint32]*quest.QuestStatusData  // active quests
RewardedQuests map[uint32]bool                     // completed quest IDs
ActiveQuests   map[uint32]*quest.Quest              // cached templates for active quests
```

**Effort:** ~1-2 days

---

## Phase 3: Quest Data Loading at Startup

**Goal:** Load quest templates and creature data into memory at server start.

### 3.1 Quest manager

`pkg/summit/world/quest/quest_manager.go`:

```go
type QuestManager struct {
    templates      map[uint32]*Quest
    creatureQuests map[uint32][]uint32      // creature_entry → quest IDs offered
    creatureInvolved map[uint32][]uint32    // creature_entry → quest IDs completed
}

func NewQuestManager(worldRepo store.WorldRepo) (*QuestManager, error) {
    // Load all quest templates from MongoDB
    // Load quest giver relations
    // Build lookup maps
}
```

### 3.2 Extend NPC with quest data

`pkg/summit/world/npc.go` — add to NPC struct:

```go
type NPC struct {
    // ... existing fields ...
    NpcFlags      uint32  // from creature_template
    QuestGiver    bool    // computed from NpcFlags & 0x02
}
```

### 3.3 Wire into server startup

`cmd/summit/summit.go` — pass `WorldRepo` to world server, which creates `QuestManager`.

**Effort:** ~1-2 days

---

## Phase 4: Quest Packet Handlers

**Goal:** Implement all quest-related CMSG handlers and SMSG response builders.

### 4.1 Quest packets to implement

**Core quest interaction (must-have):**

| Opcode | Handler | Purpose |
|--------|---------|---------|
| `CMSG_QUESTGIVER_HELLO` | `HandleQuestgiverHello` | Right-click NPC → open gossip with quest list |
| `CMSG_QUESTGIVER_QUERY_QUEST` | `HandleQuestgiverQueryQuest` | View quest details |
| `CMSG_QUESTGIVER_ACCEPT_QUEST` | `HandleQuestgiverAcceptQuest` | Accept a quest |
| `CMSG_QUESTGIVER_COMPLETE_QUEST` | `HandleQuestgiverCompleteQuest` | Turn-in quest |
| `CMSG_QUESTGIVER_REQUEST_REWARD` | `HandleQuestgiverRequestReward` | Request reward selection |
| `CMSG_QUESTGIVER_CHOOSE_REWARD` | `HandleQuestgiverChooseReward` | Choose reward & finalize |
| `CMSG_QUESTGIVER_CANCEL` | `HandleQuestgiverCancel` | Close quest dialog |
| `CMSG_QUESTLOG_REMOVE_QUEST` | `HandleQuestLogRemoveQuest` | Abandon quest |
| `CMSG_QUESTGIVER_STATUS_QUERY` | `HandleQuestgiverStatusQuery` | Query NPC status icon |

**Server responses (must-have):**

| Opcode | Packet | Purpose |
|--------|--------|---------|
| `SMSG_QUESTGIVER_STATUS` | `QuestGiverStatus` | Single NPC status icon |
| `SMSG_QUESTGIVER_QUEST_LIST` | `QuestGiverQuestList` | NPC's available quest list |
| `SMSG_QUESTGIVER_QUEST_DETAILS` | `QuestGiverQuestDetails` | Quest details before accept |
| `SMSG_QUESTGIVER_REQUEST_ITEMS` | `QuestGiverRequestItems` | Turn-in requirements |
| `SMSG_QUESTGIVER_OFFER_REWARD` | `QuestGiverOfferReward` | Reward selection screen |
| `SMSG_QUESTGIVER_QUEST_INVALID` | `QuestGiverQuestInvalid` | Can't accept reason |
| `SMSG_QUESTGIVER_QUEST_COMPLETE` | `QuestGiverQuestComplete` | Completed with rewards |
| `SMSG_QUESTGIVER_QUEST_FAILED` | `QuestGiverQuestFailed` | Quest failed |

**Progress packets (must-have):**

| Opcode | Packet | Purpose |
|--------|--------|---------|
| `SMSG_QUESTUPDATE_ADD_KILL` | `QuestUpdateAddKill` | Kill credit |
| `SMSG_QUESTUPDATE_ADD_ITEM` | `QuestUpdateAddItem` | Item credit |
| `SMSG_QUESTUPDATE_COMPLETE` | `QuestUpdateComplete` | All objectives done |

**Advanced (phase 2):**

| Opcode | Purpose |
|--------|---------|
| `CMSG_QUESTLOG_SWAP_QUEST` | Reorder quest log |
| `CMSG_QUEST_CONFIRM_ACCEPT` | Share quest confirmation |
| `CMSG_PUSHQUESTTOPARTY` | Share quest with party |
| `CMSG_QUEST_POI_QUERY` | Quest points of interest |
| `CMSG_QUESTGIVER_STATUS_MULTIPLE_QUERY` | Batch status for all visible NPCs |

### 4.2 Packet structures

`pkg/summit/world/quest/quest_packets.go` — new file:

```go
// SMSG_QUESTGIVER_STATUS
func BuildQuestGiverStatus(guid wow.GUID, status QuestGiverStatus) *wow.Packet

// SMSG_QUESTGIVER_QUEST_LIST
func BuildQuestGiverQuestList(guid wow.GUID, greeting string, quests []QuestListItem) *wow.Packet

// SMSG_QUESTGIVER_QUEST_DETAILS
func BuildQuestGiverQuestDetails(npcGUID wow.GUID, quest *Quest, activateAccept bool) *wow.Packet

// SMSG_QUESTGIVER_REQUEST_ITEMS
func BuildQuestGiverRequestItems(npcGUID wow.GUID, quest *Quest, canComplete bool) *wow.Packet

// SMSG_QUESTGIVER_OFFER_REWARD
func BuildQuestGiverOfferReward(npcGUID wow.GUID, quest *Quest, autoFinish bool) *wow.Packet

// SMSG_QUESTGIVER_QUEST_COMPLETE
func BuildQuestGiverQuestComplete(questID uint32, xp, money, honor, talents uint32) *wow.Packet

// SMSG_QUESTUPDATE_ADD_KILL
func BuildQuestUpdateAddKill(questID, creatureEntry, current, required uint32, guid wow.GUID) *wow.Packet
```

### 4.3 Handler implementations

`pkg/summit/world/quest_handlers.go` — new file on `WorldSession`:

```go
func (gc *WorldSession) HandleQuestgiverHello(data wow.PacketData) {
    reader := wow.NewPacketReader(data)
    var guid uint64
    _ = reader.Read(&guid)

    npc := gc.getNPCByGUID(guid)
    if npc == nil || !npc.QuestGiver {
        return
    }

    // Build quest list from QuestManager
    quests := gc.world.questManager.GetQuestsForCreature(npc.EntryID)
    // Send SMSG_QUESTGIVER_QUEST_LIST
}

func (gc *WorldSession) HandleQuestgiverAcceptQuest(data wow.PacketData) {
    // Parse: npcGUID, questID, ...
    // Validate: CanTakeQuest()
    // Add to player quest log
    // Send SMSG_GOSSIP_COMPLETE
}
```

### 4.4 Register handlers

`pkg/summit/world/handlers.go` — add to `RegisterHandlers`:

```go
// Quest handlers
packets.OpcodeTable.Handle(wow.ClientQuestgiverHello, gc.HandleQuestgiverHello)
packets.OpcodeTable.Handle(wow.ClientQuestgiverQueryQuest, gc.HandleQuestgiverQueryQuest)
packets.OpcodeTable.Handle(wow.ClientQuestgiverAcceptQuest, gc.HandleQuestgiverAcceptQuest)
packets.OpcodeTable.Handle(wow.ClientQuestgiverCompleteQuest, gc.HandleQuestgiverCompleteQuest)
packets.OpcodeTable.Handle(wow.ClientQuestgiverRequestReward, gc.HandleQuestgiverRequestReward)
packets.OpcodeTable.Handle(wow.ClientQuestgiverChooseReward, gc.HandleQuestgiverChooseReward)
packets.OpcodeTable.Handle(wow.ClientQuestgiverCancel, gc.HandleQuestgiverCancel)
packets.OpcodeTable.Handle(wow.ClientQuestgiverStatusQuery, gc.HandleQuestgiverStatusQuery)
packets.OpcodeTable.Handle(wow.ClientQuestlogRemoveQuest, gc.HandleQuestLogRemoveQuest)
```

**Effort:** ~5-7 days

---

## Phase 5: Quest Completion Logic

**Goal:** Implement objective tracking (kill credit, item credit, exploration).

### 5.1 Kill credit

`pkg/summit/world/quest/quest_completion.go`:

```go
func (qm *QuestManager) OnCreatureKilled(player *player.Player, creatureEntry uint32) {
    for questID, status := range player.QuestStatus {
        if status.Status != QuestStatusIncomplete {
            continue
        }
        quest := qm.templates[questID]
        for i := 0; i < 4; i++ {
            if quest.RequiredNpcOrGo[i] > 0 &&
               uint32(quest.RequiredNpcOrGo[i]) == creatureEntry &&
               status.CreatureOrGOCount[i] < quest.RequiredNpcOrGoCount[i] {
                status.CreatureOrGOCount[i]++
                // Send SMSG_QUESTUPDATE_ADD_KILL
                // Check CanCompleteQuest()
            }
        }
    }
}
```

### 5.2 Item credit

```go
func (qm *QuestManager) OnItemAdded(player *player.Player, itemEntry uint32, count uint32) {
    // Similar pattern: iterate quests, match RequiredItemId, update ItemCount
}
```

### 5.3 CanCompleteQuest check

```go
func (qm *QuestManager) CanCompleteQuest(player *player.Player, questID uint32) bool {
    quest := qm.templates[questID]
    status := player.QuestStatus[questID]

    // Check all kill/collection objectives met
    // Check exploration flag
    // Check reputation requirements
    // Check money cost
    return allObjectivesMet
}
```

### 5.4 Hook into existing systems

- **Combat system** (`pkg/summit/world/combat.go`): Call `OnCreatureKilled` when a creature dies
- **Item system** (`pkg/summit/world/item_handlers.go`): Call `OnItemAdded` when items are picked up
- **Player loot**: When a creature is looted, check quest items

**Effort:** ~3-4 days

---

## Phase 6: NPC Spawns from Database

**Goal:** Replace hardcoded NPC spawns with database-driven spawning.

### 6.1 Extend SpawnManager

`pkg/summit/world/npc.go`:

```go
func NewSpawnManagerFromDB(worldRepo store.WorldRepo) (*SpawnManager, error) {
    sm := &SpawnManager{npcs: make(map[uint32]*NPC)}

    spawns, err := worldRepo.GetCreatureSpawns(mapID)
    for _, spawn := range spawns {
        template, _ := worldRepo.GetCreatureTemplate(spawn.Entry)
        npc := NewNPCFromSpawn(spawn, template)
        sm.SpawnNPC(npc)
    }
    return sm, nil
}

func NewNPCFromSpawn(spawn *CreatureSpawnEntity, template *CreatureTemplateEntity) *NPC {
    return NewNPC(
        template.Entry,
        template.Name,
        template.DisplayID,
        template.Faction,
        template.MaxLevel,
        uint32(float32(template.BaseHealth) * template.HealthMultiplier),
        spawn.PosX, spawn.PosY, spawn.PosZ, spawn.Orientation,
        spawn.MapID,
    )
}
```

### 6.2 Set NPC flags from template

In `NewNPC` / `init()`, set `UNIT_NPC_FLAGS` update field:

```go
if npc.NpcFlags&0x02 != 0 { // UNIT_NPC_FLAG_QUESTGIVER
    n.Object.SetUInt32Value(object.UnitFieldNpcFlags, npc.NpcFlags)
}
```

### 6.3 Spawn by map

Instead of spawning all NPCs at startup, spawn per-map when a player enters. The `SpawnManager` should filter by map ID and only instantiate creatures in loaded grids.

**Effort:** ~2-3 days

---

## Phase 7: Quest Giver Status Icons

**Goal:** Show correct floating icons (!, ?, etc.) above quest giver NPCs.

### 7.1 Status calculation

```go
func (qm *QuestManager) GetQuestDialogStatus(player *player.Player, creatureEntry uint32) QuestGiverStatus {
    maxStatus := DialogStatusNone

    // Check quests offered by this NPC
    for _, questID := range qm.creatureQuests[creatureEntry] {
        status := qm.computeQuestStatus(player, questID)
        if status > maxStatus {
            maxStatus = status
        }
    }

    // Check quests completed by this NPC (turn-in)
    for _, questID := range qm.creatureInvolved[creatureEntry] {
        status := qm.computeTurnInStatus(player, questID)
        if status > maxStatus {
            maxStatus = status
        }
    }

    return maxStatus
}
```

### 7.2 Send status updates

After quest accept/complete/reward, call `SendQuestGiverStatusMultiple` to update all visible NPC icons.

**Effort:** ~1-2 days

---

## Phase 8: Critter/Ambient Creature System

**Goal:** Support ambient critters with appropriate behavior.

### 8.1 Critter characteristics

From AzerothCore analysis:
- Critters are regular `creature_template` entries with:
  - Low health (typically 1-5 HP)
  - No loot, no XP
  - `npcflag = 0` (no interaction)
  - May have `CREATURE_TYPE_FLAG_AMBIENT_ANIMATIONS`
  - Short respawn timers (5-30 seconds)
  - Often `unit_flags` with `UNIT_FLAG_IMMUNE_TO_NPC` or `UNIT_FLAG_NOT_SELECTABLE`

### 8.2 No special code needed

Critters don't require dedicated systems. They're just creature_template entries with specific properties. The existing creature spawn system handles them naturally. Just ensure:
- `creature_template` entries with critter characteristics are loaded correctly
- Low respawn timers work
- Ambient animations (bytes1/bytes2 from `creature_addon`) are applied

**Effort:** ~0.5 days (just validation)

---

## Implementation Order

| Phase | Description | Days | Dependencies |
|-------|-------------|------|-------------|
| 1 | WorldRepo + Store layer | 2-3 | None |
| 2 | Quest data structures | 1-2 | None |
| 3 | Quest loading at startup | 1-2 | Phase 1, 2 |
| 4 | Quest packet handlers | 5-7 | Phase 2, 3 |
| 5 | Quest completion logic | 3-4 | Phase 4 |
| 6 | NPC spawns from DB | 2-3 | Phase 1 |
| 7 | Quest giver status icons | 1-2 | Phase 4, 5 |
| 8 | Critter system | 0.5 | Phase 6 |

**Total estimated effort:** ~16-23 days

---

## Key Files to Create/Modify

### New files:
- `pkg/store/model/world_entities.go` — entity structs
- `internal/store/mongostore/worldstore.go` — WorldRepo implementation
- `pkg/summit/world/quest/quest_def.go` — Quest struct, enums
- `pkg/summit/world/quest/quest_status.go` — QuestStatusData
- `pkg/summit/world/quest/quest_manager.go` — loading + lookup
- `pkg/summit/world/quest/quest_completion.go` — objective tracking
- `pkg/summit/world/quest/quest_packets.go` — packet builders
- `pkg/summit/world/quest_handlers.go` — CMSG handlers on WorldSession

### Modified files:
- `pkg/store/stores.go` — extend WorldRepo interface
- `pkg/summit/world/npc.go` — NPC flags, DB spawning
- `pkg/summit/world/object/player/player.go` — add quest state fields
- `pkg/summit/world/handlers.go` — register quest handlers
- `pkg/summit/world/combat.go` — hook kill credit
- `cmd/datagen/migrate.go` — add missing table migrations

---

## Risk Areas

1. **MongoDB query performance** — Quest/creature data loaded at startup, cached in memory. No runtime DB queries per player action.
2. **Packet structure accuracy** — Must match WotLK 3.3.5a client expectations exactly. Cross-reference with AzerothCore's `QuestPackets.h` and `GossipDef.cpp`.
3. **Quest validation** — AzerothCore has ~50 validation rules on quest load. Implement the critical ones first (item/creature existence, race/class masks).
4. **Update fields** — `UNIT_NPC_FLAGS` must be set correctly for quest givers to show icons. Currently Summit doesn't set this field at all.
5. **Gossip system** — Quest interaction is built on top of the gossip system (`SMSG_GOSSIP_MESSAGE`). Summit has basic gossip support via the web client but may need extensions for quest menus.
