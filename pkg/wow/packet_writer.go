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

func (w *PacketWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

// WriteReverse takes as input a byte array and returns a reversed version of it.
func (w *PacketWriter) WriteReverse(data []byte) (int, error) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return w.buf.Write(data)
}

func (w *PacketWriter) WriteL(v any) error {
	return binary.Write(w.buf, binary.LittleEndian, v)
}

func (w *PacketWriter) WriteB(v any) error {
	return binary.Write(w.buf, binary.BigEndian, v)
}

func (w *PacketWriter) WriteByte(b byte) error {
	return w.buf.WriteByte(b)
}

func (w *PacketWriter) WriteString(v string) {
	binary.Write(w.buf, binary.BigEndian, []byte(v))
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
