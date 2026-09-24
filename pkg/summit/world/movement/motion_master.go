package movement

// CleanFlag controls deferred cleanup behavior during updates.
type CleanFlag uint8

const (
	MMCF_NONE   CleanFlag = 0x00
	MMCF_UPDATE CleanFlag = 0x01 // Called from update
	MMCF_RESET  CleanFlag = 0x02 // Flag if need top()->Reset()
	MMCF_INUSE  CleanFlag = 0x04 // Flag if in MotionMaster::UpdateMotion
)

// MotionMaster manages a stack of MovementGenerators for a creature.
// It mirrors AzerothCore's MotionMaster with a 3-slot stack:
//   - IDLE (slot 0): default movement (random, idle, waypoint)
//   - ACTIVE (slot 1): combat movement (chase, flee)
//   - CONTROLLED (slot 2): script-controlled movement
type MotionMaster struct {
	impl     [MotionSlotMAX]MovementGenerator
	top      int8 // index of the highest active slot, -1 when empty
	owner    MovementOwner
	cleanFlag CleanFlag
	expList  []MovementGenerator
	needInit [MotionSlotMAX]bool
}

// NewMotionMaster creates a MotionMaster for the given owner.
func NewMotionMaster(owner MovementOwner) *MotionMaster {
	mm := &MotionMaster{
		top:   -1,
		owner: owner,
	}
	for i := range mm.needInit {
		mm.needInit[i] = true
	}
	return mm
}

// empty returns true when no generators are in the stack.
func (mm *MotionMaster) empty() bool {
	return mm.top < 0
}

// size returns the number of active generators.
func (mm *MotionMaster) size() int {
	return int(mm.top) + 1
}

// top returns the highest active generator, or nil if empty.
func (mm *MotionMaster) Top() MovementGenerator {
	if mm.empty() {
		return nil
	}
	return mm.impl[mm.top]
}

// GetMotionSlot returns the generator at the given slot.
func (mm *MotionMaster) GetMotionSlot(slot MovementSlot) MovementGenerator {
	return mm.impl[slot]
}

// pop removes the top generator from the stack without finalizing it.
func (mm *MotionMaster) pop() {
	if mm.empty() {
		return
	}
	mm.impl[mm.top] = nil
	for !mm.empty() && mm.impl[mm.top] == nil {
		mm.top--
	}
}

// needInitTop returns true if the top generator needs initialization.
func (mm *MotionMaster) needInitTop() bool {
	if mm.empty() {
		return false
	}
	return mm.needInit[mm.top]
}

// InitTop calls Initialize on the top generator.
func (mm *MotionMaster) InitTop() {
	top := mm.Top()
	if top != nil {
		top.Initialize(mm.owner)
		mm.needInit[mm.top] = false
	}
}

// Update is called every world tick. It updates the top generator and
// handles deferred cleanup.
func (mm *MotionMaster) Update(diff uint32) {
	if mm.owner == nil {
		return
	}

	if mm.empty() {
		return
	}

	mm.cleanFlag |= MMCF_INUSE

	// Update the top generator
	mm.cleanFlag |= MMCF_UPDATE
	top := mm.Top()
	if top != nil && !top.Update(mm.owner, diff) {
		// Generator finished, expire it
		mm.cleanFlag &^= MMCF_UPDATE
		mm.MovementExpired(true)
	} else {
		mm.cleanFlag &^= MMCF_UPDATE
	}

	// Process deferred deletions
	if mm.expList != nil {
		for _, mg := range mm.expList {
			mm.directDelete(mg)
		}
		mm.expList = nil

		if mm.empty() {
			mm.Initialize()
		} else if mm.needInitTop() {
			mm.InitTop()
		} else if mm.cleanFlag&MMCF_RESET != 0 {
			top := mm.Top()
			if top != nil {
				top.Reset(mm.owner)
			}
		}
		mm.cleanFlag &^= MMCF_RESET
	}

	mm.cleanFlag &^= MMCF_INUSE
}

// Initialize clears all generators and sets up the default movement.
func (mm *MotionMaster) Initialize() {
	for !mm.empty() {
		top := mm.Top()
		mm.pop()
		if top != nil {
			mm.directDelete(top)
		}
	}
	mm.InitDefault()
}

// InitDefault sets the default movement generator based on the owner's MovementType.
func (mm *MotionMaster) InitDefault() {
	if mm.owner == nil {
		return
	}

	var gen MovementGenerator
	movementType := mm.owner.GetDefaultMovementType()

	switch movementType {
	case 1: // MotionTypeRandom
		gen = NewRandomGenerator(mm.owner.GetWanderDistance())
	case 2: // MotionTypeWaypoint
		// Waypoint generator will be created when path data is available
		gen = NewIdleGenerator()
	default: // 0 or unknown = Idle
		gen = NewIdleGenerator()
	}

	mm.Mutate(gen, MotionSlotIDLE)
}

