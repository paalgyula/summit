# SMSG_UPDATE_OBJECT — Implementation Status

This document tracks the implementation status of the WoW 3.3.5a update object
system in Summit, compared against AzerothCore as the reference implementation.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SMSG_UPDATE_OBJECT                           │
│                                                                     │
│  uint32 blockCount                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ Block 0: UPDATETYPE_OUT_OF_RANGE_OBJECTS (if any)            │  │
│  │   uint8  type = 4                                             │  │
│  │   uint32 count                                                │  │
│  │   packedGuid[]                                                │  │
│  ├───────────────────────────────────────────────────────────────┤  │
│  │ Block 1..N: Update blocks                                     │  │
│  │   uint8  type (0=Values, 1=Movement, 2=Create, 3=Create2)    │  │
│  │   [type-specific data...]                                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

## Implemented ✅

### Core Infrastructure

| Component | File | Status |
|-----------|------|--------|
| `UpdateMask` — bit-field tracking | `object/update_mask.go` | ✅ Complete |
| `Object.values[]` — field storage | `object/object.go` | ✅ Complete |
| `UpdateFields` — WotLK 3.3.5a field constants | `object/update_fields.gen.go` | ✅ Complete (generated) |
| `ObjectUpdateFlags` — 10 flag constants | `wow/update.go` | ✅ Complete |
| `ObjectUpdateType` — 6 update types | `wow/update.go` | ✅ Complete |
| `UpdateData` — multi-block accumulator | `object/update_data.go` | ✅ Complete |
| `UpdateBlockBuffer` — block builder | `object/update_block_buffer.go` | ✅ Complete |
| `GetUpdateBlockCount()` bug fix | `object/update_mask.go` | ✅ Fixed (was returning bytes not blocks) |

### Change Tracking

| Feature | File | Status |
|---------|------|--------|
| `_changesMask` on Object | `object/object.go` | ✅ Tracks which fields changed |
| `SetUInt32Value` change tracking | `object/object.go` | ✅ Only sets bit when value differs |
| `SetFloatValue` change tracking | `object/object.go` | ✅ |
| `SetInt32Value` change tracking | `object/object.go` | ✅ Delegates to SetUInt32Value |
| `SetByteValue` change tracking | `object/object.go` | ✅ |
| `HasChanges()` | `object/object.go` | ✅ |
| `ClearChanges()` | `object/object.go` | ✅ |
| `_fieldNotifyFlags` | `object/object.go` | ✅ (stored, not yet used in filtering) |

### Visibility System

| Feature | File | Status |
|---------|------|--------|
| `UFFlagPublic/Private/Owner/...` constants | `object/update_field_flags.go` | ✅ 9 flag types |
| `UnitUpdateFieldFlags[PlayerEnd-ObjectEnd]` | `object/update_field_flags.go` | ✅ Full player field coverage |
| `ItemUpdateFieldFlags[ContainerEnd-ObjectEnd]` | `object/update_field_flags.go` | ✅ |
| `GameObjectUpdateFieldFlags` | `object/update_field_flags.go` | ✅ |
| `DynamicObjectUpdateFieldFlags` | `object/update_field_flags.go` | ✅ |
| `CorpseUpdateFieldFlags` | `object/update_field_flags.go` | ✅ |
| `getFieldFlags()` per-type dispatch | `object/object.go` | ✅ |
| `BuildFilteredUpdateMask(target, isSelf)` | `object/object.go` | ✅ CREATE: all visible fields |
| `BuildIncrementalUpdateMask(target)` | `object/object.go` | ✅ VALUES: changed + visible fields |

### Update Packets

| Feature | File | Status |
|---------|------|--------|
| `BuildCreateObject(source, target)` | `updater.go` | ✅ Visibility-aware CreateObject/CreateObject2 |
| `BuildValuesUpdateObject(source, target)` | `updater.go` | ✅ Incremental values-only update |
| `BuildDestroyObject(player, onDeath)` | `updater.go` | ✅ SMSG_DESTROY_OBJECT |
| `BuildNPCCreateObject(npc)` | `updater.go` | ✅ Full NPC create block |
| `BuildGameObjectCreateObject(gobj)` | `updater.go` | ✅ Full game object create block |
| `BuildItemCreateObject(item, target)` | `updater.go` | ✅ Visibility-aware item create |
| `BuildInventoryUpdate(player)` | `updater.go` | ✅ Inventory slot values update |
| `UpdateData.BuildPacket()` | `object/update_data.go` | ✅ Multi-block with out-of-range |

