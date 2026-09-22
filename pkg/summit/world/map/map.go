package mapmanager

import (
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/areatrigger"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
)

// Map represents a game map (world, dungeon, battleground, etc.).
type Map struct {
	ID         uint32
	InstanceID uint32
	SpawnMode  uint8

	Name  string
	Entry *MapEntry

	players map[uint32]*player.Player
	npcs    map[uint32]interface{} // NPC interface
	objects map[uint32]interface{} // GameObject interface

	// updateObjects holds objects that have dirty fields and need
	// value updates sent to visible players on the next tick.
	updateObjects map[*object.Object]struct{}

	// visibilityTracker tracks which objects each player can see.
	visibilityTracker *VisibilityTracker

	// visibilityRange is the default visibility range for this map.
	visibilityRange float32

	// grids is the spatial grid for this map (key: gy*64+gx).
	grids map[uint32]*MapGrid

	// AreaTriggers on this map
	areaTriggers []*areatrigger.AreaTrigger

	mutex sync.RWMutex
	log   zerolog.Logger
}

// MapEntry holds basic map data from DBC.
type MapEntry struct {
	ID           uint32
	InstanceType uint32
	Flags        uint32
	Name         string
	LinkedZone   uint32
	MultimapID   uint32
	EntranceMap  uint32
	EntranceX    float32
	EntranceY    float32
	ExpansionID  uint32
	MaxPlayers   uint32
}

// NewMap creates a new map instance.
func NewMap(id, instanceID uint32, entry *MapEntry) *Map {
	return &Map{
		ID:                id,
		InstanceID:        instanceID,
		Entry:             entry,
		players:           make(map[uint32]*player.Player),
		npcs:              make(map[uint32]interface{}),
		objects:           make(map[uint32]interface{}),
		updateObjects:     make(map[*object.Object]struct{}),
		visibilityTracker: NewVisibilityTracker(),
		visibilityRange:   DefaultVisibilityDistance,
		log:               zerolog.Logger{},
	}
}

// AddPlayer adds a player to the map.
func (m *Map) AddPlayer(p *player.Player) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.players[p.ID] = p
	p.IsInWorld = true
	p.Object.SetUpdater(m)
}

// RemovePlayer removes a player from the map.
func (m *Map) RemovePlayer(guid uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if p, ok := m.players[guid]; ok {
		p.IsInWorld = false
		p.Object.SetUpdater(nil)
		p.Object.RemoveFromObjectUpdate()
		m.visibilityTracker.ClearPlayer(p.GUID())
		delete(m.players, guid)
	}
}

// GetPlayer returns a player by GUID.
func (m *Map) GetPlayer(guid uint32) *player.Player {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.players[guid]
}

// GetPlayers returns all players on the map.
func (m *Map) GetPlayers() []*player.Player {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]*player.Player, 0, len(m.players))
	for _, p := range m.players {
		result = append(result, p)
	}

	return result
}

// GetPlayersCount returns the number of players on the map.
func (m *Map) GetPlayersCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.players)
}

// AddNPC adds an NPC to the map.
func (m *Map) AddNPC(npc interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	type guidGetter interface {
		GetGUID() wow.GUID
	}
	type positionProvider interface {
		GetPosition() *player.WorldLocation
	}
	type objectProvider interface {
		GetNPCObject() *object.Object
	}

	g, ok := npc.(guidGetter)
	if !ok {
		return
	}

	m.npcs[uint32(g.GetGUID())] = npc

	// Also add to grid for spatial queries
	if pos, ok := npc.(positionProvider); ok {
		if obj, ok := npc.(objectProvider); ok {
			loc := pos.GetPosition()
			if loc != nil && obj.GetNPCObject() != nil {
				m.AddObjectToGrid(obj.GetNPCObject(), loc.X, loc.Y, loc.Z)
			}
		}
	}

	// Field changes (health, flags, ...) are queued on the map and sent as
	// value updates to the players that can see the NPC, like for players.
	if obj, ok := npc.(objectProvider); ok && obj.GetNPCObject() != nil {
		m.attachUpdater(obj.GetNPCObject())
	}
}

