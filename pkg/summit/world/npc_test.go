package world

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/quest"
	"github.com/paalgyula/summit/pkg/wow"
)

// The grid visibility path builds NPC create blocks with the per-type field
// visibility table, so the NPC's object must identify itself as a unit or the
// client only receives the object header (no display id, health or npc flags).
func TestNPCCreateMaskContainsUnitFields(t *testing.T) {
	npc := NewNPC(3098, "Mottled Boar", 903, 14, 1, 42, 0, 0, 0, 0, 1, quest.NpcFlagQuestGiver)

	if got := npc.Object.ObjectTypeID(); got != wow.TypeIDUnit {
		t.Fatalf("object type id = %v, want %v", got, wow.TypeIDUnit)
	}

	viewer := object.NewObject()
	viewer.InitValues(int(object.PlayerEnd))
	viewer.SetObjectTypeID(wow.TypeIDPlayer)

	mask := npc.Object.BuildFilteredUpdateMask(viewer, false)

	for _, f := range []object.UpdateField{
		object.ObjectFieldEntry,
		object.UnitFieldDisplayid,
		object.UnitFieldHealth,
		object.UnitFieldMaxhealth,
		object.UnitFieldLevel,
		object.UnitFieldFactiontemplate,
		object.UnitFieldFlags,
		object.UnitNpcFlags,
	} {
		if !mask.GetBit(uint32(f)) {
			t.Errorf("field %d missing from NPC create mask", f)
		}
	}
}

func TestNPCThreatManagement(t *testing.T) {
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 200, 0, 0, 0, 0, 0, 0)

	attacker1 := NewNPC(2001, "Attacker 1", 49, 1, 5, 200, 0, 0, 0, 0, 0, 0)
	attacker2 := NewNPC(2002, "Attacker 2", 49, 1, 5, 200, 0, 0, 0, 0, 0, 0)

	// Add initial threat
	npc.AddThreat(attacker1, 50.0)

	if got := npc.GetThreat(attacker1.GUID()); got != 50.0 {
		t.Fatalf("expected attacker1 threat 50.0, got %f", got)
	}

	if !npc.IsInCombatState() {
		t.Fatal("expected NPC to be in combat")
	}

	if npc.GetVictim() != attacker1 {
		t.Fatal("expected victim to be attacker1")
	}

	// Add higher threat from attacker 2
	npc.AddThreat(attacker2, 100.0)

	if got := npc.GetThreat(attacker2.GUID()); got != 100.0 {
		t.Fatalf("expected attacker2 threat 100.0, got %f", got)
	}

	if npc.GetVictim() != attacker2 {
		t.Fatal("expected victim to switch to attacker2 due to higher threat")
	}

	// Clear threat
	npc.ClearThreat()
	if got := npc.GetThreat(attacker1.GUID()); got != 0 {
		t.Fatalf("expected 0 threat after ClearThreat, got %f", got)
	}
	if got := npc.GetThreat(attacker2.GUID()); got != 0 {
		t.Fatalf("expected 0 threat after ClearThreat, got %f", got)
	}
}

func TestNPCCombatMeleeSwing(t *testing.T) {
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 200, 10, 10, 0, 0, 0, 0)
	target := NewNPC(2001, "Player Mock", 49, 1, 5, 200, 11, 10, 0, 0, 0, 0)

	npc.Attack(target, true)
	npc.AddThreat(target, 50.0)

	initialHealth := target.GetHealth()
	packetSent := false

	now := time.Now()
	npc.NextAttackTime = now.UnixMilli() - 100 // ready to swing

	ProcessNPCCombatTick(npc, now, func(pkt *wow.Packet) {
		if pkt != nil && pkt.Opcode() == wow.ServerAttackerstateupdate {
			packetSent = true
		}
	})

	if target.GetHealth() >= initialHealth {
		t.Fatalf("expected target health to decrease from %d, got %d", initialHealth, target.GetHealth())
	}

	if !packetSent {
		t.Fatal("expected SMSG_ATTACKERSTATEUPDATE packet to be generated")
	}

	if npc.NextAttackTime <= now.UnixMilli() {
		t.Fatal("expected NextAttackTime to be rescheduled in future")
	}
}

