package movement

import (
	"time"

	"github.com/paalgyula/summit/pkg/wow"
)

// MovementGeneratorType identifies the kind of movement generator.
// Values match AzerothCore's MovementGeneratorType enum.
type MovementGeneratorType uint8

const (
	MotionTypeIDLE       MovementGeneratorType = 0
	MotionTypeRANDOM     MovementGeneratorType = 1
	MotionTypeWAYPOINT   MovementGeneratorType = 2
	MotionTypeCONFUSED   MovementGeneratorType = 4
	MotionTypeCHASE      MovementGeneratorType = 5
	MotionTypeHOME       MovementGeneratorType = 6
	MotionTypeFLIGHT     MovementGeneratorType = 7
	MotionTypePOINT      MovementGeneratorType = 8
	MotionTypeFLEEING    MovementGeneratorType = 9
	MotionTypeDISTRACT   MovementGeneratorType = 10
	MotionTypeASSISTANCE MovementGeneratorType = 11
	MotionTypeFOLLOW     MovementGeneratorType = 14
	MotionTypeROTATE     MovementGeneratorType = 15
	MotionTypeEFFECT     MovementGeneratorType = 16
	MotionTypeESCORT     MovementGeneratorType = 17
	MotionTypeFORMATION  MovementGeneratorType = 18
	MotionTypeNULL       MovementGeneratorType = 19
)

// MovementSlot represents the priority slot in the MotionMaster.
type MovementSlot uint8

const (
	MotionSlotIDLE       MovementSlot = 0 // Default movement (random, idle, waypoint)
	MotionSlotACTIVE     MovementSlot = 1 // Combat movement (chase, flee)
	MotionSlotCONTROLLED MovementSlot = 2 // Script-controlled movement
	MotionSlotMAX        MovementSlot = 3
)

// MovementGenerator is the interface that all movement generators implement.
// Each generator has a lifecycle: Initialize → Update (repeated) → Finalize.
type MovementGenerator interface {
	// Initialize is called when the generator is first placed in a slot.
	Initialize(owner interface{})

	// Update is called every tick. Returns false when the generator is done
	// and should be popped from the stack.
	Update(owner interface{}, diff uint32) bool

	// Finalize is called when the generator is removed from the stack.
	Finalize(owner interface{})

	// Reset is called when the generator is re-activated after being covered.
	Reset(owner interface{})

	// Type returns the generator type for identification.
	Type() MovementGeneratorType

	// GetSplineId returns the current spline ID (used by escort system).
	GetSplineId() uint32
}

// MovementOwner is the interface that NPCs must implement to use the MotionMaster.
// This avoids circular imports between the world and movement packages.
type MovementOwner interface {
	// GetSpawnPosition returns the creature's spawn position.
	GetSpawnPosition() (x, y, z float32)

	// GetCurrentPosition returns the creature's current position.
	GetCurrentPosition() (x, y, z float32)

	// IsAlive returns whether the creature is alive.
	IsAlive() bool

	// IsInCombat returns whether the creature is in combat.
	IsInCombat() bool

	// GetDefaultMovementType returns the creature's default movement type from DB.
	GetDefaultMovementType() uint8

	// GetWanderDistance returns the creature's wander distance.
	GetWanderDistance() float32

	// GetEntry returns the creature's entry ID (for logging).
	GetEntry() uint32

	// GetGUID returns the creature's GUID.
	GetGUID() wow.GUID

	// GetCurrentSpeed returns the current movement speed for the given move type.
	GetCurrentSpeed(moveType wow.MoveType) float32

	// SendPacket sends a packet to all players who can see this creature.
	SendPacket(pkt *wow.Packet)

	// MoveTo starts movement toward the given destination.
	// The sendPacket callback is called with each movement packet.
	MoveTo(destX, destY, destZ float32, now time.Time, flags uint32, sendPacket func(pkt *wow.Packet))

	// StopMoving stops active movement.
	StopMoving(sendPacket func(pkt *wow.Packet))

	// SetOrientation sets the creature's facing direction.
	SetOrientation(o float32)
}

// String returns the human-readable name of the generator type.
func (t MovementGeneratorType) String() string {
	switch t {
	case MotionTypeIDLE:
		return "Idle"
	case MotionTypeRANDOM:
		return "Random"
	case MotionTypeWAYPOINT:
		return "Waypoint"
	case MotionTypeCONFUSED:
		return "Confused"
	case MotionTypeCHASE:
		return "Chase"
	case MotionTypeHOME:
		return "Home"
	case MotionTypeFLIGHT:
		return "Flight"
	case MotionTypePOINT:
		return "Point"
	case MotionTypeFLEEING:
		return "Fleeing"
	case MotionTypeDISTRACT:
		return "Distract"
	case MotionTypeASSISTANCE:
		return "Assistance"
	case MotionTypeFOLLOW:
		return "Follow"
	case MotionTypeROTATE:
		return "Rotate"
	case MotionTypeEFFECT:
		return "Effect"
	case MotionTypeESCORT:
		return "Escort"
	case MotionTypeFORMATION:
		return "Formation"
	case MotionTypeNULL:
		return "NULL"
	default:
		return "Unknown"
	}
}

// IsStatic returns true if the generator is the idle generator (a singleton).
func IsStatic(gen MovementGenerator) bool {
	return gen != nil && gen.Type() == MotionTypeIDLE
}