// attachUpdater makes the map the object's update sink. The initial field
// setup already dirtied every value, so the mask is cleared first; the create
// block carries the full state anyway.
func (m *Map) attachUpdater(obj *object.Object) {
	obj.ClearChanges()
	obj.SetUpdater(m)
}

// RemoveNPC takes an NPC out of the map (corpse decay, despawn): it leaves
// the grid, and every player that had it in view gets a destroy packet.
func (m *Map) RemoveNPC(guid wow.GUID) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	npc, ok := m.npcs[uint32(guid)]
	if !ok {
		return
	}

	delete(m.npcs, uint32(guid))

	obj := m.getNPCObject(npc)
	if obj == nil {
		return
	}

	if pos := m.getNPCPosition(npc); pos != nil {
		m.RemoveObjectFromGrid(obj, pos.X, pos.Y)
	}

	// Drop any pending value update (RemoveFromObjectUpdate would re-lock)
	delete(m.updateObjects, obj)
	obj.MarkUpdateSent()

	for _, p := range m.players {
		if m.visibilityTracker.IsVisible(p.GUID(), obj.GUID()) {
			m.visibilityTracker.ClearVisible(p.GUID(), obj.GUID())
			m.sendNPCDestroyToPlayer(obj, p)
		}
	}
}

// AddGameObject adds a game object to the map.
func (m *Map) AddGameObject(gobj interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	type guidGetter interface {
		GetGUID() wow.GUID
	}
	type positionProvider interface {
		GetPosition() *player.WorldLocation
	}
	type objectProvider interface {
		GetObject() *object.Object
	}

	g, ok := gobj.(guidGetter)
	if !ok {
		return
	}

	m.objects[uint32(g.GetGUID())] = gobj

	// Also add to grid for spatial queries
	if pos, ok := gobj.(positionProvider); ok {
		if obj, ok := gobj.(objectProvider); ok {
			loc := pos.GetPosition()
			if loc != nil && obj.GetObject() != nil {
				m.AddObjectToGrid(obj.GetObject(), loc.X, loc.Y, loc.Z)
			}
		}
	}

	if obj, ok := gobj.(objectProvider); ok && obj.GetObject() != nil {
		m.attachUpdater(obj.GetObject())
	}
}

// HavePlayers returns true if there are players on the map.
func (m *Map) HavePlayers() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.players) > 0
}

// UpdatePlayerVisibility checks all objects within range of a player and
// sends create/destroy packets as needed. This mirrors AzerothCore's
// Player::UpdateVisibilityForPlayer. Acquires the map lock.
func (m *Map) UpdatePlayerVisibility(p *player.Player) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.updatePlayerVisibilityLocked(p)
}

// UpdateObjectVisibility notifies all players within range about changes to an object.
// This mirrors AzerothCore's WorldObject::UpdateObjectVisibility.
// Call this when an object's visibility state changes (stealth, faction, etc.)
func (m *Map) UpdateObjectVisibility(obj *object.Object) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if obj == nil {
		return
	}

	// Get object position - for now, only handle players
	// TODO: Support NPC/GO position lookup
	var objX, objY float32

	// Check if this is a player object
	for _, p := range m.players {
		if p.Object == obj {
			objX = p.Location.X
			objY = p.Location.Y
			break
		}
	}

	// Notify all players within range about the change
	for _, player := range m.players {
		if !player.IsInWorld || player.Sender == nil {
			continue
		}

		dist := Distance2DPositions(
			objX, objY,
			player.Location.X, player.Location.Y,
		)

		sightRange := GetEffectiveSightRange(player, m)
		if dist <= sightRange {
			// Player is in range - they should see the update
			// The value update system will handle sending the changed fields
			// Force the object to be queued for update
			obj.AddToObjectUpdateIfNeeded()
		}
	}
}