func TestBuildMonsterMovePacket(t *testing.T) {
	guid := wow.NewGUID(wow.UnitGUID, 1001)
	pkt := BuildMonsterMovePacket(guid, 10, 20, 30, 40, 50, 60, 1, 1500, SplineFlagWalkMode)

	if pkt.Opcode() != wow.ServerMonsterMove {
		t.Fatalf("expected opcode SMSG_MONSTER_MOVE, got %s", pkt.Opcode())
	}

	stopPkt := BuildMonsterMoveStopPacket(guid, 10, 20, 30, 2)
	if stopPkt.Opcode() != wow.ServerMonsterMove {
		t.Fatalf("expected opcode SMSG_MONSTER_MOVE, got %s", stopPkt.Opcode())
	}
}

func TestNPCMovementInterpolation(t *testing.T) {
	npc := NewNPC(1001, "Boar", 903, 14, 1, 50, 0, 0, 0, 0, 0, 0)
	now := time.Now()

	npc.MoveTo(100, 0, 0, now, SplineFlagRunMode, nil)

	if !npc.IsMoving {
		t.Fatal("expected NPC to be moving")
	}

	// 50% elapsed
	halfTime := now.Add(time.Duration(npc.MoveDurationMs/2) * time.Millisecond)
	npc.UpdatePositionFromSpline(halfTime)

	if npc.X < 40 || npc.X > 60 {
		t.Fatalf("expected X ~50 at midpoint, got %f", npc.X)
	}

	// 100% elapsed
	fullTime := now.Add(time.Duration(npc.MoveDurationMs+10) * time.Millisecond)
	npc.UpdatePositionFromSpline(fullTime)

	if npc.IsMoving {
		t.Fatal("expected NPC to stop moving after duration elapsed")
	}
	if npc.X != 100 {
		t.Fatalf("expected X = 100 at completion, got %f", npc.X)
	}
}

func TestNPCChaseMovement(t *testing.T) {
	npc := NewNPC(1001, "Defias Thug", 49, 14, 5, 200, 0, 0, 0, 0, 0, 0)
	target := NewNPC(2001, "Player", 49, 1, 5, 200, 20, 0, 0, 0, 0, 0)

	npc.Attack(target, true)
	npc.ChaseTarget = target

	var packetSent bool
	now := time.Now()

	ProcessNPCChase(npc, now, func(pkt *wow.Packet) {
		if pkt != nil && pkt.Opcode() == wow.ServerMonsterMove {
			packetSent = true
		}
	})

	if !npc.IsMoving {
		t.Fatal("expected NPC to begin moving towards target")
	}
	if !packetSent {
		t.Fatal("expected SMSG_MONSTER_MOVE to be sent")
	}
}

