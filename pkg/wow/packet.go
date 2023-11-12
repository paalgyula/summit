//nolint:wrapcheck
package wow

import (
	"bytes"
	"encoding/binary"
)

type Packet struct {
	buf    *bytes.Buffer
	opcode OpCode
}

func NewPacket(opCode OpCode) *Packet {
	var buf bytes.Buffer

	return &Packet{
		opcode: opCode,
		buf:    &buf,
	}
}

func (w *Packet) OpCode() int {
	return int(w.opcode)
}

func (w *Packet) WriteBytes(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *Packet) WriteZeroPadded(p []byte, size int) (int, error) {
	p = PadBigIntBytes(p, size)

	return w.WriteBytes(p)
}

// WriteReverseBytes takes as input a byte array and returns a reversed version of it.
func (w *Packet) WriteReverseBytes(data []byte) (int, error) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return w.buf.Write(data)
}

// Write writes data into the packet. You can specify the byte
// order, but default its LittleEndian.
func (w *Packet) Write(v any, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	return binary.Write(w.buf, bo, v)
}

func (w *Packet) WriteB(v any) error {
	return binary.Write(w.buf, binary.BigEndian, v)
}

// Deprecated: use the simple Write method instead.
func (w *Packet) WriteByte(b byte) error {
	return w.buf.WriteByte(b)
}

func (w *Packet) WriteOne(b int) error {
	return w.buf.WriteByte(uint8(b))
}

func (w *Packet) WriteUint32(b int) error {
	return w.Write(uint32(b))
}

// Write writes the string into the packet terminated by a null character.
// You can specify the byte order, but default its BigEndian.
func (w *Packet) WriteString(v string, byteOrder ...binary.ByteOrder) {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	_ = binary.Write(w.buf, bo, []byte(v))
	w.buf.WriteRune(0x00)
}

func (w *Packet) WriteStringFixed(v string, size int) {
	if size > len(v) {
		size = len(v)
	}

	_ = binary.Write(w.buf, binary.LittleEndian, []byte(v)[:size])
}

func (w *Packet) Bytes() []byte {
	return w.buf.Bytes()
}

func (w *Packet) Len() int {
	return w.buf.Len()
}

// PadBigIntBytes takes as input an array of bytes and a size and ensures that the
// byte array is at least nBytes in length. \x00 bytes will be added to the end
// until the desired length is reached.
func PadBigIntBytes(data []byte, nBytes int) []byte {
	if len(data) > nBytes {
		return data[:nBytes]
	}

	currSize := len(data)
	for i := 0; i < nBytes-currSize; i++ {
		data = append(data, '\x00')
	}

	return data
}

type RealmPacket interface {
	UnmarshalPacket(data PacketData) error
}

type PacketData []byte

func (pd PacketData) Reader() *PacketReader {
	return NewPacketReader(pd)
}