// sendCreateToPlayer sends a create object packet for a player to another player.
func (m *Map) sendCreateToPlayer(source, target *player.Player) {
	if target.Sender == nil {
		return
	}

	// Build create object packet
	buf := object.NewUpdateBlockBuffer()

	// Update type: CreateObject
	_ = buf.WriteOne(int(wow.UpdateTypeCreateObject))

	// GUID
	_ = buf.Write(source.GUID())

	// Object type ID
	_ = buf.WriteOne(int(wow.TypeIDPlayer))

	// Update flags
	flags := source.Object.UpdateFlags() | wow.UpdateFlagStationaryPosition
	_ = buf.Write(flags)

	mv := &object.MovementBlock{
		Flags:   source.MoveFlags &^ wow.MovementFlagOnTransport,
		Time:    uint32(time.Now().UnixMilli()),
		X:       source.Location.X,
		Y:       source.Location.Y,
		Z:       source.Location.Z,
		O:       source.Location.O,
		LowGUID: 0x08,
	}
	if source.Unit != nil {
		mv.Speeds = source.Unit.Speed
	} else {
		mv.Speeds = object.DefaultUnitSpeeds()
	}
	object.WriteMovementBlock(buf, flags, mv)

	// Values update - full mask for create
	mask := source.Object.BuildFilteredUpdateMask(target.Object, false)
	block := source.Object.BuildValuesUpdateBlock(mask, target.Object)
	buf.WriteBytes(block)

	// Build packet
	ud := object.NewUpdateData()
	ud.AddUpdateBlock(buf.Bytes())
	pkt := ud.BuildPacket()

	target.Sender.Send(pkt)
}

// sendDestroyToPlayer sends a destroy object packet for a player to another player.
func (m *Map) sendDestroyToPlayer(source *player.Player, target *player.Player) {
	if target.Sender == nil {
		return
	}

	// Build destroy packet
	pkt := wow.NewPacket(wow.ServerDestroyObject)
	_ = pkt.Write(source.GUID())
	_ = pkt.WriteOne(0) // not despawn animation
	target.Sender.Send(pkt)
}

// sendNPCCreateToPlayer sends a create object packet for an NPC to a player.
func (m *Map) sendNPCCreateToPlayer(npc interface{}, target *player.Player) {
	if target.Sender == nil {
		return
	}

	// Type assert to get NPC methods
	type npcProvider interface {
		GetGUID() wow.GUID
		GetNPCObject() *object.Object
	}

	npcObj, ok := npc.(npcProvider)
	if !ok {
		return
	}

	// Build create object packet
	buf := object.NewUpdateBlockBuffer()

	// Update type: CreateObject
	_ = buf.WriteOne(int(wow.UpdateTypeCreateObject))

	// GUID
	_ = buf.Write(npcObj.GetGUID())

	// Object type ID
	_ = buf.WriteOne(int(wow.TypeIDUnit))

	// Update flags
	flags := wow.UpdateFlagLowGUID | wow.UpdateFlagLiving | wow.UpdateFlagStationaryPosition
	_ = buf.Write(flags)

	mv := &object.MovementBlock{Speeds: object.DefaultUnitSpeeds(), LowGUID: 0x0B}
	if pos := m.getNPCPosition(npc); pos != nil {
		mv.X, mv.Y, mv.Z, mv.O = pos.X, pos.Y, pos.Z, pos.O
	}
	object.WriteMovementBlock(buf, flags, mv)

	// Values update - full mask for create
	obj := npcObj.GetNPCObject()
	if obj != nil {
		mask := obj.BuildFilteredUpdateMask(target.Object, false)
		block := obj.BuildValuesUpdateBlock(mask, target.Object)
		buf.WriteBytes(block)
	}

	// Build packet
	ud := object.NewUpdateData()
	ud.AddUpdateBlock(buf.Bytes())
	pkt := ud.BuildPacket()

	target.Sender.Send(pkt)
}

// sendNPCDestroyToPlayer sends a destroy object packet for an NPC to a player.
func (m *Map) sendNPCDestroyToPlayer(obj *object.Object, target *player.Player) {
	if target.Sender == nil || obj == nil {
		return
	}

	// Build destroy packet - use a minimal player-like struct for the GUID
	pkt := wow.NewPacket(wow.ServerDestroyObject)
	_ = pkt.Write(obj.GUID())
	_ = pkt.WriteOne(0) // not despawn animation
	target.Sender.Send(pkt)
}

