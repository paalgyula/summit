package wow

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
)

type countingReader struct {
	reader    io.Reader
	BytesRead int
}

func (cr *countingReader) Read(b []byte) (int, error) {
	readed, err := cr.reader.Read(b)
	cr.BytesRead += readed

	return readed, err
}

type Reader struct {
	reader *countingReader
}

func NewPacketReader(bb []byte) *Reader {
	return &Reader{
		reader: &countingReader{
			reader:    bytes.NewReader(bb),
			BytesRead: 0,
		},
	}
}

func NewConnectionReader(r io.Reader) *Reader {
	return &Reader{
		reader: &countingReader{
			reader:    r,
			BytesRead: 0,
		},
	}
}

// ReadedCount returns the number of bytes readed from this reader.
func (r *Reader) ReadedCount() int {
	return r.reader.BytesRead
}

// ResetCounter resets the readed bytes count.
func (r *Reader) ResetCounter() {
	r.reader.BytesRead = 0
}

func (r *Reader) ReadBytes(p []byte) (int, error) {
	return r.reader.Read(p)
}

// Reads the object from the buffer. The byte order can be specified,
// but the default is LittleEndian
func (r *Reader) Read(p any, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	return binary.Read(r.reader, bo, p)
}

func (r *Reader) ReadL(v any) error {
	return binary.Read(r.reader, binary.LittleEndian, v)
}

func (r *Reader) ReadB(v any) error {
	return binary.Read(r.reader, binary.BigEndian, v)
}

// ReadStringFixed reads fixed length string
func (r *Reader) ReadStringFixed(dest *string, length int, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	bb := make([]byte, length)

	if err := binary.Read(r.reader, bo, &bb); err != nil {
		return fmt.Errorf("wow.ReadStringFixed: %w", err)
	}

	*dest = string(bb)

	return nil
}

func (r *Reader) ReadString(dest *string, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	var bb []byte

	var b byte
	binary.Read(r.reader, bo, &b)

	for b != '\x00' {
		bb = append(bb, b)
		if err := binary.Read(r.reader, binary.LittleEndian, &b); err != nil {
			return fmt.Errorf("wow.ReadString: %w", err)
		}
	}

	*dest = string(bb)

	return nil
}

// ReadNBytes reads first N bytes from the reader and returns it.
// When can't read from enough bytes from the stream, it will throw an error
func (r *Reader) ReadNBytes(n int) ([]byte, error) {
	buf := make([]byte, n)

	n2, err := r.ReadBytes(buf)
	if err != nil {
		return buf, fmt.Errorf("wow.ReadBytes: %w", err)
	}

	if n2 != n {
		log.Warn().Err(err).Msgf("readed %d instead of required: %d", n2, n)
		fmt.Printf("%s", hex.Dump(buf[:n2]))

		return buf, errors.New("cant read that much bytes")
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

func (r *Reader) ReadAll() ([]byte, error) {
	return io.ReadAll(r.reader)
}

func (r *Reader) DumpRemaining() ([]byte, error) {
	r.ResetCounter()
	bb, err := r.ReadAll()
	fmt.Println(r.ReadedCount())

	fmt.Printf("%s", hex.Dump(bb))

	return bb, err
}
