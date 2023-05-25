package wow

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
)

type Reader struct {
	reader io.Reader
}

func NewPacketReader(bb []byte) *Reader {
	return &Reader{
		reader: bytes.NewReader(bb),
	}
}

func NewConnectionReader(r io.Reader) *Reader {
	return &Reader{
		reader: r,
	}
}

func (r *Reader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
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

func (r *Reader) ReadString(dest *string) {
	var bb []byte

	var b byte
	binary.Read(r.reader, binary.LittleEndian, &b)

	for b != '\x00' {
		bb = append(bb, b)
		binary.Read(r.reader, binary.LittleEndian, &b)
	}

	*dest = string(bb)
}

func (r *Reader) ReadBytes(n int) ([]byte, error) {
	buf := make([]byte, n)

	n2, err := r.Read(buf)
	if err != nil {
		return buf, fmt.Errorf("wow.ReadBytes: %w", err)
	}

	if n2 != n {
		log.Warn().Err(err).Msgf("readed %d instead of required: %d", n2, n)
		fmt.Printf("%s", hex.Dump(buf[:n2]))
	}

	return buf, nil
}

func (r *Reader) ReadReverseBytes(n int) []byte {
	buf := make([]byte, n)

	err := r.ReadB(buf)
	if err != nil {
		log.Fatal().Err(err)
	}

	return ReverseBytes(buf)
}

func ReverseBytes(data []byte) []byte {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return data
}
