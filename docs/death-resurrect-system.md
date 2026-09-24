# Death / Resurrect / Spirit Healer / Respawn System

> Reference implementations: TrinityCore and AzerothCore
> All packet structures MUST match 3.3.5a exactly (field order, types, opcodes)

## Table of Contents
1. [Opcode Table](#1-opcode-table)
2. [Death State Machine](#2-death-state-machine)
3. [Complete Death Flow](#3-complete-death-flow)
4. [Packet Handlers (Client → Server)](#4-packet-handlers)
5. [Server → Client Packets](#5-server-client-packets)
6. [Spirit Healer System](#6-spirit-healer-system)
7. [Corpse & Bones](#7-corpse--bones)
8. [Durability Loss](#8-durability-loss)
9. [Resurrect Methods](#9-resurrect-methods)
10. [Corpse Reclaim Delay](#10-corpse-reclaim-delay)
11. [Constants](#11-constants)

---

## 1. Opcode Table

### Client → Server (CMSG)

| Opcode | Hex | Handler | TC File | AC File |
|--------|-----|---------|---------|---------|
| CMSG_REPOP_REQUEST | 0x15A | HandleRepopRequest | MiscHandler.cpp:60 | MiscHandler.cpp:59 |
| CMSG_RESURRECT_RESPONSE | 0x15C | HandleResurrectResponse | MiscHandler.cpp:579 | MiscHandler.cpp:668 |
| CMSG_RECLAIM_CORPSE | 0x1D2 | HandleReclaimCorpse | MiscHandler.cpp:546 | MiscHandler.cpp:633 |
| CMSG_SPIRIT_HEALER_ACTIVATE | 0x21C | HandleSpiritHealerActivate | NPCHandler.cpp:192 | NPCHandler.cpp:238 |
| CMSG_AREA_SPIRIT_HEALER_QUERY | 0x2E2 | HandleAreaSpiritHealerQuery | BattlegroundMgr.cpp:676 | MiscHandler.cpp:1640 |
| CMSG_AREA_SPIRIT_HEALER_QUEUE | 0x2E3 | HandleAreaSpiritHealerQueue | Battleground.cpp:1257 | MiscHandler.cpp:1663 |

### Server → Client (SMSG)

| Opcode | Hex | Purpose | Notes |
|--------|-----|---------|-------|
| SMSG_PRE_RESURRECT | 0x494 | Before ghost transition | PackedGuid playerGUID |
| SMSG_RESURRECT_REQUEST | 0x15B | Offer resurrection | Shows accept/decline dialog |
| SMSG_DEATH_RELEASE_LOC | 0x378 | Spirit healer minimap pin | map=-1 clears pin |
| SMSG_CORPSE_RECLAIM_DELAY | 0x269 | Corpse reclaim timer | uint32 milliseconds |
| SMSG_AREA_SPIRIT_HEALER_TIME | 0x2E4 | BG spirit healer timer | ObjectGuid + int32 timeLeft |
| SMSG_DURABILITY_DAMAGE_DEATH | — | Durability flash | Empty body (opcode only) |

---

## 2. Death State Machine

### DeathState Enum (TC/AC Unit.h)

```cpp
enum DeathState
{
    ALIVE          = 0,
    JUST_DIED      = 1,  // Just killed, before next world update
    CORPSE         = 2,  // Player corpse exists, timer running
    DEAD           = 3,  // Spirit released (ghost form)
    JUST_RESPAWNED = 4   // Just respawned
};
```

### Key Player Flags

```cpp
PLAYER_FLAGS_GHOST            = 0x00000010,  // Ghost form
PLAYER_FLAGS_IS_OUT_OF_BOUNDS = 0x00000200,  // Out of world bounds
```

### Death Constants (TC/AC Player.cpp)

```cpp
#define DEATH_EXPIRE_STEP (5*MINUTE)               // 300 seconds per tier
#define MAX_DEATH_COUNT 3                           // Max escalating tiers
static uint32 corpseReclaimDelay[MAX_DEATH_COUNT] = { 30, 60, 120 }; // seconds
```

---

## 3. Complete Death Flow

### Phase 1: Player Dies (KillPlayer)

**Entry:** `Unit::DealDamage()` → `victim->setDeathState(DeathState::JustDied)`

#### Step 1a: Player::setDeathState(JUST_DIED) (Player.cpp)

```
1. ClearSpellQueue
2. SetDrunkValue(0)                    // Clear drunken state
3. ClearComboPoints()
4. clearResurrectRequestData()         // Clear pending resurrections
5. RemovePet(nullptr, PET_SAVE_NOT_IN_SLOT, true)
6. Save ressSpellId = GetUInt32Value(PLAYER_SELF_RES_SPELL)
7. GetResurrectionSpellId()            // Find self-res (Reincarnation, Soulstone)
8. FailQuestsOnDeath()
9. Update achievement criteria
10. Call Unit::setDeathState(JUST_DIED)
```

#### Step 1b: Unit::setDeathState(JUST_DIED) (Unit.cpp)

```
1. m_deathState = JustDied
2. CombatStop()                        // Leave combat
3. ClearComboPointHolders()
4. InterruptNonMeleeSpells(false)      // Cancel casts
5. UnsummonAllTotems(true)
6. RemoveAllControlled(true)           // Dismiss pets/guards
7. RemoveAllAurasOnDeath()             // Remove non-passive auras
8. ClearAllReactives()
9. ClearDiminishings()
10. GetMotionMaster()->Clear() + MoveIdle()
11. StopMoving()
12. SetHealth(0)                       // ← HEALTH SET TO 0
13. SetPower(getPowerType(), 0)        // ← POWER SET TO 0
14. SetUInt32Value(UNIT_NPC_EMOTESTATE, 0)
```

#### Step 1c: Player::KillPlayer() (Player.cpp)

```
1. MoveFall() if flying
2. SendMoveRoot(true)                  // Freeze movement
3. StopMirrorTimers()                  // Stop breath/fatigue bars
4. setDeathState(CORPSE)               // State transition
5. ReplaceAllDynamicFlags(UNIT_DYNFLAG_NONE)
6. ApplyModFlag(PLAYER_FIELD_BYTES, PLAYER_FIELD_BYTE_RELEASE_TIMER, ...)  // Enable "Release Spirit"
7. m_deathTimer = 6 * MINUTE * IN_MILLISECONDS  // 6 minute countdown
8. UpdateCorpseReclaimDelay()          // Escalate reclaim delay
9. SendCorpseReclaimDelay(calculateCorpseReclaimDelay())
```

**Durability loss happens here** (Unit.cpp):
```cpp
if (Player* plrVictim = victim->ToPlayer())
{
    plrVictim->DurabilityLossAll(sWorld->getRate(RATE_DURABILITY_LOSS_ON_DEATH) / 100.0f, false);
    plrVictim->SendDurabilityLoss();  // SMSG_DURABILITY_DAMAGE_DEATH
}
```

**SMSG_UPDATE_OBJECT sent automatically:**
- Health=0, Power=0, UNIT_FLAG_DEAD set, DynamicFlags cleared
- Auras removed (SMSG_AURA_UPDATE for each)

---

### Phase 2: Player Clicks "Release Spirit"

**Client sends:** CMSG_REPOP_REQUEST

**Handler:** HandleRepopRequest (MiscHandler.cpp:60)

```
1. Guard: if alive or already ghost → return
2. Guard: if HasPreventResurectionAura() → return
3. If JUST_DIED state (lag race) → call KillPlayer() first
4. RemovePet()
5. BuildPlayerRepop()                  // THE KEY FUNCTION
6. RepopAtGraveyard()                  // Teleport to graveyard
```

#### BuildPlayerRepop() (Player.cpp:4319)

```
1. Send SMSG_PRE_RESURRECT with GetPackGUID()
2. If Night Elf → CastSpell(20584)     // Racial ghost speed
3. CastSpell(8326, true)               // SPELL_AURA_GHOST
   - Sets PLAYER_FLAGS_GHOST
   - Sets SERVERSIDE_VISIBILITY_GHOST
   - Makes player visible only to ghosts/spirit healers
4. CreateCorpse()                       // Create corpse at death location
5. GetMap()->AddToMap(corpse)
6. SetHealth(1)                         // Ghost body = 1 HP
7. SetWaterWalking(true)                // Ghost can walk on water
8. SendMoveRoot(false)                  // Allow movement
9. RemoveUnitFlag(UNIT_FLAG_SKINNABLE)
10. SendCorpseReclaimDelay(CalculateCorpseReclaimDelay())
11. corpse->ResetGhostTime()
12. StopMirrorTimers()
13. SetByteValue(UNIT_FIELD_BYTES_1, ANIM_TIER, UNIT_BYTE1_FLAG_ALWAYS_STAND)  // Ghost animation
```

**Packets sent:**
1. `SMSG_PRE_RESURRECT` (0x494) — PackedGuid playerGUID
2. `SMSG_AURA_UPDATE` — Ghost aura (8326)
3. `SMSG_FORCE_RUN_SPEED_CHANGE` — Ghost speed = 8
4. `SMSG_FORCE_SWIM_SPEED_CHANGE` — Ghost swim speed
5. `SMSG_MOVE_WATER_WALK` — Set water walking
6. `SMSG_UPDATE_OBJECT` — Corpse added to map
7. `SMSG_STANDSTATE_UPDATE` (0x29D) — Ghost animation
8. `SMSG_CORPSE_RECLAIM_DELAY` (0x269) — uint32 delayMs

#### RepopAtGraveyard() (Player.cpp:4800)

```
1. Get zone info (AreaTableEntry)
2. If in AREA_FLAG_NEED_FLY zone → auto-resurrect
3. Find closest graveyard:
   - Battleground → bg->GetClosestGraveyard(this)
   - Battlefield → bf->GetClosestGraveyard(this)
   - Normal → sGraveyard->GetClosestGraveyard(this, GetTeamId())
4. m_deathTimer = 0                     // Stop countdown
5. TeleportTo(graveyard position)
6. If dead → Send SMSG_DEATH_RELEASE_LOC (minimap pin)
```

**SMSG_DEATH_RELEASE_LOC (0x378):**
```
uint32  mapId     // Map of spirit healer
float   x         // X coordinate
float   y         // Y coordinate
float   z         // Z coordinate

To CLEAR pin (on resurrect): map=0xFFFFFFFF (-1), x=0, y=0, z=0
```

---

## 4. Packet Handlers

### CMSG_REPOP_REQUEST (0x15A) — Release Spirit

**Packet structure:**
```
uint8  (skipped, unused)
```

**Pre-conditions:**
- Player must be dead
- Not already ghost
- No prevent-resurrection aura

**Actions:**
1. RemovePet()
2. BuildPlayerRepop() → create corpse, ghost form, SMSG_PRE_RESURRECT, ghost aura, SMSG_CORPSE_RECLAIM_DELAY
3. RepopAtGraveyard() → teleport to graveyard, SMSG_DEATH_RELEASE_LOC

---

### CMSG_RECLAIM_CORPSE (0x1D2) — Resurrect at Corpse

**Packet structure:**
```
ObjectGuid  corpseGUID
```

**Pre-conditions:**
- Player must be dead (ghost)
- Not in arena
- Corpse exists
- Reclaim delay expired
- Corpse within CORPSE_RECLAIM_RADIUS

**Actions:**
1. ResurrectPlayer(0.5f) — 50% health (100% in BG)
2. SpawnCorpseBones()

---

### CMSG_SPIRIT_HEALER_ACTIVATE (0x21C) — Talk to Spirit Healer

**Packet structure:**
```
ObjectGuid  creatureGUID
```

**Pre-conditions:**
- NPC must have UNIT_NPC_FLAG_SPIRITHEALER
- Player must be dead

**Actions:**
1. Remove feign death if active
2. SendSpiritResurrect()

---

### CMSG_RESURRECT_RESPONSE (0x15C) — Accept/Decline Resurrection

**Packet structure:**
```
ObjectGuid  Resurrecter
uint8       Response        // 0 = reject, 1 = accept
```

**Pre-conditions:**
- Player must be dead
- Must have pending resurrection request

**Actions:**
- Response=0 → ClearResurrectRequestData()
- Response=1 → ResurrectUsingRequestData()

---

## 5. Server → Client Packets

### SMSG_PRE_RESURRECT (0x494)

```
PackedGuid  playerGUID
```
**Sent in:** BuildPlayerRepop() before ghost transition

---

### SMSG_DEATH_RELEASE_LOC (0x378)

```
uint32  mapId
float   x
float   y
float   z
```
**Sent in:** RepopAtGraveyard() (shows pin), ResurrectPlayer() (clears pin with map=-1)

---

### SMSG_CORPSE_RECLAIM_DELAY (0x269)

```
uint32  delayMs    // Delay in milliseconds
```
**Sent in:** BuildPlayerRepop(), KillPlayer(), SendCorpseReclaimDelay()

**Escalating delay:**
| Death # | Delay |
|---------|-------|
| 1st | 30 seconds |
| 2nd (within 5 min) | 60 seconds |
| 3rd+ (within 10 min) | 120 seconds |

---

### SMSG_RESURRECT_REQUEST (0x15B)

```
ObjectGuid  casterGUID
uint32      nameLength
char[]      casterName (null-terminated)
uint8       sicknessFlag     // 0 = player, 1 = creature (spirit healer)
uint32      delay            // Time until expires (0 = instant for no-res-timer)
```
**Sent in:** Spell::SendResurrectRequest()

---

### SMSG_AREA_SPIRIT_HEALER_TIME (0x2E4)

```
ObjectGuid  HealerGuid
int32       TimeLeft         // Milliseconds until next mass resurrect
```
**Sent in:** BattlegroundMgr::SendAreaSpiritHealerQueryOpcode()

**Calculation:** `TimeLeft = max(30000 - bg->GetLastResurrectTime(), 0)`

---

### SMSG_DURABILITY_DAMAGE_DEATH

```
(empty body — just opcode, no fields)
```
**Sent in:** SendDurabilityLoss()

---

### SMSG_UPDATE_OBJECT (0x0A9)

**Key fields updated during death:**

| Field | Death State | Value |
|-------|-------------|-------|
| UNIT_FIELD_FLAGS | JustDied | UNIT_FLAG_DEAD (0x00200000) |
| UNIT_DYNAMIC_FLAGS | Corpse | UNIT_DYNFLAG_NONE |
| PLAYER_FLAGS | Ghost | PLAYER_FLAGS_GHOST (0x10) |
| UNIT_FIELD_HEALTH | JustDied | 0 |
| UNIT_FIELD_HEALTH | Ghost | 1 |
| Power values | JustDied | 0 |
| PLAYER_FIELD_BYTES | Corpse | PLAYER_FIELD_BYTE_RELEASE_TIMER |

**On resurrection:**
- UNIT_FIELD_FLAGS → remove UNIT_FLAG_DEAD
- PLAYER_FLAGS → remove PLAYER_FLAGS_GHOST
- Health → restore percentage
- Mana → restore percentage
- Rage → set to 0

---

## 6. Spirit Healer System

### Spirit Healer vs Spirit Guide

| Type | NPC Flag | Behavior |
|------|----------|----------|
| Spirit Healer (outdoor) | UNIT_NPC_FLAG_SPIRITHEALER | Instant resurrect, gossip menu |
| Spirit Guide (BG/WG) | UNIT_NPC_FLAG_SPIRITGUIDE | Queue-based 30s mass resurrect |

### Spirit Healer Visibility (Creature.cpp)

```cpp
if (IsSpiritHealer() || IsSpiritGuide() || HasFlagsExtra(CREATURE_FLAG_EXTRA_GHOST_VISIBILITY))
{
    m_serverSideVisibility.SetValue(SERVERSIDE_VISIBILITY_GHOST, GHOST_VISIBILITY_GHOST);
    m_serverSideVisibilityDetect.SetValue(SERVERSIDE_VISIBILITY_GHOST, GHOST_VISIBILITY_GHOST);
}
```
Both are visible only to ghosts.

### Gossip Menu Flow

1. Ghost talks to Spirit Healer
2. `HandleGossipHelloOpcode()` → `PrepareGossipMenu()` + `SendPreparedGossip()`
3. Player clicks "Resurrect" option
4. Client sends `CMSG_SPIRIT_HEALER_ACTIVATE`
5. Server calls `SendSpiritResurrect()`

### BG Spirit Guide Flow

1. Player talks to Spirit Guide
2. `HandleGossipHelloOpcode()` → detects `IsSpiritGuide()`
3. `bg->AddPlayerToResurrectQueue(npcGUID, playerGUID)`
4. Player gets `SPELL_WAITING_FOR_RESURRECT` (spell 2584)
5. Every 30 seconds, BG mass-resurrects all queued players

---

## 7. Corpse & Bones

### CreateCorpse() (Player.cpp:4497)

```
1. SpawnCorpseBones() — convert existing corpse
2. Create Corpse object: CORPSE_RESURRECTABLE_PVP or CORPSE_RESURRECTABLE_PVE
3. Set appearance: CORPSE_FIELD_BYTES_1, CORPSE_FIELD_BYTES_2
4. Set flags: CORPSE_FLAG_UNK2, + CORPSE_FLAG_HIDE_HELM, + CORPSE_FLAG_HIDE_CLOAK
5. Set display ID, guild ID
6. Set equipment: CORPSE_FIELD_ITEM + i
7. AddToMap(corpse)
8. SaveToDB() (except in BG/arena)
```

### SpawnCorpseBones() (Player.cpp:4563)

```
1. _corpseLocation.WorldRelocate()
2. GetMap()->ConvertCorpseToBones(GetGUID())
3. Save to DB if not logging out
```

### ConvertCorpseToBones() (Map.cpp:4629)

```
1. Find corpse by player GUID
2. RemoveCorpse(corpse) from map
3. DeleteFromDB()
4. If config allows bones (CONFIG_DEATH_BONES_WORLD / CONFIG_DEATH_BONES_BG_OR_ARENA):
   - Create new Corpse object (bones)
   - Set CORPSE_FIELD_FLAGS = CORPSE_FLAG_UNK2 | CORPSE_FLAG_BONES
   - Clear all CORPSE_FIELD_ITEM slots
   - AddCorpse(bones) + AddToMap(bones)
5. Delete original corpse
```

---

## 8. Durability Loss

### Source 1: Combat Death (Unit.cpp)

```cpp
plrVictim->DurabilityLossAll(sWorld->getRate(RATE_DURABILITY_LOSS_ON_DEATH) / 100.0f, false);
plrVictim->SendDurabilityLoss();
```
- Equipment only (not inventory)
- Default: 10% of max durability

### Source 2: Spirit Healer Resurrect (NPCHandler.cpp)

```cpp
_player->DurabilityLossAll(0.25f, true);  // 25% ALL items including bags
```

### DurabilityLossAll() (Player.cpp:4581)

```
1. Equipment slots: EQUIPMENT_SLOT_START to EQUIPMENT_SLOT_END
2. If inventory=true:
   - Inventory items: INVENTORY_SLOT_ITEM_START to INVENTORY_SLOT_ITEM_END
   - Items inside bags: INVENTORY_SLOT_BAG_START to INVENTORY_SLOT_BAG_END
```

### DurabilityLoss() → DurabilityPointsLoss() (Player.cpp:4607, 4774)

```
1. Get maxDurability from item
2. Calculate loss = maxDurability * percent
3. If loss < 1, loss = 1
4. oldDur = GetUInt32Value(ITEM_FIELD_DURABILITY)
5. newDur = oldDur - loss (min 0)
6. If newDur == 0 && oldDur > 0 && item is equipped → _ApplyItemMods (remove stats)
7. SetUInt32Value(ITEM_FIELD_DURABILITY, newDur)
```

---

## 9. Resurrect Methods

### Method 1: Walk to Corpse (CMSG_RECLAIM_CORPSE)

```
1. ResurrectPlayer(0.5f)
   - Send SMSG_DEATH_RELEASE_LOC(map=-1) — clear minimap pin
   - Remove ghost aura (8326)
   - setDeathState(ALIVE)
   - SetHealth(maxHealth * 0.5)
   - SetPower(POWER_MANA, maxMana * 0.5)
   - SetPower(POWER_RAGE, 0)
   - SetPower(POWER_ENERGY, maxEnergy * 0.5)
   - UpdateObjectVisibility()
2. SpawnCorpseBones()
```

### Method 2: Spirit Healer (CMSG_SPIRIT_HEALER_ACTIVATE)

```
1. ResurrectPlayer(0.5f, true)  // applySickness=true
   - Same as Method 1 BUT:
   - CastSpell(15007)  // Resurrection Sickness
   - Duration: (level - 10) minutes (capped at 10 min for level 20+)
2. DurabilityLossAll(RATE_DURABILITY_LOSS_ON_SPIRIT_RESURRECT, true)
3. SpawnCorpseBones()
4. Teleport to corpse graveyard if different
```

### Method 3: Spell Resurrection (SMSG_RESURRECT_REQUEST)

```
1. Spell casts SetResurrectRequestData(caster, health, mana, aura)
2. Sends SMSG_RESURRECT_REQUEST with caster name
3. Client shows Accept/Decline dialog
4. Player sends CMSG_RESURRECT_RESPONSE with Response=1
5. ResurrectUsingRequestData():
   - TeleportTo(resurrect location)
   - ResurrectPlayer(0.0f, false)  // no sickness
   - SetHealth(resurrectHealth)     // flat value from spell
   - SetPower(POWER_MANA, resurrectMana)
   - SetPower(POWER_RAGE, 0)
   - SetFullPower(POWER_ENERGY)
   - SpawnCorpseBones()
```

### Method 4: BG Area Spirit Healer (queue-based)

```
1. Player talks to Spirit Guide
2. AddPlayerToResurrectQueue(npcGUID, playerGUID)
3. CastSpell(2584)  // SPELL_WAITING_FOR_RESURRECT
4. BG::ResurrectDeadPlayers() on timer (every 30 seconds):
   - ResurrectPlayer(1.0f)  // 100% health, no sickness
   - SpawnCorpseBones()
```

---

## 10. Corpse Reclaim Delay

### Constants

```cpp
#define DEATH_EXPIRE_STEP (5*MINUTE)    // 300 seconds
#define MAX_DEATH_COUNT 3
static uint32 corpseReclaimDelay[MAX_DEATH_COUNT] = { 30, 60, 120 }; // seconds
```

### GetCorpseReclaimDelay() (Player.cpp)

```cpp
uint32 Player::GetCorpseReclaimDelay(bool pvp) const
{
    if (pvp && !CONFIG_DEATH_CORPSE_RECLAIM_DELAY_PVP)
        return corpseReclaimDelay[0];  // 30s
    if (!pvp && !CONFIG_DEATH_CORPSE_RECLAIM_DELAY_PVE)
        return 0;  // instant in PvE when disabled

    time_t now = GameTime::GetGameTime().count();
    uint64 count = (now < m_deathExpireTime - 1) ? (m_deathExpireTime - 1 - now) / DEATH_EXPIRE_STEP : 0;
    return corpseReclaimDelay[count];
}
```

### UpdateCorpseReclaimDelay() (Player.cpp)

Called in KillPlayer(). Tracks rapid deaths:
```cpp
void Player::UpdateCorpseReclaimDelay()
{
    time_t now = GameTime::GetGameTime().count();
    if (now < m_deathExpireTime)
    {
        uint64 count = (m_deathExpireTime - now) / DEATH_EXPIRE_STEP + 1;
        if (count < MAX_DEATH_COUNT)
            m_deathExpireTime = now + (count + 1) * DEATH_EXPIRE_STEP;
        else
            m_deathExpireTime = now + MAX_DEATH_COUNT * DEATH_EXPIRE_STEP;
    }
    else
        m_deathExpireTime = now + DEATH_EXPIRE_STEP;
}
```

---

## 11. Constants

### Ghost Aura

| Spell ID | Aura Type | Effect |
|----------|-----------|--------|
| 8326 | SPELL_AURA_GHOST (95) | Sets PLAYER_FLAGS_GHOST, ghost visibility |
| 20584 | (Night Elf racial) | Ghost speed bonus |
| 27827 | (Spirit of Redemption) | Priest talent auto-res |
| 15007 | (Resurrection Sickness) | Applied on spirit healer resurrect |
| 2584 | (Waiting for Resurrect) | BG queue aura |

### Resurrection Sickness Duration

| Level | Duration |
|-------|----------|
| 1-10 | No sickness |
| 11-19 | (Level - 10) minutes |
| 20+ | 10 minutes |

### Config Keys

| Config | Purpose | Default |
|--------|---------|---------|
| RATE_DURABILITY_LOSS_ON_DEATH | % durability lost on death | 10.0 |
| RATE_DURABILITY_LOSS_ON_SPIRIT_RESURRECT | % durability on spirit rez | 25.0 |
| CONFIG_DEATH_CORPSE_RECLAIM_DELAY_PVP | Enable delay in PvP | true |
| CONFIG_DEATH_CORPSE_RECLAIM_DELAY_PVE | Enable delay in PvE | false |
| CONFIG_DEATH_SICKNESS_LEVEL | Min level for sickness | 10 |
| CONFIG_DEATH_BONES_BG_OR_ARENA | Spawn bones in BG | true |
| CONFIG_DEATH_BONES_WORLD | Spawn bones in world | true |
| CONFIG_DURABILITY_LOSS_IN_PVP | Durability in PvP | false |

---

## 12. Summit Implementation Status

### Implemented

| Feature | Status | File |
|---------|--------|------|
| HandleRepopRequest | ✅ | death.go:60 |
| HandleReclaimCorpse | ✅ | death.go:86 |
| HandleResurrectResponse | ✅ | death.go:117 |
| HandleSpiritHealerActivate | ✅ | death.go:138 |
| GetClosestGraveyard | ✅ | death.go:28 |
| SetHealth broadcasts | ✅ | player/player.go |
| SetPower broadcasts | ✅ | player/player.go |

### Missing / Incomplete

| Feature | Status | Notes |
|---------|--------|-------|
| SMSG_PRE_RESURRECT | ❌ | Must send before ghost transition |
| SMSG_DEATH_RELEASE_LOC | ❌ | Spirit healer minimap pin |
| SMSG_CORPSE_RECLAIM_DELAY | ✅ | Sent in HandleRepopRequest |
| SMSG_RESURRECT_REQUEST | ❌ | Spell-based resurrection |
| SMSG_DURABILITY_DAMAGE_DEATH | ❌ | Empty packet, just opcode |
| Ghost aura (8326) | ❌ | PLAYER_FLAGS_GHOST not set |
| CreateCorpse | ❌ | Corpse object not created |
| SpawnCorpseBones | ❌ | Corpse→bones conversion |
| DurabilityLossAll | ❌ | No durability system yet |
| Resurrection Sickness | ❌ | No spell 15007 cast |
| Corpse reclaim delay escalation | ❌ | No death count tracking |
| Spirit healer gossip menu | ❌ | No gossip menu shown |
| BG spirit guide queue | ❌ | No queue system |
| Ghost water walking | ❌ | Not implemented |
| Ghost animation (ALWAYS_STAND) | ❌ | Not implemented |
