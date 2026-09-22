package world

import (
	"math"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/loot"
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

// GOState mirrors the C++ GOState enum (GameObjectData.h).
type GOState uint8

const (
	// GOStateActive is shown as used/open and not reset (e.g. open door).
	GOStateActive GOState = 0
	// GOStateReady is shown as ready (e.g. closed door).
	GOStateReady GOState = 1
	// GOStateActiveAlternative is used in an alternate way (e.g. opened by cannon).
	GOStateActiveAlternative GOState = 2
)

// GameObjectTemplate wraps the basedata template with convenience accessors.
type GameObjectTemplate struct {
	*basedata.GameObjectTemplate
}

// Type returns the game object type as the local enum.
func (t *GameObjectTemplate) Type() GameObjectType {
	return GameObjectType(t.GameObjectTemplate.Type)
}

// GetFlags returns the GO flags loaded from the template/addon.
func (t *GameObjectTemplate) GetFlags() uint32 { return t.GameObjectTemplate.Flags }

// GetFaction returns the GO faction loaded from the template/addon.
func (t *GameObjectTemplate) GetFaction() uint32 { return t.GameObjectTemplate.Faction }

// GameObjectData wraps a basedata.GameObjectSpawn for use by the world server.
type GameObjectData struct {
	*basedata.GameObjectSpawn
}

// GameObject represents a spawned game object in the world.
type GameObject struct {
	*object.Object

	ID        uint32 // spawn ID
	Entry     uint32
	Name      string
	Type      GameObjectType
	DisplayID uint32
	Faction   uint32
	Flags     uint32
	Size      float32

	X, Y, Z, O float32
	Map        uint32

	GOState      GOState
	AnimProgress uint32

	// WorldRotation is the GO's world-space unit quaternion (x, y, z, w).
	// Mirrors AzerothCore GameObject::WorldRotation.
	WorldRotation [4]float32

	// ParentRotation is GAMEOBJECT_PARENTROTATION (transport / addon quat);
	// identity by default.
	ParentRotation [4]float32

	// PackedRotation is the client-packed form of WorldRotation, sent inline
	// when UPDATEFLAG_ROTATION is set.
	PackedRotation int64

	Template *GameObjectTemplate

	// Runtime state (mirrors AzerothCore GameObject's protected members).
	LootState        LootState
	ArtKit           uint8
	SpawnedByDefault bool
	RespawnDelay     time.Duration
	RespawnAt        time.Time
	DespawnAt        time.Time
	CooldownAt       time.Time
	RestockAt        time.Time
	UseCount         uint32
	LinkedTrap       wow.GUID
	OwnerGUID        wow.GUID
	SpellID          uint32

	// Loot holds the generated loot for chests/fishing holes; nil until opened.
	Loot *loot.Loot
	// LootRecipient is the player currently allowed to loot, if any.
	LootRecipient wow.GUID
}

// NewGameObject creates a new game object from template and spawn data.
func NewGameObject(template *GameObjectTemplate, data *GameObjectData) *GameObject {
	g := &GameObject{
		Object:           object.NewObject(),
		ID:               data.GUID,
		Entry:            data.Entry,
		Name:             template.Name,
		Type:             template.Type(),
		DisplayID:        template.DisplayID,
		Faction:          template.GetFaction(),
		Flags:            template.GetFlags(),
		Size:             template.Size,
		X:                data.PosX,
		Y:                data.PosY,
		Z:                data.PosZ,
		O:                data.O,
		Map:              data.MapID,
		GOState:          GOState(data.State),
		AnimProgress:     uint32(data.AnimProgress),
		ArtKit:           uint8(template.ArtKit),
		Template:         template,
		LootState:        GO_READY,
		SpawnedByDefault: true,
	}

	// Respawn delay follows the spawn's spawntimesecs when set.
	if data.SpawnTimeSecs > 0 {
		g.RespawnDelay = time.Duration(data.SpawnTimeSecs) * time.Second
	}

	// Rotation: prefer the spawn's world quaternion, otherwise derive a
	// Z-axis quaternion from the spawn orientation (mirrors
	// GameObject::SetWorldRotation's zero-quaternion fallback).
	g.ParentRotation = [4]float32{0, 0, 0, 1}
	g.WorldRotation = QuaternionFromSpawn(data.Rotation, data.O)
	g.PackedRotation = PackQuaternion(g.WorldRotation)

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

	// GAMEOBJECT_PARENTROTATION: quaternion (x, y, z, w)
	for i := 0; i < 4; i++ {
		g.Object.SetFloatValue(object.GameobjectParentrotation+object.UpdateField(i), g.ParentRotation[i])
	}

	// GAMEOBJECT_DYNAMIC: two uint16 (dynamic flags low | path progress high).
	g.Object.SetUInt32Value(object.GameobjectDynamic, 0)

	// GAMEOBJECT_BYTES_1: state(0) | type(1) | artkit(2) | animprogress(3)
	g.Object.SetByteValue(object.GameobjectBytes_1, 0, byte(g.GOState))
	g.Object.SetByteValue(object.GameobjectBytes_1, 1, byte(g.Type))
	g.Object.SetByteValue(object.GameobjectBytes_1, 2, g.ArtKit)
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

// GetByEntry returns the first spawned game object with the given template
// entry, or nil. Used to resolve linked traps and other entry-based lookups.
func (gm *GameObjectManager) GetByEntry(entry uint32) *GameObject {
	for _, gobj := range gm.spawns {
		if gobj.Entry == entry {
			return gobj
		}
	}

	return nil
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

// GetPackedRotation returns the client-packed world rotation, sent inline when
// UPDATEFLAG_ROTATION is set (AzerothCore Object::BuildMovementUpdate).
func (g *GameObject) GetPackedRotation() int64 {
	return g.PackedRotation
}

// InteractionDistance returns the per-type interaction range in yards,
// mirroring AzerothCore's GameObject::GetInteractionDistance.
func (g *GameObject) InteractionDistance() float32 {
	switch g.Type {
	case GameObjectTypeQuestGiver, GameObjectTypeText,
		GameObjectTypeFlagStand, GameObjectTypeFlagDrop, GameObjectTypeMiniGame:
		return 5.5555553
	case GameObjectTypeBinder:
		return 10.0
	case GameObjectTypeChair, GameObjectTypeBarberChair:
		return 3.0
	case GameObjectTypeFishingNode:
		return 100.0
	case GameObjectTypeFishingHole:
		return 20.0 + 0.5 // 20 + CONTACT_DISTANCE
	case GameObjectTypeCamera, GameObjectTypeMapObject,
		GameObjectTypeDungeonDifficulty, GameObjectTypeDestructibleBuilding,
		GameObjectTypeDoor:
		return 5.0
	case GameObjectTypeGuildBank, GameObjectTypeMailbox:
		return 10.0
	default:
		return 5.5 // INTERACTION_DISTANCE
	}
}

// IsWithinInteractionDistance reports whether the given world point is within
// the game object's interaction range.
func (g *GameObject) IsWithinInteractionDistance(x, y, z float32) bool {
	dx := g.X - x
	dy := g.Y - y
	dz := g.Z - z

	d := g.InteractionDistance()

	return dx*dx+dy*dy+dz*dz <= d*d
}

// CreateUpdateType returns CREATE_OBJECT2 for the game object types AzerothCore
// flags as such; every other GO uses a plain CREATE_OBJECT.
func (g *GameObject) CreateUpdateType() wow.ObjectUpdateType {
	switch g.Type {
	case GameObjectTypeTrap, GameObjectTypeDuelArbiter,
		GameObjectTypeFlagStand, GameObjectTypeFlagDrop:
		return wow.UpdateTypeCreateObject2
	default:
		return wow.UpdateTypeCreateObject
	}
}

// QuaternionFromSpawn prefers the spawn's world quaternion; when it is zero it
// derives a Z-axis quaternion from the spawn orientation, matching
// GameObject::SetWorldRotation's zero-rotation fallback.
func QuaternionFromSpawn(rot [4]float32, orientation float32) [4]float32 {
	if rot[0] != 0 || rot[1] != 0 || rot[2] != 0 || rot[3] != 0 {
		return normalizeQuaternion(rot)
	}

	half := float64(orientation) * 0.5

	return [4]float32{0, 0, float32(math.Sin(half)), float32(math.Cos(half))}
}

// normalizeQuaternion returns a unit quaternion (falls back to identity).
func normalizeQuaternion(q [4]float32) [4]float32 {
	length := math.Sqrt(float64(q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3]))
	if length == 0 {
		return [4]float32{0, 0, 0, 1}
	}

	inv := float32(1.0 / length)

	return [4]float32{q[0] * inv, q[1] * inv, q[2] * inv, q[3] * inv}
}

// PackQuaternion packs a unit quaternion into the client's 21/22-bit-per-axis
// packed rotation format (AzerothCore GameObject::UpdatePackedRotation).
func PackQuaternion(q [4]float32) int64 {
	const (
		packYZ     = int32(1 << 20)
		packX      = packYZ << 1
		packYZMask = int64((packYZ << 1) - 1)
		packXMask  = int64((packX << 1) - 1)
	)

	wSign := int32(1)
	if q[3] < 0 {
		wSign = -1
	}

	x := int64(int32(q[0]*float32(packX))*wSign) & packXMask
	y := int64(int32(q[1]*float32(packYZ))*wSign) & packYZMask
	z := int64(int32(q[2]*float32(packYZ))*wSign) & packYZMask

	return z | (y << 21) | (x << 42)
}
