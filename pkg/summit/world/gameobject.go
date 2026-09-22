package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// GameObjectType mirrors the C++ GameobjectTypes enum.
type GameObjectType int

const (
	GameObjectTypeDoor GameObjectType = iota
	GameObjectTypeButton
	GameObjectTypeQuestGiver
	GameObjectTypeChest
	GameObjectTypeBinder
	GameObjectTypeGeneric
	GameObjectTypeTrap
	GameObjectTypeChair
	GameObjectTypeSpellFocus
	GameObjectTypeText
	GameObjectTypeGoober
	GameObjectTypeTransport
	GameObjectTypeAreaDamage
	GameObjectTypeCamera
	GameObjectTypeMapObject
	GameObjectTypeMoTransport
	GameObjectTypeDuelArbiter
	GameObjectTypeFishingNode
	GameObjectTypeSummoningRitual
	GameObjectTypeMailbox
	GameObjectTypeAuctionHouse
	GameObjectTypeGuardPost
	GameObjectTypeSpellcaster
	GameObjectTypeMeetingStone
	GameObjectTypeFlagStand
	GameObjectTypeFishingHole
	GameObjectTypeFlagDrop
	GameObjectTypeMiniGame
	GameObjectTypeLotteryKiosk
	GameObjectTypeCapturePoint
	GameObjectTypeAuraGenerator
	GameObjectTypeDungeonDifficulty
	GameObjectTypeBarberChair
	GameObjectTypeDestructibleBuilding
	GameObjectTypeGuildBank
)

// GOState mirrors the C++ GOState enum.
type GOState uint8

const (
	GOStateReady   GOState = 0
	GOStateActive  GOState = 1
	GOStateLocked  GOState = 2
)

// GameObjectTemplate wraps the basedata template with convenience accessors.
type GameObjectTemplate struct {
	*basedata.GameObjectTemplate
}

// Type returns the game object type as the local enum.
func (t *GameObjectTemplate) Type() GameObjectType {
	return GameObjectType(t.GameObjectTemplate.Type)
}

// GetFlags returns 0 — flags are not stored in basedata yet.
// TODO: extend basedata.GameObjectTemplate with Flags, Faction fields from AC.
func (t *GameObjectTemplate) GetFlags() uint32 { return 0 }

// GetFaction returns 0 — faction is not stored in basedata yet.
func (t *GameObjectTemplate) GetFaction() uint32 { return 0 }

// GameObjectData wraps a basedata.GameObjectSpawn for use by the world server.
type GameObjectData struct {
	*basedata.GameObjectSpawn
}

// GameObject represents a spawned game object in the world.
type GameObject struct {
	*object.Object

	ID       uint32 // spawn ID
	Entry    uint32
	Name     string
	Type     GameObjectType
	DisplayID uint32
	Faction  uint32
	Flags    uint32
	Size     float32

	X, Y, Z, O float32
	Map         uint32

	GOState      GOState
	AnimProgress uint32

	Template *GameObjectTemplate

	// Chest state
	Looted      bool
	RespawnTime int64 // when to respawn (Unix ms), 0 = never
}

// NewGameObject creates a new game object from template and spawn data.
func NewGameObject(template *GameObjectTemplate, data *GameObjectData) *GameObject {
	g := &GameObject{
		Object:       object.NewObject(),
		ID:           data.GUID,
		Entry:        data.Entry,
		Name:         template.Name,
		Type:         template.Type(),
		DisplayID:    template.DisplayID,
		Faction:      template.GetFaction(),
		Flags:        template.GetFlags(),
		Size:         template.Size,
		X:            data.PosX,
		Y:            data.PosY,
		Z:            data.PosZ,
		O:            data.O,
		Map:          data.MapID,
		GOState:      GOState(data.State),
		AnimProgress: uint32(data.AnimProgress),
		Template:     template,
	}

	g.init()

	return g
}

