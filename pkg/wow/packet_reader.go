package wow

import (
	"bytes"
	"encoding/binary"
	"io"
)

type Reader struct {
	reader io.Reader
}

func NewPacketReader(bb []byte) *Reader {
	return &Reader{
		reader: bytes.NewReader(bb),
	}
}

func (r *Reader) ReadL(v any) error {
	return binary.Read(r.reader, binary.LittleEndian, v)
}

func (r *Reader) ReadB(v any) error {
	return binary.Read(r.reader, binary.BigEndian, v)
}

// ReadStringFixed reads fixed length string
func (r *Reader) ReadStringFixed(len int) string {
	bb := make([]byte, len)

	binary.Read(r.reader, binary.LittleEndian, &bb)

	return string(bb)
}

func (r *Reader) ReadString() string {
	var bb []byte

	var b byte
	binary.Read(r.reader, binary.LittleEndian, &b)

	for b != '\x00' {
		bb = append(bb, b)
		binary.Read(r.reader, binary.LittleEndian, &b)
	}

	return string(bb)
}

func (r *Reader) ReadReverseBytes(n int) []byte {
	buf := make([]byte, n)
	r.ReadB(buf)

	return reverse(buf)
}

func reverse(data []byte) []byte {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return data
}
