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

// GetUpdateBlockCount returns the number of 32-bit update blocks that
// contain at least one set bit. This is the value the client uses to
// determine how many uint32 mask words to read.
func (um *UpdateMask) GetUpdateBlockCount() uint32 {
	if um.blocks == 0 {
		return 0
	}

	// Walk backwards through the byte array to find the last non-zero byte.
	var lastNonZero int = -1
	for i := int(um.blocks*4) - 1; i >= 0; i-- {
		if um.updateMask[i] != 0 {
			lastNonZero = i
			break
		}
	}

	if lastNonZero < 0 {
		return 0
	}

	// Convert byte index to 32-bit block index (+1 for count).
	return uint32(lastNonZero/4) + 1
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
