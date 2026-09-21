package mapmanager

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Integration tests — full update cycle with grid-based visibility
// ---------------------------------------------------------------------------

// spySender records packets sent to a player.
type spySender struct {
	packets []*wow.Packet
	create  int
	destroy int
	values  int
}

func (s *spySender) Send(pkt *wow.Packet) {
	s.packets = append(s.packets, pkt)
	if pkt.Opcode() == wow.ServerUpdateObject {
		b := pkt.Bytes()
		if len(b) > 4 {
			switch b[4] {
			case wow.UpdateTypeCreateObject, wow.UpdateTypeCreateObject2:
				s.create++
			case wow.UpdateTypeValues:
				s.values++
			}
		}
	} else if pkt.Opcode() == wow.ServerDestroyObject {
		s.destroy++
	}
}

// TestGridIntegration_PlayerInVisibilityRange verifies that a player
// within visibility range receives a create packet for another player.
func TestGridIntegration_PlayerInVisibilityRange(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.InitVisibilityDistance() // 100y for continent

	p1 := player.NewPlayer()
	p1.ID = 1
	p1.Race, p1.Class = wow.RaceOrc, wow.ClassWarior
	p1.Init()
	p1.Location.X = 0
	p1.Location.Y = 0
	s1 := &spySender{}
	p1.Sender = s1
	m.AddPlayer(p1)

	p2 := player.NewPlayer()
	p2.ID = 2
	p2.Race, p2.Class = wow.RaceOrc, wow.ClassWarior
	p2.Init()
	p2.Location.X = 50 // 50 yards away — within 100y range
	p2.Location.Y = 0
	s2 := &spySender{}
	p2.Sender = s2
	m.AddPlayer(p2)

	// Run update cycle
	m.Update(50)

	// Both players should see each other (create packets sent)
	assert.Greater(t, s1.create, 0, "p1 should receive create for p2")
	assert.Greater(t, s2.create, 0, "p2 should receive create for p1")
}

// TestGridIntegration_PlayerOutOfRange verifies that players outside
// visibility range do NOT receive create packets.
func TestGridIntegration_PlayerOutOfRange(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.InitVisibilityDistance() // 100y

	p1 := player.NewPlayer()
	p1.ID = 1
	p1.Race, p1.Class = wow.RaceOrc, wow.ClassWarior
	p1.Init()
	p1.Location.X = 0
	p1.Location.Y = 0
	s1 := &spySender{}
	p1.Sender = s1
	m.AddPlayer(p1)

	p2 := player.NewPlayer()
	p2.ID = 2
	p2.Race, p2.Class = wow.RaceOrc, wow.ClassWarior
	p2.Init()
	p2.Location.X = 200 // 200 yards away — outside 100y range
	p2.Location.Y = 0
	s2 := &spySender{}
	p2.Sender = s2
	m.AddPlayer(p2)

	// Run update cycle
	m.Update(50)

	// Neither should see the other
	assert.Equal(t, 0, s1.create, "p1 should NOT receive create for p2")
	assert.Equal(t, 0, s2.create, "p2 should NOT receive create for p1")
}

// TestGridIntegration_PlayerMovesIntoRange verifies that when a player
// moves into range, they receive a create packet on the next tick.
func TestGridIntegration_PlayerMovesIntoRange(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.InitVisibilityDistance() // 100y

	p1 := player.NewPlayer()
	p1.ID = 1
	p1.Race, p1.Class = wow.RaceOrc, wow.ClassWarior
	p1.Init()
	p1.Location.X = 0
	p1.Location.Y = 0
	s1 := &spySender{}
	p1.Sender = s1
	m.AddPlayer(p1)

	p2 := player.NewPlayer()
	p2.ID = 2
	p2.Race, p2.Class = wow.RaceOrc, wow.ClassWarior
	p2.Init()
	p2.Location.X = 200 // Start out of range
	p2.Location.Y = 0
	s2 := &spySender{}
	p2.Sender = s2
	m.AddPlayer(p2)

	// First tick — out of range
	m.Update(50)
	assert.Equal(t, 0, s1.create)
	assert.Equal(t, 0, s2.create)

	// Move p2 into range
	p2.Location.X = 50

	// Second tick — now in range
	m.Update(50)
	assert.Greater(t, s1.create, 0, "p1 should see p2 after move")
	assert.Greater(t, s2.create, 0, "p2 should see p1 after move")
}

// TestGridIntegration_PlayerMovesOutOfRange verifies that when a player
// moves out of range, they receive a destroy packet.
func TestGridIntegration_PlayerMovesOutOfRange(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.InitVisibilityDistance() // 100y

	p1 := player.NewPlayer()
	p1.ID = 1
	p1.Race, p1.Class = wow.RaceOrc, wow.ClassWarior
	p1.Init()
	p1.Location.X = 0
	p1.Location.Y = 0
	s1 := &spySender{}
	p1.Sender = s1
	m.AddPlayer(p1)

	p2 := player.NewPlayer()
	p2.ID = 2
	p2.Race, p2.Class = wow.RaceOrc, wow.ClassWarior
	p2.Init()
	p2.Location.X = 50 // Start in range
	p2.Location.Y = 0
	s2 := &spySender{}
	p2.Sender = s2
	m.AddPlayer(p2)

	// First tick — in range, create sent
	m.Update(50)
	require.Greater(t, s1.create, 0)

	// Move p2 out of range
	p2.Location.X = 200

	// Second tick — out of range, destroy sent
	m.Update(50)
	assert.Greater(t, s1.destroy, 0, "p1 should receive destroy for p2")
}