// getNPCObject extracts the Object from an NPC interface.
func (m *Map) getNPCObject(npc interface{}) *object.Object {
	type npcObjectProvider interface {
		GetNPCObject() *object.Object
	}

	if provider, ok := npc.(npcObjectProvider); ok {
		return provider.GetNPCObject()
	}
	return nil
}

// getNPCPosition extracts the position from an NPC interface.
func (m *Map) getNPCPosition(npc interface{}) *player.WorldLocation {
	type positionProvider interface {
		GetPosition() *player.WorldLocation
	}

	if provider, ok := npc.(positionProvider); ok {
		return provider.GetPosition()
	}
	return nil
}

// getGameObjectObject extracts the Object from a game object interface.
func (m *Map) getGameObjectObject(gobj interface{}) *object.Object {
	type objectProvider interface {
		GetObject() *object.Object
	}

	if provider, ok := gobj.(objectProvider); ok {
		return provider.GetObject()
	}
	return nil
}

// getGameObjectPosition extracts the position from a game object interface.
func (m *Map) getGameObjectPosition(gobj interface{}) *player.WorldLocation {
	type positionProvider interface {
		GetPosition() *player.WorldLocation
	}

	if provider, ok := gobj.(positionProvider); ok {
		return provider.GetPosition()
	}
	return nil
}

// sendGameObjectCreateToPlayer sends a create object packet for a game object to a player.
func (m *Map) sendGameObjectCreateToPlayer(gobj interface{}, target *player.Player) {
	if target.Sender == nil {
		return
	}

	type guidProvider interface {
		GetGUID() wow.GUID
	}
	type objectProvider interface {
		GetObject() *object.Object
	}
	type positionProvider interface {
		GetPosition() *player.WorldLocation
	}

	guid, ok := gobj.(guidProvider)
	if !ok {
		return
	}
	obj, ok := gobj.(objectProvider)
	if !ok {
		return
	}
	pos, ok := gobj.(positionProvider)
	if !ok {
		return
	}

	loc := pos.GetPosition()
	if loc == nil {
		return
	}

	buf := object.NewUpdateBlockBuffer()

	// Update type: CreateObject
	_ = buf.WriteOne(int(wow.UpdateTypeCreateObject))

	// GUID
	_ = buf.Write(guid.GetGUID())

	// Object type ID
	_ = buf.WriteOne(int(wow.TypeIDGameObject))

	// Update flags
	flags := wow.UpdateFlagLowGUID | wow.UpdateFlagStationaryPosition | wow.UpdateFlagRotation
	_ = buf.Write(flags)

	object.WriteMovementBlock(buf, flags, &object.MovementBlock{
		X: loc.X, Y: loc.Y, Z: loc.Z, O: loc.O,
		LowGUID: uint32(guid.GetGUID().Counter()),
	})

	// Values update - full mask for create
	goObj := obj.GetObject()
	if goObj != nil {
		mask := goObj.BuildFilteredUpdateMask(target.Object, false)
		block := goObj.BuildValuesUpdateBlock(mask, target.Object)
		buf.WriteBytes(block)
	}

	// Build packet
	ud := object.NewUpdateData()
	ud.AddUpdateBlock(buf.Bytes())
	pkt := ud.BuildPacket()

	target.Sender.Send(pkt)
}

// sendGameObjectDestroyToPlayer sends a destroy object packet for a game object to a player.
func (m *Map) sendGameObjectDestroyToPlayer(obj *object.Object, target *player.Player) {
	if target.Sender == nil || obj == nil {
		return
	}

	pkt := wow.NewPacket(wow.ServerDestroyObject)
	_ = pkt.Write(obj.GUID())
	_ = pkt.WriteOne(0) // not despawn animation
	target.Sender.Send(pkt)
}

// AddUpdateObject implements object.ObjectUpdater. It queues an object
// for value updates on the next tick.
func (m *Map) AddUpdateObject(obj *object.Object) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.updateObjects[obj] = struct{}{}
}

// RemoveUpdateObject implements object.ObjectUpdater. It removes an
// object from the update queue.
func (m *Map) RemoveUpdateObject(obj *object.Object) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.updateObjects, obj)
}

