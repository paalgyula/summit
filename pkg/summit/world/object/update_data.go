package object

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// UpdateData accumulates update blocks and builds an SMSG_UPDATE_OBJECT packet.
// Matches AzerothCore's UpdateData class.
type UpdateData struct {
	blockCount      uint32
	outOfRangeGUIDs []wow.GUID
	data            []byte
}

// NewUpdateData creates an empty UpdateData.
func NewUpdateData() *UpdateData {
	return &UpdateData{
		outOfRangeGUIDs: make([]wow.GUID, 0, 15),
	}
}

// AddOutOfRangeGUID records a GUID that has gone out of range.
func (ud *UpdateData) AddOutOfRangeGUID(guid wow.GUID) {
	ud.outOfRangeGUIDs = append(ud.outOfRangeGUIDs, guid)
}

// AddUpdateBlock appends a raw update block and increments the block count.
func (ud *UpdateData) AddUpdateBlock(block []byte) {
	ud.data = append(ud.data, block...)
	ud.blockCount++
}

// HasData returns true if there are any blocks or out-of-range GUIDs.
func (ud *UpdateData) HasData() bool {
	return ud.blockCount > 0 || len(ud.outOfRangeGUIDs) > 0
}

// Clear resets the UpdateData to empty.
func (ud *UpdateData) Clear() {
	ud.data = ud.data[:0]
	ud.outOfRangeGUIDs = ud.outOfRangeGUIDs[:0]
	ud.blockCount = 0
}

// BuildPacket constructs the final SMSG_UPDATE_OBJECT packet.
// Format:
//
//	uint32 blockCount (+ 1 if out-of-range GUIDs exist)
//	[if out-of-range GUIDs:]
//	  uint8  UPDATETYPE_OUT_OF_RANGE_OBJECTS (4)
//	  uint32 guidCount
//	  packedGuid[guidCount]
//	[update blocks...]
func (ud *UpdateData) BuildPacket() *wow.Packet {
	pkt := wow.NewPacket(wow.ServerUpdateObject)

	hasOutOfRange := len(ud.outOfRangeGUIDs) > 0

	// Block count: +1 for the out-of-range pseudo-block
	totalBlocks := ud.blockCount
	if hasOutOfRange {
		totalBlocks++
	}

	_ = pkt.Write(totalBlocks)

	// Out-of-range block (must come first, before normal blocks)
	if hasOutOfRange {
		_ = pkt.WriteOne(wow.UpdateTypeOutOfRangeObjects)
		_ = pkt.Write(uint32(len(ud.outOfRangeGUIDs)))

		for _, guid := range ud.outOfRangeGUIDs {
			pkt.WriteBytes(guid.Pack())
		}
	}

	// Append all normal update blocks
	pkt.WriteBytes(ud.data)

	return pkt
}