### Movement Update

| Feature | File | Status |
|---------|------|--------|
| `UPDATEFLAG_LIVING` — movement flags + time | `updater.go` | ✅ |
| `UPDATEFLAG_STATIONARY_POSITION` — X/Y/Z/O | `updater.go` | ✅ Uses player's Location |
| Unit speeds (walk/run/swim/flight/turn) | `updater.go` | ✅ All 8 speed types |
| `UPDATEFLAG_LOWGUID` — per-type low word | `updater.go` | ✅ Player/Unit/Object types |
| Transport movement flag handling | `updater.go` | ✅ Strips for units, sets for players on transport |

### Login Flow Integration

| Packet | Function | Status |
|--------|----------|--------|
| Player self-create | `sendPlayerCreate()` | ✅ Uses `BuildCreateObject(p, p)` |
| Other player create | `sendCreateObjectForPlayer()` | ✅ Uses `BuildCreateObject(source, target)` |
| NPC create | `sendCreateObjectForNPC()` | ✅ Uses `BuildNPCCreateObject()` |
| GameObject create | `sendCreateObjectForGameObject()` | ✅ Uses `BuildGameObjectCreateObject()` |
| Health update broadcast | `broadcastHealthUpdate()` | ✅ Uses `BuildValuesUpdateObject()` |

### Bug Fixes

| Bug | Fix |
|-----|-----|
| `GetUpdateBlockCount()` returned bytes not blocks | Rewrote to compute from last non-zero byte / 4 |
| `HasChanges()` only checked first `BlockCount()` bytes | Now iterates all mask bytes |
| `BuildValuesUpdateBlock()` missing uint8 blockCount prefix | Added prefix to match AzerothCore wire format |
| `UpdateFlagHighGUID` undefined (0x0008 ≠ HIGHGUID) | Removed — 0x0008 is `UPDATEFLAG_UNKNOWN` in 3.3.5a |
| `UpdateFlagHasPosition` undefined | Replaced with `UpdateFlagStationaryPosition` |

---

## Not Implemented / Missing ❌

### Movement Update — Incomplete Sections

These sections are now structurally present in `updater.go` but use placeholder values:

| Feature | Flag/Condition | Notes |
|---------|---------------|-------|
| Transport data (GUID + offsets) | `MOVEMENTFLAG_ONTRANSPORT` | Structure written, TODO: real transport GUID + offsets |
| Swimming/flying pitch | `MOVEMENTFLAG_SWIMMING \| MOVEMENTFLAG_FLYING2` | Structure written, TODO: read actual pitch |
| Fall time | Always for players | Structure written, TODO: read actual fall time |
| Falling data (velocity, angles) | `MOVEMENTFLAG_FALLING` | Structure written, TODO: read actual fall data |
| Spline elevation | `MOVEMENTFLAG_SPLINE_ELEVATION` | Structure written, TODO: read actual elevation |
| Spline data | `MOVEMENTFLAG_SPLINE_ENABLED` | Structure written, TODO: read from MoveSpline |
| `UPDATEFLAG_HAS_TARGET` (0x4) | When unit has victim | Structure written, TODO: pack victim GUID |
| `UPDATEFLAG_TRANSPORT` (0x2) | For transport game objects | Structure written, TODO: path progress |
| `UPDATEFLAG_VEHICLE` (0x80) | For vehicle units | Structure written, TODO: vehicle ID |
| `UPDATEFLAG_ROTATION` (0x200) | For game objects | Structure written, TODO: packed rotation |
| `UPDATEFLAG_UNKNOWN` (0x8) | If set | ✅ Writes uint32(0) |
| `UPDATEFLAG_POSITION` (0x100) | Transport-attached position | ✅ Implemented |