// Mutate replaces the generator at the given slot.
// If called during an update, the old generator is deferred for deletion.
func (mm *MotionMaster) Mutate(gen MovementGenerator, slot MovementSlot) {
	delayed := mm.cleanFlag&MMCF_UPDATE != 0

	// Clear existing generator at this slot
	if old := mm.impl[slot]; old != nil {
		mm.impl[slot] = nil
		for !mm.empty() && mm.Top() == nil {
			mm.top--
		}

		if delayed {
			mm.delayedDelete(old)
		} else {
			mm.directDelete(old)
		}
	}

	// Set the new generator
	if int8(slot) > mm.top {
		mm.top = int8(slot)
	}

	mm.impl[slot] = gen
	if int8(slot) < mm.top {
		mm.needInit[slot] = true
	} else {
		mm.needInit[slot] = false
		if gen != nil {
			gen.Initialize(mm.owner)
		}
	}
}

// Clear removes all generators except the idle one.
func (mm *MotionMaster) Clear(reset bool) {
	if mm.cleanFlag&MMCF_UPDATE != 0 {
		if reset {
			mm.cleanFlag |= MMCF_RESET
		} else {
			mm.cleanFlag &^= MMCF_RESET
		}
		mm.delayedClean()
	} else {
		mm.directClean(reset)
	}
}

// MovementExpired pops the top generator and resets the one below.
func (mm *MotionMaster) MovementExpired(reset bool) {
	if mm.cleanFlag&MMCF_UPDATE != 0 {
		if reset {
			mm.cleanFlag |= MMCF_RESET
		} else {
			mm.cleanFlag &^= MMCF_RESET
		}
		mm.delayedExpire()
	} else {
		mm.directExpire(reset)
	}
}

// MoveIdle sets the idle movement generator in the IDLE slot.
func (mm *MotionMaster) MoveIdle() {
	if mm.empty() || !IsStatic(mm.Top()) {
		mm.Mutate(NewIdleGenerator(), MotionSlotIDLE)
	}
}

// MoveRandom sets a random movement generator in the IDLE slot.
func (mm *MotionMaster) MoveRandom(wanderDistance float32) {
	if mm.owner == nil {
		return
	}
	mm.Mutate(NewRandomGenerator(wanderDistance), MotionSlotIDLE)
}

// --- Deferred cleanup methods ---

func (mm *MotionMaster) directClean(reset bool) {
	for mm.size() > 1 {
		top := mm.Top()
		mm.pop()
		if top != nil {
			mm.directDelete(top)
		}
	}

	if mm.empty() {
		return
	}

	if mm.needInitTop() {
		mm.InitTop()
	} else if reset {
		top := mm.Top()
		if top != nil {
			top.Reset(mm.owner)
		}
	}
}

func (mm *MotionMaster) delayedClean() {
	for mm.size() > 1 {
		top := mm.Top()
		mm.pop()
		if top != nil {
			mm.delayedDelete(top)
		}
	}
}

func (mm *MotionMaster) directExpire(reset bool) {
	if mm.size() > 1 {
		top := mm.Top()
		mm.pop()
		if top != nil {
			mm.directDelete(top)
		}
	}

	for !mm.empty() && mm.Top() == nil {
		mm.top--
	}

	if mm.empty() {
		mm.Initialize()
	} else if mm.needInitTop() {
		mm.InitTop()
	} else if reset {
		top := mm.Top()
		if top != nil {
			top.Reset(mm.owner)
		}
	}
}

func (mm *MotionMaster) delayedExpire() {
	if mm.size() > 1 {
		top := mm.Top()
		mm.pop()
		if top != nil {
			mm.delayedDelete(top)
		}
	}

	for !mm.empty() && mm.Top() == nil {
		mm.top--
	}
}

func (mm *MotionMaster) directDelete(gen MovementGenerator) {
	if IsStatic(gen) {
		return
	}
	gen.Finalize(mm.owner)
	// In Go, garbage collection handles memory; no manual delete needed
}

func (mm *MotionMaster) delayedDelete(gen MovementGenerator) {
	if IsStatic(gen) {
		return
	}
	if mm.expList == nil {
		mm.expList = make([]MovementGenerator, 0)
	}
	mm.expList = append(mm.expList, gen)
}

// GetCurrentMovementGeneratorType returns the type of the top generator.
func (mm *MotionMaster) GetCurrentMovementGeneratorType() MovementGeneratorType {
	if mm.empty() {
		return MotionTypeIDLE
	}
	top := mm.Top()
	if top == nil {
		return MotionTypeIDLE
	}
	return top.Type()
}

// HasMovementGeneratorType returns true if any slot has a generator of the given type.
func (mm *MotionMaster) HasMovementGeneratorType(t MovementGeneratorType) bool {
	if mm.empty() && t == MotionTypeIDLE {
		return true
	}
	for i := mm.top; i >= 0; i-- {
		if mm.impl[i] != nil && mm.impl[i].Type() == t {
			return true
		}
	}
	return false
}

// GetDestination returns the current movement destination, if known.
func (mm *MotionMaster) GetDestination() (x, y, z float32, ok bool) {
	if mm.empty() {
		return 0, 0, 0, false
	}
	// Generators can implement this interface for path info
	type destinationProvider interface {
		GetDestination() (float32, float32, float32, bool)
	}
	top := mm.Top()
	if dp, ok := top.(destinationProvider); ok {
		return dp.GetDestination()
	}
	return 0, 0, 0, false
}
