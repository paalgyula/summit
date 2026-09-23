package movement

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
	MotionSlotIDLE      MovementSlot = 0 // Default movement (random, idle, waypoint)
	MotionSlotACTIVE    MovementSlot = 1 // Combat movement (chase, flee)
	MotionSlotCONTROLLED MovementSlot = 2 // Script-controlled movement
	MotionSlotMAX       MovementSlot = 3
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