### Update System — Remaining Missing Features

| Feature | Description | Status |
|---------|-------------|--------|
| **Object update queue** | Periodic flush of changed objects to players | ✅ Implemented (`Map.updateObjects`) |
| **`AddToObjectUpdateIfNeeded()`** | Object self-queues when a field changes | ✅ Implemented |
| **`ClearUpdateMask()`** | After sending updates, clear changes mask | ✅ Implemented |
| **Visibility-based create decisions** | CREATE_OBJECT vs CREATE_OBJECT2 | ✅ Implemented (self, GO types, DynamicObject, Corpse) |
| **`_fieldNotifyFlags` filtering** | Fields always included regardless of changes | ✅ Implemented |
| **Out-of-range removal** | Detect when players leave range | ✅ Implemented (`VisibilityTracker`, `UpdatePlayerVisibility`) |
| **`SMSG_DESTROY_OBJECT` integration** | Hook into logout, death | ✅ Implemented |
| **Dynamic flags** | `UNIT_DYNAMIC_FLAGS` and `GAMEOBJECT_DYNAMIC` | ❌ Not implemented |
| **`PARTY_MEMBER` visibility** | Fields visible only to party/raid | ❌ No party system |
| **`SPECIAL_INFO` visibility** | Fields visible with empathy aura | ❌ Not implemented |

### Object Types — Update Support

| Type | Create | Values | Movement | Destroy |
|------|--------|--------|----------|---------|
| Player | ✅ | ✅ | ✅ | ✅ |
| Unit/Creature | ✅ | ✅ | ✅ | ❌ (no corpse handling) |
| Item | ✅ | ✅ | N/A | ✅ |
| Container | ✅ (struct added) | ❌ (no fields) | N/A | ❌ |
| GameObject | ✅ | ✅ | ✅ (stationary) | ❌ |
| DynamicObject | ✅ (struct added) | ❌ (no fields) | N/A | ❌ |
| Corpse | ✅ (struct added) | ❌ (no fields) | N/A | ❌ |

### Visibility System ✅

| Component | File | Status |
|-----------|------|--------|
| `VisibilityTracker` | `map/visibility.go` | ✅ Tracks which objects each player sees |
| `GetVisibilityRange()` | `map/visibility.go` | ✅ Returns visibility range for objects |
| `GetSightRange()` | `map/visibility.go` | ✅ Returns sight range for players |
| `Distance2DPositions()` | `map/visibility.go` | ✅ Calculates 2D distance between positions |
| `UpdatePlayerVisibility()` | `map/map.go` | ✅ Checks range, sends create/destroy |
| `UpdateObjectVisibility()` | `map/map.go` | ✅ Notifies players about object changes |
| `Map.Update()` | `map/map.go` | ✅ Calls visibility update each tick |
| `Map.visibilityTracker` | `map/map.go` | ✅ Per-map visibility state |

### Specific Missing Packets

| Opcode | Name | Status |
|--------|------|--------|
| `SMSG_UPDATE_OBJECT` | Main update | ✅ Implemented |
| `SMSG_DESTROY_OBJECT` | Destroy | ✅ Built, not integrated |
| `SMSG_TRANSFER_PENDING` | Teleport pending | ❌ Not implemented |
| `SMSG_MOVE_*` | Movement opcodes | ❌ Separate system (not part of UPDATETYPE_MOVEMENT) |

---

## Wire Format Reference

### CreateObject Block (Type 2/3)

```
uint8   updateType          (2=CreateObject, 3=CreateObject2)
packed  guid
uint8   objectTypeID        (0=Object, 1=Item, 3=Unit, 4=Player, 5=GameObject...)
uint16  updateFlags         (bitmask of ObjectUpdateFlags)
[movement update data]      (varies by flags)
[values update]             (uint8 blockCount + mask + values)
```

### Values Update Block

```
uint8   blockCount          (number of uint32 mask words)
uint32[] updateMask         (1 bit per field, little-endian)
uint32[] values             (only for set bits in mask, in order)
```

