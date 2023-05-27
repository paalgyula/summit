package object

type UpdateMask struct {
	updateMask []uint8
	count      uint32
	blocks     uint32
}

// SetBit sets the bit at the given index in the UpdateMask, if the
// index is less than the count.
//
// index uint32: the index of the bit to be set.
func (um *UpdateMask) SetBit(index uint32) {
	if index < um.count {
		(um.updateMask[index>>3]) |= 1 << (index & 0x7)
	}
}

// UnsetBit unsets a bit at the given index in the UpdateMask.
//
// index uint32: the index of the bit to unset.
func (um *UpdateMask) UnsetBit(index uint32) {
	if index < um.count {
		(um.updateMask[index>>3]) &= (0xFF ^ (1 << (index & 0x7)))
	}
}

// GetBit returns a boolean indicating whether the bit at the given index is set in the UpdateMask.
//
// index: The index of the bit to check.
// Returns a boolean indicating whether the bit is set.
func (um *UpdateMask) GetBit(index uint32) bool {
	if index < um.count {
		return ((um.updateMask[index>>3]) & (1 << (index & 0x7))) != 0
	}
	return false
}

// GetUpdateBlockCount returns the number of update blocks in the UpdateMask.
//
// No parameters.
// uint32.
func (um *UpdateMask) GetUpdateBlockCount() uint32 {
	var x uint32
	for x = um.blocks - 1; x > 0; x-- {
		if um.updateMask[x] != 0 {
			break
		}
	}
	return (x + 1)
}

func (um *UpdateMask) BlockCount() uint32 {
	return um.blocks
}

func (um *UpdateMask) Length() uint32 {
	return (um.blocks * 4)
}

func (um *UpdateMask) Count() uint32 {
	return um.count
}

func (um *UpdateMask) Mask() []uint8 {
	return um.updateMask
}

// SetCount sets the number of values in the UpdateMask, and updates
// the internal block representation accordingly.
//
// valuesCount uint32: the number of values in the UpdateMask.
func (um *UpdateMask) SetCount(valuesCount uint32) {
	if um.updateMask != nil {
		um.updateMask = nil
	}

	um.count = valuesCount
	um.blocks = um.count >> 5
	if um.count&31 != 0 {
		um.blocks++
	}

	um.updateMask = make([]uint8, um.blocks*4)
	for i := range um.updateMask {
		um.updateMask[i] = 0
	}
}

func (um *UpdateMask) Clear() {
	if um.updateMask != nil {
		for i := range um.updateMask {
			um.updateMask[i] = 0
		}
	}
}