// init sets up the game object's update fields.
func (g *GameObject) init() {
	g.Object.InitValues(int(object.GameobjectEnd))
	g.Object.SetObjectTypeID(wow.TypeIDGameObject)

	// GUID
	guid := wow.NewGUID(wow.GameObjectGUID, g.ID)
	g.Object.SetGUID(guid)

	// Object fields
	g.Object.SetUInt32Value(object.ObjectFieldGuid, uint32(guid))
	g.Object.SetUInt32Value(object.ObjectFieldGuid+1, uint32(uint64(guid)>>32))
	g.Object.SetUInt32Value(object.ObjectFieldType, uint32(wow.TypeIDGameObject))
	g.Object.SetUInt32Value(object.ObjectFieldEntry, g.Entry)
	g.Object.SetFloatValue(object.ObjectFieldScaleX, g.Size)

	// GameObject fields
	g.Object.SetUInt32Value(object.GameobjectDisplayid, g.DisplayID)
	g.Object.SetUInt32Value(object.GameobjectFaction, g.Faction)
	g.Object.SetUInt32Value(object.GameobjectFlags, g.Flags)
	g.Object.SetUInt32Value(object.GameobjectLevel, 1)

	// GAMEOBJECT_BYTES_1: state(0) | type(1) | artkit(2) | animprogress(3)
	g.Object.SetByteValue(object.GameobjectBytes_1, 0, byte(g.GOState))
	g.Object.SetByteValue(object.GameobjectBytes_1, 1, byte(g.Type))
	g.Object.SetByteValue(object.GameobjectBytes_1, 2, 0) // artkit
	g.Object.SetByteValue(object.GameobjectBytes_1, 3, byte(g.AnimProgress))
}

// GetGUID returns the game object's GUID.
func (g *GameObject) GetGUID() wow.GUID {
	return wow.NewGUID(wow.GameObjectGUID, g.ID)
}

// GetPosition returns the game object's world location (satisfies Map.AddGameObject positionProvider).
func (g *GameObject) GetPosition() *player.WorldLocation {
	return &player.WorldLocation{
		X:   g.X,
		Y:   g.Y,
		Z:   g.Z,
		O:   g.O,
		Map: g.Map,
	}
}

// GetObject returns the game object's underlying Object (satisfies Map.AddGameObject objectProvider).
func (g *GameObject) GetObject() *object.Object {
	return g.Object
}

// IsLooted returns whether the chest has been looted.
func (g *GameObject) IsLooted() bool {
	return g.Looted
}

// SetLooted marks the chest as looted.
func (g *GameObject) SetLooted(looted bool) {
	g.Looted = looted
}

// GameObjectManager holds all spawned game objects.
type GameObjectManager struct {
	templates map[uint32]*GameObjectTemplate
	spawns    map[uint32]*GameObject
}

// NewGameObjectManager creates a new game object manager.
// If basedata is available, it loads templates and spawns from there.
func NewGameObjectManager() *GameObjectManager {
	gm := &GameObjectManager{
		templates: make(map[uint32]*GameObjectTemplate),
		spawns:    make(map[uint32]*GameObject),
	}

	// Try to load from basedata (loaded from MongoDB or JSON)
	if bd := basedata.GetInstance(); bd != nil {
		gm.loadFromBasedata(bd)
	} else {
		// Fallback: register minimal test templates
		gm.registerDefaultTemplates()
		gm.spawnDefaults()
	}

	return gm
}

// loadFromBasedata loads templates and spawns from the basedata store.
func (gm *GameObjectManager) loadFromBasedata(bd *basedata.Store) {
	// Load templates
	for entry, tpl := range bd.GameObjectTemplates {
		gm.templates[entry] = &GameObjectTemplate{tpl}
	}

	// Load spawns
	for guid, spawn := range bd.GameObjectSpawns {
		tpl, ok := gm.templates[spawn.Entry]
		if !ok {
			continue
		}

		data := &GameObjectData{spawn}
		g := NewGameObject(tpl, data)
		gm.spawns[guid] = g
	}
}

// registerDefaultTemplates registers basic game object templates.
func (gm *GameObjectManager) registerDefaultTemplates() {
	// Test chest
	gm.templates[100001] = &GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry:     100001,
			Type:      uint8(GameObjectTypeChest),
			DisplayID: 31,
			Name:      "Test Chest",
			Size:      1.0,
			// Data[0]=lockId, Data[1]=lootId
			Data: [24]int32{0, 2589},
		},
	}

	// Test door
	gm.templates[100002] = &GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry:     100002,
			Type:      uint8(GameObjectTypeDoor),
			DisplayID: 57,
			Name:      "Test Door",
			Size:      1.0,
			// Data[0]=startOpen, Data[1]=lockId, Data[2]=autoCloseTime(ms)
			Data: [24]int32{0, 0, 5000},
		},
	}

	// Test sign
	gm.templates[100003] = &GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry:     100003,
			Type:      uint8(GameObjectTypeText),
			DisplayID: 293,
			Name:      "Welcome Sign",
			Size:      1.0,
		},
	}

	// Test goober (button-like clickable object)
	gm.templates[100004] = &GameObjectTemplate{
		GameObjectTemplate: &basedata.GameObjectTemplate{
			Entry:     100004,
			Type:      uint8(GameObjectTypeGoober),
			DisplayID: 293,
			Name:      "Test Goober",
			Size:      1.0,
			// Data[0]=lockId, Data[1]=questId, Data[2]=eventId, Data[3]=autoCloseTime
			Data: [24]int32{0, 0, 0, 3000},
		},
	}
}