func TestNPCWaypointMovement(t *testing.T) {
	npc := NewNPC(1001, "Patrol Guard", 49, 14, 5, 200, 0, 0, 0, 0, 0, 0)
	npc.MovementType = store.MotionTypeWaypoint
	npc.WanderRadius = 0

	// Set up a 3-point patrol path with run speed
	npc.WaypointPath = &store.WaypointPath{
		PathID: 1,
		Points: []store.Waypoint{
			{Point: 1, X: 10, Y: 0, Z: 0, MoveType: 1}, // run
			{Point: 2, X: 20, Y: 10, Z: 0, MoveType: 1},
			{Point: 3, X: 0, Y: 10, Z: 0, Delay: 5000, MoveType: 0}, // walk + 5s delay
		},
	}

	now := time.Now()
	var packetsSent int

	// First tick: should move to waypoint 1
	ProcessNPCWaypoint(npc, now, func(pkt *wow.Packet) {
		if pkt != nil && pkt.Opcode() == wow.ServerMonsterMove {
			packetsSent++
		}
	})

	if !npc.IsMoving {
		t.Fatal("expected NPC to start moving to first waypoint")
	}
	if packetsSent != 1 {
		t.Fatalf("expected 1 SMSG_MONSTER_MOVE, got %d", packetsSent)
	}

	// Complete the spline instantly (move time 0)
	instantly := now.Add(time.Duration(npc.MoveDurationMs+1) * time.Millisecond)
	npc.UpdatePositionFromSpline(instantly)

	// Second tick: should move to waypoint 2
	ProcessNPCWaypoint(npc, instantly, func(pkt *wow.Packet) {
		if pkt != nil && pkt.Opcode() == wow.ServerMonsterMove {
			packetsSent++
		}
	})

	if !npc.IsMoving {
		t.Fatal("expected NPC to start moving to second waypoint")
	}
	if packetsSent != 2 {
		t.Fatalf("expected 2 SMSG_MONSTER_MOVE total, got %d", packetsSent)
	}

	// Complete the spline
	instantly2 := instantly.Add(time.Duration(npc.MoveDurationMs+1) * time.Millisecond)
	npc.UpdatePositionFromSpline(instantly2)

	// Third tick: should move to waypoint 3 (which has a delay)
	ProcessNPCWaypoint(npc, instantly2, func(pkt *wow.Packet) {
		if pkt != nil && pkt.Opcode() == wow.ServerMonsterMove {
			packetsSent++
		}
	})

	if !npc.IsMoving {
		t.Fatal("expected NPC to start moving to third waypoint")
	}
	if packetsSent != 3 {
		t.Fatalf("expected 3 SMSG_MONSTER_MOVE total, got %d", packetsSent)
	}

	// Complete the spline
	instantly3 := instantly2.Add(time.Duration(npc.MoveDurationMs+1) * time.Millisecond)
	npc.UpdatePositionFromSpline(instantly3)

	// Should be waiting on delay now (WaypointDelayEnd was set when we started moving to wp3)
	duringDelay := instantly3.Add(1 * time.Second) // 1s after arrival, well within 5s delay
	ProcessNPCWaypoint(npc, duringDelay, func(pkt *wow.Packet) {
		// Should NOT send any packet during delay
		if pkt != nil {
			packetsSent++
		}
	})

	if npc.IsMoving {
		t.Fatal("expected NPC to NOT be moving during delay")
	}
	if packetsSent != 3 {
		t.Fatalf("expected still 3 packets (waiting on delay), got %d", packetsSent)
	}

	// After delay expires, should advance to next waypoint (wraps to 0)
	afterDelay := instantly3.Add(6 * time.Second) // well after 5s delay
	ProcessNPCWaypoint(npc, afterDelay, func(pkt *wow.Packet) {
		if pkt != nil && pkt.Opcode() == wow.ServerMonsterMove {
			packetsSent++
		}
	})

	if !npc.IsMoving {
		t.Fatal("expected NPC to resume moving after delay expires")
	}
	if packetsSent != 4 {
		t.Fatalf("expected 4 SMSG_MONSTER_MOVE total, got %d", packetsSent)
	}
}

func TestNPCWaypointNoMovementWhenIdle(t *testing.T) {
	npc := NewNPC(1001, "Idle NPC", 49, 14, 1, 50, 5, 5, 0, 0, 0, 0)
	npc.MovementType = store.MotionTypeIdle
	npc.WanderRadius = 0

	now := time.Now()
	ProcessNPCWaypoint(npc, now, func(pkt *wow.Packet) {
		t.Fatal("should not send any packet for idle NPC")
	})
}

