# Map Management, AreaTriggers, and WorldStates Re-implementation Plan

## Overview

Re-implement map management, areatriggers, and world states from AzerothCore (C++) to Summit (Go).

## Source Analysis (AzerothCore)

### Map Management
- **MapMgr** (`src/server/game/Maps/MapMgr.h/.cpp`): Singleton managing all maps
  - `CreateBaseMap(mapId)`: Creates/returns base map
  - `FindMap(mapId, instanceId)`: Finds specific instance
  - `CreateMap(mapId, player)`: Creates map for player
  - `Update(diff)`: Updates all maps
  - `SetMapUpdateInterval(t)`: Configures update interval

- **Map** (`src/server/game/Maps/Map.h/.cpp`): Base map class
  - Grid-based system with `Cell`, `GridCoord`
  - `Visit()`: Iterates objects in cells
  - `Update()`: Processes sessions, objects, respawns, scripts
  - `GetHeight()`, `GetWaterOrGroundLevel()`: Terrain queries

- **MapInstanced**: For instanced content (dungeons/raids)
- **InstanceMap**: Dungeon/raid instances with reset logic
- **BattlegroundMap**: BG/Arena instances

### AreaTriggers
- **Data** (`areatrigger` table): entry, map, x, y, z, radius, dimensions
- **Loading** (`ObjectMgr::LoadAreaTriggers`): Loads from DB into `_areaTriggerStore`
- **Scripts** (`AreaTriggerScript`): Virtual `OnTrigger()` method
- **OnlyOnceAreaTriggerScript**: One-time triggers with instance tracking
- **Teleports** (`areatrigger_teleport` table): Target map/position

### WorldStates
- **Storage** (`worldstates` table): entry → value mapping
- **WorldState class**: Global singleton
  - `setWorldState(index, value)`: Persists to DB
  - `getWorldState(index)`: Reads value
  - `LoadWorldStates()`: Loads all on startup
  - `SendWorldstateUpdate()`: Sends to players

## Target Architecture (Summit)

### Current State
- No map management system
- Flat `SpawnManager` for NPCs
- `GameObjectManager` for game objects
- DBC files parsed (MapEntry, AreaTableEntry, etc.)

### Proposed Structure

```
pkg/summit/world/
├── map/
│   ├── map.go           # Map struct and core logic
│   ├── mapmgr.go        # MapManager singleton
│   ├── mapinstance.go   # Instanced maps
│   └── grid.go          # Grid/cell system (future)
├── areatrigger/
│   ├── areatrigger.go   # AreaTrigger struct
│   ├── manager.go       # Loading and lookup
│   └── scripts.go       # Script interface
└── worldstate/
    ├── worldstate.go    # WorldState struct
    └── manager.go       # Global state management
```

## Implementation Plan

### Phase 1: Map Management (Core)

#### 1.1 Map Struct
```go
// pkg/summit/world/map/map.go
type Map struct {
    ID          uint32
    InstanceID  uint32
    SpawnMode   uint8
    Parent      *Map
    
    Name        string
    Entry       *wotlk.MapEntry
    
    Players     map[uint32]*player.Player
    NPCs        map[uint32]*NPC
    GameObjects map[uint32]*GameObject
    
    mutex       sync.RWMutex
}

func (m *Map) Update(diff uint32) { ... }
func (m *Map) AddPlayer(p *player.Player) { ... }
func (m *Map) RemovePlayer(guid uint32) { ... }
func (m *Map) GetPlayers() []*player.Player { ... }
func (m *Map) GetHeight(x, y, z float32) float32 { ... }
```

#### 1.2 MapManager
```go
// pkg/summit/world/map/mapmgr.go
type MapManager struct {
    maps        map[uint32]*Map
    instances   map[uint32]map[uint32]*Map  // mapId → instanceId → Map
    nextInstID  uint32
    mutex       sync.RWMutex
}

var instance *MapManager
var once sync.Once

func GetMapManager() *MapManager { ... }
func (mm *MapManager) CreateBaseMap(mapId uint32) *Map { ... }
func (mm *MapManager) FindMap(mapId, instanceId uint32) *Map { ... }
func (mm *MapManager) CreateMap(mapId uint32, p *player.Player) *Map { ... }
func (mm *MapManager) Update(diff uint32) { ... }
```

#### 1.3 InstanceMap
```go
// pkg/summit/world/map/mapinstance.go
type InstanceMap struct {
    Map
    Difficulty    uint8
    ResetTime     time.Time
    InstanceData  interface{}
}

func (im *InstanceMap) CanEnter(p *player.Player) error { ... }
```

### Phase 2: AreaTriggers

#### 2.1 AreaTrigger Struct
```go
// pkg/summit/world/areatrigger/areatrigger.go
type AreaTrigger struct {
    Entry       uint32
    MapID       uint32
    X, Y, Z     float32
    Radius      float32
    Length      float32
    Width       float32
    Height      float32
    Orientation float32
}

type AreaTriggerTeleport struct {
    TargetMapID       uint32
    TargetX, TargetY, TargetZ float32
    TargetO           float32
}
```

#### 2.2 Manager
```go
// pkg/summit/world/areatrigger/manager.go
type Manager struct {
    triggers      map[uint32]*AreaTrigger
    teleports     map[uint32]*AreaTriggerTeleport
    scripts       map[uint32]AreaTriggerScript
}

func (m *Manager) LoadFromDB(db *sql.DB) error { ... }
func (m *Manager) GetTrigger(entry uint32) *AreaTrigger { ... }
func (m *Manager) CheckTrigger(p *player.Player, entry uint32) bool { ... }
func (m *Manager) RegisterScript(entry uint32, script AreaTriggerScript) { ... }
```