// SendObjectUpdates drains the update queue and sends value updates to
// all visible players for each dirty object. Mirrors AzerothCore's
// Map::SendObjectUpdates. Acquires the map lock.
func (m *Map) SendObjectUpdates() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.SendObjectUpdatesLocked()
}

// SendObjectUpdatesLocked is the internal version that assumes the lock is held.
func (m *Map) SendObjectUpdatesLocked() {
	if len(m.updateObjects) == 0 {
		return
	}

	// Collect players snapshot for building updates outside the lock
	players := make([]*player.Player, 0, len(m.players))
	for _, p := range m.players {
		if p.IsInWorld {
			players = append(players, p)
		}
	}

	// Take the dirty set: ClearUpdateMask re-enters the map lock through
	// RemoveUpdateObject, so it must not run while iterating m.updateObjects.
	dirty := make([]*object.Object, 0, len(m.updateObjects))
	for obj := range m.updateObjects {
		dirty = append(dirty, obj)
	}

	m.updateObjects = make(map[*object.Object]struct{})

	for _, obj := range dirty {
		m.sendUpdateForObject(obj, players)
		obj.ClearChanges()
		obj.MarkUpdateSent()
	}
}

// sendUpdateForObject builds value update packets for a single object
// and sends them to all visible players.
func (m *Map) sendUpdateForObject(obj *object.Object, players []*player.Player) {
	for _, target := range players {
		if target.Sender == nil {
			continue
		}

		// Only players that have the object created client-side get updates
		// for it; the player itself is never in its own visibility set.
		if target.GUID() != obj.GUID() && !m.visibilityTracker.IsVisible(target.GUID(), obj.GUID()) {
			continue
		}

		// Build incremental values update (only changed + visible fields)
		mask := obj.BuildIncrementalUpdateMask(target.Object)
		if mask.GetUpdateBlockCount() == 0 {
			continue
		}

		block := obj.BuildValuesUpdateBlock(mask, target.Object)

		// Wrap in SMSG_UPDATE_OBJECT with UPDATETYPE_VALUES
		ud := object.NewUpdateData()
		buf := object.NewUpdateBlockBuffer()

		// Update type: Values
		_ = buf.WriteOne(0) // UPDATETYPE_VALUES = 0

		// GUID
		_ = buf.Write(obj.GUID())

		// Values block
		buf.WriteBytes(block)

		ud.AddUpdateBlock(buf.Bytes())

		pkt := ud.BuildPacket()
		target.Sender.Send(pkt)
	}
}

// Update processes map updates.
func (m *Map) Update(diff uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Update visibility for all players (check range, send create/destroy)
	for _, p := range m.players {
		if p.IsInWorld {
			// Update visibility for this player (check range, send create/destroy)
			// This is done inside the lock to avoid race conditions
			m.updatePlayerVisibilityLocked(p)
		}
	}

	// Send queued value updates to visible players
	m.SendObjectUpdatesLocked()
}

