package object

import (
	"bytes"
	"encoding/binary"
)

// UpdateBlockBuffer is a convenience wrapper around bytes.Buffer for building
// raw update blocks (not full packets). It uses little-endian encoding for all
// writes, matching the WoW wire format.
type UpdateBlockBuffer struct {
	buf bytes.Buffer
}

// NewUpdateBlockBuffer creates a new empty UpdateBlockBuffer.
func NewUpdateBlockBuffer() *UpdateBlockBuffer {
	return &UpdateBlockBuffer{}
}

// Write writes a value in little-endian byte order.
func (b *UpdateBlockBuffer) Write(v any) error {
	return binary.Write(&b.buf, binary.LittleEndian, v)
}

// WriteOne writes a single byte.
func (b *UpdateBlockBuffer) WriteOne(v int) error {
	return b.buf.WriteByte(byte(v))
}

// WriteUint32 writes a uint32.
func (b *UpdateBlockBuffer) WriteUint32(v uint32) error {
	return binary.Write(&b.buf, binary.LittleEndian, v)
}

// WriteBytes appends raw bytes.
func (b *UpdateBlockBuffer) WriteBytes(data []byte) {
	b.buf.Write(data)
}

// Bytes returns the accumulated bytes.
func (b *UpdateBlockBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

// Len returns the number of accumulated bytes.
func (b *UpdateBlockBuffer) Len() int {
	return b.buf.Len()
}