#### 2.3 Script Interface
```go
// pkg/summit/world/areatrigger/scripts.go
type AreaTriggerScript interface {
    OnTrigger(player *player.Player, trigger *AreaTrigger) bool
}

type OnlyOnceAreaTriggerScript interface {
    AreaTriggerScript
    OnFirstTrigger(player *player.Player, trigger *AreaTrigger) bool
}
```

### Phase 3: WorldStates

#### 3.1 WorldState Struct
```go
// pkg/summit/world/worldstate/worldstate.go
type WorldState struct {
    states     map[uint32]uint64
    mutex      sync.RWMutex
}

var instance *WorldState
var once sync.Once

func GetWorldState() *WorldState { ... }
func (ws *WorldState) LoadFromDB(db *sql.DB) error { ... }
func (ws *WorldState) SetState(index uint32, value uint64) { ... }
func (ws *WorldState) GetState(index uint32) uint64 { ... }
func (ws *WorldState) SendToPlayer(p *player.Player, index uint32) { ... }
func (ws *WorldState) BroadcastToAll(index uint32, value uint32) { ... }
```

#### 3.2 Packets
```go
// pkg/summit/world/packets/worldstate.go
func SendInitWorldStates(p *player.Player, mapID, zoneID, areaID uint32, states map[uint32]int32) { ... }
func SendUpdateWorldState(p *player.Player, variableID, value int32) { ... }
```

## Database Schema

### New Tables (World DB)

```sql
-- AreaTrigger definitions
CREATE TABLE areatrigger (
    entry INT UNSIGNED NOT NULL,
    map INT UNSIGNED NOT NULL,
    x FLOAT NOT NULL,
    y FLOAT NOT NULL,
    z FLOAT NOT NULL,
    radius FLOAT NOT NULL DEFAULT 0,
    length FLOAT NOT NULL DEFAULT 0,
    width FLOAT NOT NULL DEFAULT 0,
    height FLOAT NOT NULL DEFAULT 0,
    orientation FLOAT NOT NULL DEFAULT 0,
    PRIMARY KEY (entry)
);

-- AreaTrigger teleports
CREATE TABLE areatrigger_teleport (
    ID INT UNSIGNED NOT NULL,
    target_map INT UNSIGNED NOT NULL,
    target_position_x FLOAT NOT NULL,
    target_position_y FLOAT NOT NULL,
    target_position_z FLOAT NOT NULL,
    target_orientation FLOAT NOT NULL DEFAULT 0,
    PRIMARY KEY (ID)
);

-- WorldStates (character DB)
CREATE TABLE worldstates (
    entry INT UNSIGNED NOT NULL,
    value INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (entry)
);
```

## Integration Points

### Server.go Changes
```go
type Server struct {
    // ... existing fields
    mapManager     *mapmanager.MapManager
    areatriggerMgr *areatrigger.Manager
    worldState     *worldstate.WorldState
}

func (s *Server) Init() error {
    s.mapManager = mapmanager.GetMapManager()
    s.areatriggerMgr = areatrigger.NewManager()
    s.worldState = worldstate.GetWorldState()
    
    // Load data
    s.areatriggerMgr.LoadFromDB(s.worldStore.DB())
    s.worldState.LoadFromDB(s.charStore.DB())
    
    return nil
}
```

### Updater.go Changes
```go
func (u *Updater) Update(diff uint32) {
    // Update all maps
    s.mapManager.Update(diff)
}
```

### Movement Handler Changes
```go
func (gc *WorldSession) HandleMovementOpcodes(opcode wow.OpCode, data []byte) {
    // ... existing movement handling
    
    // Check areatriggers after position update
    gc.checkAreaTriggers()
}

func (gc *WorldSession) checkAreaTriggers() {
    map := gc.player.GetMap()
    if map == nil { return }
    
    // Check if player entered any areatrigger
    gc.areatriggerMgr.CheckPlayerTriggers(gc.player, map)
}
```

## Testing Strategy

1. **Unit Tests**: Each component with mock DB
2. **Integration Tests**: Map updates with mock players
3. **Manual Testing**: Login, move between maps, verify triggers

## Migration Notes

- NPC spawns currently in `SpawnManager` → move to map-based storage
- Game objects similarly → integrate with map system
- Session player tracking → use map's player list

## Files to Create

1. `pkg/summit/world/map/map.go`
2. `pkg/summit/world/map/mapmgr.go`
3. `pkg/summit/world/map/mapinstance.go`
4. `pkg/summit/world/areatrigger/areatrigger.go`
5. `pkg/summit/world/areatrigger/manager.go`
6. `pkg/summit/world/areatrigger/scripts.go`
7. `pkg/summit/world/worldstate/worldstate.go`
8. `pkg/summit/world/worldstate/manager.go`

## Files to Modify

1. `pkg/summit/world/server.go` - Initialize managers
2. `pkg/summit/world/updater.go` - Call map updates
3. `pkg/summit/world/handlers.go` - Add areatrigger checks
4. `pkg/summit/world/worldsession.go` - Player-map association
5. `pkg/summit/world/npc.go` - Move to map-based storage
6. `pkg/summit/world/gameobject.go` - Move to map-based storage

## Estimated Effort

- Phase 1 (Map Core): 3-4 hours
- Phase 2 (AreaTriggers): 2-3 hours  
- Phase 3 (WorldStates): 1-2 hours
- Integration & Testing: 2-3 hours

**Total: ~10-12 hours**