func TestNPCRandomMovementFromSpawn(t *testing.T) {
	// Simulate a creature spawn with wander_distance set
	spawn := &store.CreatureSpawn{
		SpawnID:        1001,
		Entry:          3098,
		MapID:          0,
		PosX:           100,
		PosY:           200,
		PosZ:           50,
		Orientation:    1.0,
		MovementType:   store.MotionTypeRandom,
		WanderDistance: 15.0,
	}
	tmpl := &store.CreatureTemplate{
		Entry:       3098,
		Name:        "Wandering Boar",
		Faction:     14,
		MaxLevel:    5,
		ModelIDs:    []uint32{903},
		HealthMultiplier: 1.0,
	}

	npc := NewNPCFromSpawn(spawn, tmpl)

	if npc.MovementType != store.MotionTypeRandom {
		t.Fatalf("expected MovementType %d (random), got %d", store.MotionTypeRandom, npc.MovementType)
	}
	if npc.WanderRadius != 15.0 {
		t.Fatalf("expected WanderRadius 15.0 from spawn, got %f", npc.WanderRadius)
	}
}

func TestNPCWaypointMovementFromAddon(t *testing.T) {
	// Simulate the NPC as it would be loaded from DB with creature_addon path
	npc := NewNPC(2001, "Patrol", 49, 14, 5, 200, 5, 5, 0, 0, 0, 0)
	npc.MovementType = store.MotionTypeWaypoint
	npc.WanderRadius = 0
	npc.WaypointPath = &store.WaypointPath{
		PathID: 42,
		Points: []store.Waypoint{
			{Point: 1, X: 10, Y: 10, Z: 0, MoveType: 1},
			{Point: 2, X: 20, Y: 20, Z: 0, MoveType: 1},
		},
	}

	now := time.Now()
	ProcessNPCWander(npc, now, func(pkt *wow.Packet) {
		t.Fatal("waypoint NPC should not wander randomly")
	})

	ProcessNPCWaypoint(npc, now, func(pkt *wow.Packet) {
		if pkt == nil || pkt.Opcode() != wow.ServerMonsterMove {
			t.Fatal("expected SMSG_MONSTER_MOVE for waypoint movement")
		}
	})

	if !npc.IsMoving {
		t.Fatal("expected NPC to start waypoint movement")
	}
}

func TestNPCGetAttackDistance(t *testing.T) {
	// Base aggro radius 20
	npc := NewNPC(1001, "Defias Thug", 49, 14, 10, 200, 0, 0, 0, 0, 0, 0)
	npc.AggroRadius = 20.0

	// Target same level (10)
	targetSame := NewNPC(2001, "Target Same", 49, 1, 10, 200, 0, 0, 0, 0, 0, 0)
	if d := npc.GetAttackDistance(targetSame); d != 20.0 {
		t.Fatalf("expected 20.0, got %f", d)
	}

	// Target higher level (15 -> levelDiff = +5, dist = 20 - 5 = 15)
	targetHigh := NewNPC(2002, "Target High", 49, 1, 15, 200, 0, 0, 0, 0, 0, 0)
	if d := npc.GetAttackDistance(targetHigh); d != 15.0 {
		t.Fatalf("expected 15.0, got %f", d)
	}

	// Target very high level (50 -> levelDiff = +40, dist = 20 - 40 = -20 -> clamped to 5.0 min)
	targetVeryHigh := NewNPC(2003, "Target Very High", 49, 1, 50, 200, 0, 0, 0, 0, 0, 0)
	if d := npc.GetAttackDistance(targetVeryHigh); d != 5.0 {
		t.Fatalf("expected min 5.0, got %f", d)
	}

	// Target lower level (5 -> levelDiff = -5, dist = 20 - (-5) = 25)
	targetLow := NewNPC(2004, "Target Low", 49, 1, 5, 200, 0, 0, 0, 0, 0, 0)
	if d := npc.GetAttackDistance(targetLow); d != 25.0 {
		t.Fatalf("expected 25.0, got %f", d)
	}

	// Target very low level (levelDiff clamped at -25 -> dist = 20 - (-25) = 45 -> clamped to 45.0 max)
	targetVeryLow := NewNPC(2005, "Target Very Low", 49, 1, 1, 200, 0, 0, 0, 0, 0, 0)
	npcBoss := NewNPC(1002, "Boss", 49, 14, 80, 200, 0, 0, 0, 0, 0, 0)
	npcBoss.AggroRadius = 20.0
	if d := npcBoss.GetAttackDistance(targetVeryLow); d != 45.0 {
		t.Fatalf("expected max 45.0, got %f", d)
	}
}