// TestGridIntegration_NPCVisibility verifies that NPCs within range
// are visible to players.
func TestGridIntegration_NPCVisibility(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.InitVisibilityDistance()

	p := player.NewPlayer()
	p.ID = 1
	p.Race, p.Class = wow.RaceOrc, wow.ClassWarior
	p.Init()
	p.Location.X = 0
	p.Location.Y = 0
	s := &spySender{}
	p.Sender = s
	m.AddPlayer(p)

	// Add NPC near player
	npc := newTestNPC(100, 0, 0, 0)
	m.AddNPC(npc)

	// Run update cycle
	m.Update(50)

	// Player should see NPC
	assert.Greater(t, s.create, 0, "player should see nearby NPC")
}

// TestGridIntegration_FarVisibleNPC verifies that NPCs with Large visibility
// are visible from further away than normal NPCs.
func TestGridIntegration_FarVisibleNPC(t *testing.T) {
	m := NewMap(0, 0, nil)
	m.InitVisibilityDistance() // 100y

	p := player.NewPlayer()
	p.ID = 1
	p.Race, p.Class = wow.RaceOrc, wow.ClassWarior
	p.Init()
	p.Location.X = 0
	p.Location.Y = 0
	s := &spySender{}
	p.Sender = s
	m.AddPlayer(p)

	// Normal NPC at 150 yards — outside normal visibility
	normalNPC := newTestNPC(100, 150, 0, 0)
	m.AddNPC(normalNPC)

	// Far-visible NPC at 150 yards — within Large visibility (200y)
	farNPC := newTestNPC(101, 150, 0, 0)
	farNPC.Object.SetVisibilityOverrideType(object.VisibilityDistanceLarge)
	m.AddNPC(farNPC)

	// Run update cycle
	m.Update(50)

	// Player should NOT see normal NPC (150 > 100y)
	// Player SHOULD see far-visible NPC (150 < 200y)
	visNormal := m.visibilityTracker.IsVisible(p.GUID(), normalNPC.Object.GUID())
	visFar := m.visibilityTracker.IsVisible(p.GUID(), farNPC.Object.GUID())

	assert.False(t, visNormal, "normal NPC at 150y should NOT be visible")
	assert.True(t, visFar, "far-visible NPC at 150y SHOULD be visible")
}

// TestGridIntegration_DungeonVisibility verifies that dungeons use
// the 170y visibility range.
func TestGridIntegration_DungeonVisibility(t *testing.T) {
	entry := &MapEntry{InstanceType: 1}
	m := NewMap(389, 0, entry)
	m.InitVisibilityDistance()

	assert.InDelta(t, 170.0, m.visibilityRange, 0.1)

	p := player.NewPlayer()
	p.ID = 1
	p.Race, p.Class = wow.RaceOrc, wow.ClassWarior
	p.Init()
	p.Location.X = 0
	p.Location.Y = 0
	s := &spySender{}
	p.Sender = s
	m.AddPlayer(p)

	// NPC at 150 yards — within dungeon visibility (170y)
	npc := newTestNPC(100, 150, 0, 0)
	m.AddNPC(npc)

	m.Update(50)

	vis := m.visibilityTracker.IsVisible(p.GUID(), npc.Object.GUID())
	assert.True(t, vis, "NPC at 150y should be visible in 170y dungeon range")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testNPC is a minimal NPC for integration tests.
type testNPC struct {
	Object *object.Object
	ID     uint32
	X, Y, Z float32
}

func newTestNPC(id uint32, x, y, z float32) *testNPC {
	obj := object.NewObject()
	obj.InitValues(int(object.PlayerEnd))
	// Set a unique GUID based on the NPC ID (UnitGUID = 0xF1300)
	obj.SetGUID(wow.NewGUID(wow.UnitGUID, id))
	return &testNPC{
		Object: obj,
		ID:     id,
		X:      x,
		Y:      y,
		Z:      z,
	}
}

func (n *testNPC) GetGUID() wow.GUID {
	return n.Object.GUID()
}

// GetNPCObject satisfies the npcProvider interface used by sendNPCCreateToPlayer.
func (n *testNPC) GetNPCObject() *object.Object {
	return n.Object
}

// GetObject satisfies the objectProvider interface used by getNPCObject in updatePlayerVisibilityLocked.
func (n *testNPC) GetObject() *object.Object {
	return n.Object
}

func (n *testNPC) GetPosition() *player.WorldLocation {
	return &player.WorldLocation{X: n.X, Y: n.Y, Z: n.Z}
}