// spawnDefaults spawns some test game objects.
func (gm *GameObjectManager) spawnDefaults() {
	// Spawn a test chest near starting area
	gm.spawnFromData(100001, &basedata.GameObjectSpawn{
		GUID:  1,
		Entry: 100001,
		MapID: 0, PhaseMask: 1,
		PosX: -8940.0, PosY: -140.0, PosZ: 83.0,
		State: uint8(GOStateReady),
	})

	// Spawn a test door
	gm.spawnFromData(100002, &basedata.GameObjectSpawn{
		GUID:  2,
		Entry: 100002,
		MapID: 0, PhaseMask: 1,
		PosX: -8935.0, PosY: -135.0, PosZ: 83.0, O: 1.57,
		State: uint8(GOStateReady),
	})

	// Spawn a test sign
	gm.spawnFromData(100003, &basedata.GameObjectSpawn{
		GUID:  3,
		Entry: 100003,
		MapID: 0, PhaseMask: 1,
		PosX: -8930.0, PosY: -130.0, PosZ: 83.5,
		State: uint8(GOStateReady),
	})
}

// spawnFromData spawns a game object from a basedata spawn record.
func (gm *GameObjectManager) spawnFromData(entry uint32, spawn *basedata.GameObjectSpawn) {
	tpl, ok := gm.templates[entry]
	if !ok {
		return
	}

	data := &GameObjectData{spawn}
	g := NewGameObject(tpl, data)
	gm.spawns[spawn.GUID] = g
}

// SpawnObject spawns a game object from template and data.
func (gm *GameObjectManager) SpawnObject(entry uint32, data *GameObjectData) {
	template, ok := gm.templates[entry]
	if !ok {
		return
	}

	g := NewGameObject(template, data)
	gm.spawns[data.GUID] = g
}

// GetObject returns a game object by spawn ID.
func (gm *GameObjectManager) GetObject(id uint32) *GameObject {
	return gm.spawns[id]
}

// GetObjectByGUID returns a game object by GUID (extracts spawn ID from GUID).
func (gm *GameObjectManager) GetObjectByGUID(guid wow.GUID) *GameObject {
	return gm.spawns[guid.Counter()]
}

// GetObjectsInMap returns all game objects in the given map.
func (gm *GameObjectManager) GetObjectsInMap(mapID uint32) []*GameObject {
	var result []*GameObject

	for _, gobj := range gm.spawns {
		if gobj.Map == mapID {
			result = append(result, gobj)
		}
	}

	return result
}

// GetObjects returns all spawned game objects.
func (gm *GameObjectManager) GetObjects() []*GameObject {
	result := make([]*GameObject, 0, len(gm.spawns))
	for _, gobj := range gm.spawns {
		result = append(result, gobj)
	}

	return result
}

// GetTemplate returns a game object template by entry.
func (gm *GameObjectManager) GetTemplate(entry uint32) *GameObjectTemplate {
	return gm.templates[entry]
}

// Use processes a player using a game object.
func (g *GameObject) Use(player interface{}) {
	switch g.Type {
	case GameObjectTypeChest:
		g.useChest()
	case GameObjectTypeDoor:
		g.useDoor()
	case GameObjectTypeButton:
		g.useButton()
	}
}

// useChest handles opening a chest.
func (g *GameObject) useChest() {
	if g.Looted {
		return
	}

	g.Looted = true

	// Mark as opened
	g.Object.SetByteValue(object.GameobjectBytes_1, 0, byte(GOStateActive))
}

// useDoor handles opening/closing a door.
func (g *GameObject) useDoor() {
	if g.GOState == GOStateReady {
		g.GOState = GOStateActive
	} else {
		g.GOState = GOStateReady
	}

	g.Object.SetByteValue(object.GameobjectBytes_1, 0, byte(g.GOState))
}

// useButton handles pressing a button.
func (g *GameObject) useButton() {
	if g.GOState == GOStateReady {
		g.GOState = GOStateActive
	} else {
		g.GOState = GOStateReady
	}

	g.Object.SetByteValue(object.GameobjectBytes_1, 0, byte(g.GOState))
}
