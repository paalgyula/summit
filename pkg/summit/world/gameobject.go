package world

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
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

// GameObjectTemplate holds the template data for a game object type.
// Similar to AzerothCore's GameObjectTemplate.
type GameObjectTemplate struct {
	Entry     uint32
	Type      GameObjectType
	DisplayID uint32
	Name      string
	Faction   uint32
	Flags     uint32
	Size      float32

	// LockID for interaction (0 = no lock)
	LockID uint32

	// QuestID that this GO is related to (0 = none)
	QuestID uint32

	// Chest loot (item entries)
	ChestLoot []uint32

	// Door/GOober settings
	DoorOpenDelay uint32 // ms to auto-close
}

// GameObjectData holds spawn data for a specific game object instance.
// Similar to AzerothCore's GameObjectData.
type GameObjectData struct {
	ID        uint32 // spawn ID
	Entry     uint32 // template entry
	MapID     uint32
	PhaseMask uint32
	PosX      float32
	PosY      float32
	PosZ      float32
	O         float32
	SpawnMask uint32
	AnimProgress uint32
	GOState   GOState
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

	GOState     GOState
	AnimProgress uint32

	Template *GameObjectTemplate

	// Chest state
	Looted    bool
	RespawnTime int64 // when to respawn (Unix ms), 0 = never
}

// NewGameObject creates a new game object from template and spawn data.
func NewGameObject(template *GameObjectTemplate, data *GameObjectData) *GameObject {
	g := &GameObject{
		Object:      object.NewObject(),
		ID:          data.ID,
		Entry:       data.Entry,
		Name:        template.Name,
		Type:        template.Type,
		DisplayID:   template.DisplayID,
		Faction:     template.Faction,
		Flags:       template.Flags,
		Size:        template.Size,
		X:           data.PosX,
		Y:           data.PosY,
		Z:           data.PosZ,
		O:           data.O,
		Map:         data.MapID,
		GOState:     data.GOState,
		AnimProgress: data.AnimProgress,
		Template:    template,
	}

	g.init()

	return g
}

// init sets up the game object's update fields.
func (g *GameObject) init() {
	g.Object.InitValues(int(object.GameobjectEnd))

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
func NewGameObjectManager() *GameObjectManager {
	gm := &GameObjectManager{
		templates: make(map[uint32]*GameObjectTemplate),
		spawns:    make(map[uint32]*GameObject),
	}

	// Register some basic templates
	gm.registerDefaultTemplates()

	// Spawn test game objects
	gm.spawnDefaults()

	return gm
}

// registerDefaultTemplates registers basic game object templates.
func (gm *GameObjectManager) registerDefaultTemplates() {
	// Test chest
	gm.templates[100001] = &GameObjectTemplate{
		Entry:     100001,
		Type:      GameObjectTypeChest,
		DisplayID: 31, // generic chest model
		Name:      "Test Chest",
		Faction:   0,
		Flags:     0,
		Size:      1.0,
		LockID:    0, // no lock
		ChestLoot: []uint32{2589}, // Linen Cloth
	}

	// Test door
	gm.templates[100002] = &GameObjectTemplate{
		Entry:       100002,
		Type:        GameObjectTypeDoor,
		DisplayID:   57, // generic door model
		Name:        "Test Door",
		Faction:     0,
		Flags:       0,
		Size:        1.0,
		DoorOpenDelay: 5000, // 5 seconds
	}

	// Test sign
	gm.templates[100003] = &GameObjectTemplate{
		Entry:     100003,
		Type:      GameObjectTypeText,
		DisplayID: 293, // generic sign model
		Name:      "Welcome Sign",
		Faction:   0,
		Flags:     0,
		Size:      1.0,
	}
}

// spawnDefaults spawns some test game objects.
func (gm *GameObjectManager) spawnDefaults() {
	// Spawn a test chest near starting area
	gm.SpawnObject(100001, &GameObjectData{
		ID:        1,
		Entry:     100001,
		MapID:     0,
		PhaseMask: 1,
		PosX:      -8940.0,
		PosY:      -140.0,
		PosZ:      83.0,
		O:         0,
		SpawnMask: 1,
		GOState:   GOStateReady,
	})

	// Spawn a test door
	gm.SpawnObject(100002, &GameObjectData{
		ID:        2,
		Entry:     100002,
		MapID:     0,
		PhaseMask: 1,
		PosX:      -8935.0,
		PosY:      -135.0,
		PosZ:      83.0,
		O:         1.57, // facing south
		SpawnMask: 1,
		GOState:   GOStateReady,
	})

	// Spawn a test sign
	gm.SpawnObject(100003, &GameObjectData{
		ID:        3,
		Entry:     100003,
		MapID:     0,
		PhaseMask: 1,
		PosX:      -8930.0,
		PosY:      -130.0,
		PosZ:      83.5,
		O:         0,
		SpawnMask: 1,
		GOState:   GOStateReady,
	})
}

// SpawnObject spawns a game object from template and data.
func (gm *GameObjectManager) SpawnObject(entry uint32, data *GameObjectData) {
	template, ok := gm.templates[entry]
	if !ok {
		return
	}

	g := NewGameObject(template, data)
	gm.spawns[data.ID] = g
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
