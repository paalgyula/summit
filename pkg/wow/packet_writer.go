package wow

import (
	"bytes"
	"encoding/binary"
)

type PacketWriter struct {
	buf *bytes.Buffer
}

func NewPacketWriter() *PacketWriter {
	var buf bytes.Buffer

	return &PacketWriter{
		buf: &buf,
	}
}

func (w *PacketWriter) WriteBytes(p []byte) (int, error) {
	return w.buf.Write(p)
}

// WriteReverseBytes takes as input a byte array and returns a reversed version of it.
func (w *PacketWriter) WriteReverseBytes(data []byte) (int, error) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return w.buf.Write(data)
}

// Write writes data into the packet. You can specify the byte
// order, but default its LittleEndian.
func (w *PacketWriter) Write(v any, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	return binary.Write(w.buf, bo, v)
}

func (w *PacketWriter) WriteB(v any) error {
	return binary.Write(w.buf, binary.BigEndian, v)
}

// Deprecated: use the simple Write method instead
func (w *PacketWriter) WriteByte(b byte) error {
	return w.buf.WriteByte(b)
}

func (w *PacketWriter) WriteOne(b int) error {
	return w.buf.WriteByte(uint8(b))
}

// Write writes the string into the packet terminated by a null character.
// You can specify the byte order, but default its BigEndian.
func (w *PacketWriter) WriteString(v string, byteOrder ...binary.ByteOrder) {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	binary.Write(w.buf, bo, []byte(v))
	w.buf.WriteRune(0x00)
}

func (w *PacketWriter) WriteStringFixed(v string, size int) {
	if size > len(v) {
		size = len(v)
	}

	binary.Write(w.buf, binary.LittleEndian, []byte(v)[:size])
}

func (w *PacketWriter) Bytes() []byte {
	return w.buf.Bytes()
}

func (w *PacketWriter) Len() int {
	return w.buf.Len()
}