// updatePlayerVisibilityLocked checks all objects within range of a player and
// sends create/destroy packets as needed. Must be called with m.mutex held.
func (m *Map) updatePlayerVisibilityLocked(p *player.Player) {
	if p == nil || !p.IsInWorld || p.Sender == nil {
		return
	}

	sightRange := GetEffectiveSightRange(p, m)
	playerGUID := p.GUID()

	// Check all other players
	for _, other := range m.players {
		if other.ID == p.ID || !other.IsInWorld {
			continue
		}

		// Calculate distance between players
		dist := Distance2DPositions(
			p.Location.X, p.Location.Y,
			other.Location.X, other.Location.Y,
		)

		isVisible := m.visibilityTracker.IsVisible(playerGUID, other.GUID())
		inRange := dist <= sightRange

		if inRange && !isVisible {
			// Player just came into range - send create
			m.visibilityTracker.SetVisible(playerGUID, other.GUID())
			m.sendCreateToPlayer(other, p)
		} else if !inRange && isVisible {
			// Player went out of range - send destroy
			m.visibilityTracker.ClearVisible(playerGUID, other.GUID())
			m.sendDestroyToPlayer(other, p)
		}
	}

	// Check all NPCs
	for _, npc := range m.npcs {
		npcObj := m.getNPCObject(npc)
		if npcObj == nil {
			continue
		}

		npcPos := m.getNPCPosition(npc)
		if npcPos == nil {
			continue
		}

		dist := Distance2DPositions(
			p.Location.X, p.Location.Y,
			npcPos.X, npcPos.Y,
		)

		// Use the NPC's visibility range if it has an override,
		// otherwise use the player's sight range.
		objVisibilityRange := GetVisibilityRange(npcObj, m)
		effectiveRange := sightRange
		if objVisibilityRange > effectiveRange {
			effectiveRange = objVisibilityRange
		}

		isVisible := m.visibilityTracker.IsVisible(playerGUID, npcObj.GUID())
		inRange := dist <= effectiveRange

		if inRange && !isVisible {
			// NPC just came into range - send create
			m.visibilityTracker.SetVisible(playerGUID, npcObj.GUID())
			m.sendNPCCreateToPlayer(npc, p)
		} else if !inRange && isVisible {
			// NPC went out of range - send destroy
			m.visibilityTracker.ClearVisible(playerGUID, npcObj.GUID())
			m.sendNPCDestroyToPlayer(npcObj, p)
		}
	}

	// Check all game objects
	for _, gobj := range m.objects {
		gobjObj := m.getGameObjectObject(gobj)
		if gobjObj == nil {
			continue
		}

		gobjPos := m.getGameObjectPosition(gobj)
		if gobjPos == nil {
			continue
		}

		dist := Distance2DPositions(
			p.Location.X, p.Location.Y,
			gobjPos.X, gobjPos.Y,
		)

		isVisible := m.visibilityTracker.IsVisible(playerGUID, gobjObj.GUID())
		inRange := dist <= sightRange

		if inRange && !isVisible {
			// Game object just came into range - send create
			m.visibilityTracker.SetVisible(playerGUID, gobjObj.GUID())
			m.sendGameObjectCreateToPlayer(gobj, p)
		} else if !inRange && isVisible {
			// Game object went out of range - send destroy
			m.visibilityTracker.ClearVisible(playerGUID, gobjObj.GUID())
			m.sendGameObjectDestroyToPlayer(gobjObj, p)
		}
	}
}

// AddAreaTrigger adds an areatrigger to this map.
func (m *Map) AddAreaTrigger(at *areatrigger.AreaTrigger) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.areaTriggers = append(m.areaTriggers, at)
}

// GetAreaTriggers returns all areatriggers on this map.
func (m *Map) GetAreaTriggers() []*areatrigger.AreaTrigger {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.areaTriggers
}

// IsDungeon returns true if this is a dungeon map.
func (m *Map) IsDungeon() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 1 // INSTANCE_MULTIMAP
}

// IsRaid returns true if this is a raid map.
func (m *Map) IsRaid() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 4 // INSTANCE_RAID
}

// IsBattleground returns true if this is a battleground map.
func (m *Map) IsBattleground() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 3 // INSTANCE_BATTLEGROUND
}

// IsBattleArena returns true if this is an arena map.
func (m *Map) IsBattleArena() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 2 // INSTANCE_ARENA
}

// IsWorldMap returns true if this is a world map (open world).
func (m *Map) IsWorldMap() bool {
	if m.Entry == nil {
		return false
	}
	return m.Entry.InstanceType == 0 // INSTANCE_NONE
}

// GetVisibilityRange returns the map's default visibility range.
func (m *Map) GetVisibilityRange() float32 {
	return m.visibilityRange
}

// IsInstanceable returns true if this map supports instancing.
func (m *Map) IsInstanceable() bool {
	return m.IsDungeon() || m.IsRaid() || m.IsBattleground() || m.IsBattleArena()
}

// GetMapDifficulty returns the map difficulty for the current spawn mode.
// This is a simplified version - actual implementation would look up MapDifficulty.dbc.
func (m *Map) GetMapDifficulty() uint32 {
	return uint32(m.SpawnMode)
}

// SendToPlayers sends a packet to all players on the map.
func (m *Map) SendToPlayers(data []byte) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, p := range m.players {
		if p.IsInWorld {
			// Would send packet via session - needs integration with WorldSession
			_ = data
			_ = p
		}
	}
}