### Movement Update — UPDATEFLAG_LIVING (0x20)

```
uint32  moveFlags           (MovementFlag bitmask)
uint8   extraMoveFlags      (always 0 for now)
uint32  time                (millisecond timestamp)
[transport data]            (if MOVEMENTFLAG_ONTRANSPORT)
[pitch]                     (if SWIMMING or FLYING2)
uint32  fallTime            (for players)
[fall data]                 (if MOVEMENTFLAG_FALLING: 4 floats)
[splineElevation]           (if MOVEMENTFLAG_SPLINE_ELEVATION: 1 float)
float32 walkSpeed
float32 runSpeed
float32 runBackSpeed
float32 swimSpeed
float32 swimBackSpeed
float32 flightSpeed
float32 flightBackSpeed
float32 turnRate
[spline data]               (if MOVEMENTFLAG_SPLINE_ENABLED)
```

### Movement Update — UPDATEFLAG_STATIONARY_POSITION (0x40)

```
float32 x
float32 y
float32 z
float32 orientation
```

### Update Flags Reference

| Bit | Flag | Description |
|-----|------|-------------|
| 0x0001 | `UpdateFlagSelf` | Object is the viewer (self) |
| 0x0002 | `UpdateFlagTransport` | Object is on a transport |
| 0x0004 | `UpdateFlagHasTarget` | Unit has an attack target |
| 0x0008 | `UpdateFlagUnknown` | Unknown — writes uint32(0) |
| 0x0010 | `UpdateFlagLowGUID` | Include low GUID word |
| 0x0020 | `UpdateFlagLiving` | Living movement data (speeds, flags) |
| 0x0040 | `UpdateFlagStationaryPosition` | Static position (X/Y/Z/O) |
| 0x0080 | `UpdateFlagVehicle` | Vehicle unit |
| 0x0100 | `UpdateFlagPosition` | Transport-attached position |
| 0x0200 | `UpdateFlagRotation` | Game object rotation |

---

## Testing

### Test Files

- `object/changes_test.go` — 16 tests for change tracking, UpdateMask, field notify
- `object/visibility_test.go` — 5 tests for field visibility flags
- `object/update_data_test.go` — 15 tests for UpdateData, filtered masks, binary format
- `object/object_test.go` — Existing packet parsing test

### Running Tests

```bash
go test ./pkg/summit/world/object/... -v    # Object + update system tests
go test ./...                                # Full suite (only serworm fails pre-existing)
```

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `pkg/summit/world/object/object.go` | Object struct, values storage, change tracking, filtered masks, AddToObjectUpdateIfNeeded |
| `pkg/summit/world/object/update_mask.go` | Bit-field mask for tracking changed/visible fields |
| `pkg/summit/world/object/update_fields.gen.go` | Generated WotLK 3.3.5a field constants |
| `pkg/summit/world/object/update_field_flags.go` | Per-field visibility flag arrays |
| `pkg/summit/world/object/update_data.go` | Multi-block SMSG_UPDATE_OBJECT accumulator |
| `pkg/summit/world/object/update_block_buffer.go` | Byte buffer for building raw update blocks |
| `pkg/summit/world/object/gameobject.go` | GameObject struct with proper type flags |
| `pkg/summit/world/object/container.go` | Container struct |
| `pkg/summit/world/object/dynamic_object.go` | DynamicObject struct |
| `pkg/summit/world/object/corpse.go` | Corpse struct |
| `pkg/summit/world/updater.go` | High-level packet builders (Create, Values, Destroy, NPC, GO, Movement) |
| `pkg/summit/world/login.go` | Login flow — sends create objects to players |
| `pkg/summit/world/map/map.go` | Map update queue, visibility updates, SendObjectUpdates |
| `pkg/summit/world/map/visibility.go` | VisibilityTracker, range calculation, distance helpers |
| `pkg/summit/world/object/player/player.go` | Player struct with PacketSender interface |
| `pkg/wow/update.go` | ObjectUpdateType and ObjectUpdateFlags constants |
| `pkg/wow/movement.go` | MovementFlag and MoveType constants |