func TestNPCCanStartAttack(t *testing.T) {
	// Hostile NPC at (0, 0, 0), level 10, faction 14
	npc := NewNPC(1001, "Defias Thug", 49, 14, 10, 200, 0, 0, 0, 0, 0, 0)
	npc.AggroRadius = 20.0

	// Friendly NPC (faction 14) -> should not attack
	friendly := NewNPC(1002, "Defias Ally", 49, 14, 10, 200, 5, 0, 0, 0, 0, 0)
	if npc.CanStartAttack(friendly, false) {
		t.Fatal("expected CanStartAttack to return false for friendly target")
	}

	// Hostile target (faction 1 - player faction) within aggro distance (10y away)
	hostile := NewNPC(2001, "Hostile Player Mock", 49, 1, 10, 200, 10, 0, 0, 0, 0, 0)
	if !npc.CanStartAttack(hostile, false) {
		t.Fatal("expected CanStartAttack to return true for in-range hostile target")
	}

	// Hostile target too far away (30y away, aggro distance is 20y)
	farHostile := NewNPC(2002, "Far Hostile", 49, 1, 10, 200, 30, 0, 0, 0, 0, 0)
	if npc.CanStartAttack(farHostile, false) {
		t.Fatal("expected CanStartAttack to return false for out-of-range hostile target")
	}
	// Force attack should bypass distance check
	if !npc.CanStartAttack(farHostile, true) {
		t.Fatal("expected CanStartAttack with force=true to return true for out-of-range target")
	}

	// Hostile target with excessive Z difference (> 11.0 yards)
	highHostile := NewNPC(2003, "High Hostile", 49, 1, 10, 200, 5, 0, 15, 0, 0, 0)
	if npc.CanStartAttack(highHostile, false) {
		t.Fatal("expected CanStartAttack to return false when Z diff > 11.0 yards")
	}

	// Dead target
	deadHostile := NewNPC(2004, "Dead Hostile", 49, 1, 10, 200, 5, 0, 0, 0, 0, 0)
	deadHostile.Health = 0
	if npc.CanStartAttack(deadHostile, false) {
		t.Fatal("expected CanStartAttack to return false for dead target")
	}
}

func TestNPCMovementPacketSender(t *testing.T) {
	npc := NewNPC(1001, "Defias Thug", 49, 14, 10, 200, 0, 0, 0, 0, 0, 0)

	var sentPackets []*wow.Packet
	npc.PacketSender = func(pkt *wow.Packet) {
		sentPackets = append(sentPackets, pkt)
	}

	// MoveTo with nil sendPacket should use PacketSender
	now := time.Now()
	npc.MoveTo(10, 20, 0, now, 0, nil)
	if len(sentPackets) != 1 {
		t.Fatalf("expected 1 packet sent, got %d", len(sentPackets))
	}
	if sentPackets[0].Opcode() != wow.ServerMonsterMove {
		t.Fatalf("expected ServerMonsterMove opcode, got %v", sentPackets[0].Opcode())
	}

	// StopMoving with nil sendPacket should also use PacketSender
	npc.StopMoving(nil)
	if len(sentPackets) != 2 {
		t.Fatalf("expected 2 packets sent, got %d", len(sentPackets))
	}
	if sentPackets[1].Opcode() != wow.ServerMonsterMove {
		t.Fatalf("expected ServerMonsterMove stop opcode, got %v", sentPackets[1].Opcode())
	}
}

