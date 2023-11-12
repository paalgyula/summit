//nolint:wrapcheck,errcheck
package wow

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
)

type countingReadder struct {
	reader    io.Reader
	BytesRead int
}

// Read reads data into the provided byte slice.
//
// It returns the number of bytes read and any error encountered.
func (cr *countingReadder) Read(b []byte) (int, error) {
	readed, err := cr.reader.Read(b)
	cr.BytesRead += readed

	return readed, err
}

type PacketReader struct {
	reader *countingReadder
}

func NewPacketReader(bb []byte) *PacketReader {
	return &PacketReader{
		reader: &countingReadder{
			reader:    bytes.NewReader(bb),
			BytesRead: 0,
		},
	}
}

// NewConnectionReader initializes a new PacketReader from a net.Conn (or from any reader implementation).
func NewConnectionReader(r io.Reader) *PacketReader {
	return &PacketReader{
		reader: &countingReadder{
			reader:    r,
			BytesRead: 0,
		},
	}
}

// ReadedCount returns the number of bytes readed from this reader.
func (r *PacketReader) ReadedCount() int {
	return r.reader.BytesRead
}

// ResetCounter resets the readed bytes count.
func (r *PacketReader) ResetCounter() {
	r.reader.BytesRead = 0
}

// ReadBytes reads data into p.
//
// It returns the number of bytes read and any error encountered.
func (r *PacketReader) ReadBytes(p []byte) (int, error) {
	return r.reader.Read(p)
}

// Reads the object from the buffer. The byte order can be specified,
// but the default is LittleEndian.
func (r *PacketReader) Read(p any, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	return binary.Read(r.reader, bo, p)
}

// ReadL reads binary data from the Reader using the LittleEndian encoding and stores it in the provided variable.
//
// v: the variable to store the read data.
// error: returns an error if the reading operation fails.
func (r *PacketReader) ReadL(v any) error {
	return binary.Read(r.reader, binary.LittleEndian, v)
}

// ReadB reads binary data from the Reader in big endian byte order
//
// v: a variable to store the read data.
// error: an error if the read operation fails.
func (r *PacketReader) ReadB(v any) error {
	return binary.Read(r.reader, binary.BigEndian, v)
}

// ReadStringFixed reads fixed length string.
func (r *PacketReader) ReadStringFixed(dest *string, length int, byteOrder ...binary.ByteOrder) error {
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

// ReadString reads a string from the Reader and stores it in the provided destination.
//
// The function takes an optional byteOrder parameter, which specifies the byte order used to read the string.
// The function returns an error if there is an error reading the string from the Reader.
func (r *PacketReader) ReadString(dest *string, byteOrder ...binary.ByteOrder) error {
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
// When can't read from enough bytes from the stream, it will throw an error.
func (r *PacketReader) ReadNBytes(n int) ([]byte, error) {
	buf := make([]byte, n)

	n2, err := r.ReadBytes(buf)
	if err != nil {
		return buf, fmt.Errorf("wow.ReadBytes: %w", err)
	}

	if n2 != n {
		log.Warn().Err(err).Msgf("readed %d instead of required: %d", n2, n)
		log.Printf("%s", hex.Dump(buf[:n2]))

		return buf, io.ErrUnexpectedEOF
	}

	return buf, nil
}

// ReadReverseBytes reads and returns n bytes in reverse order from the reader.
//
// It takes an integer n as a parameter, which specifies the number of bytes to read.
// It returns a byte slice containing the read bytes.
func (r *PacketReader) ReadReverseBytes(n int) []byte {
	buf := make([]byte, n)

	err := r.ReadB(buf)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	return ReverseBytes(buf)
}

// ReadAll reads all the data from the reader and returns it as a byte slice.
//
// It returns the byte slice containing the data and an error if any.
func (r *PacketReader) ReadAll() ([]byte, error) {
	return io.ReadAll(r.reader) //nolint:wrapcheck
}

// DumpRemaining is a function that dumps the remaining data from the Reader.
//
// It resets the counter of the Reader and reads all the remaining data.
// Then it prints the number of bytes read and the hexadecimal dump of the data.
// Finally, it returns the read data and any error that occurred.
func (r *PacketReader) DumpRemaining() ([]byte, error) {
	r.ResetCounter()
	bb, err := r.ReadAll()
	log.Print(r.ReadedCount())

	log.Printf("%s", hex.Dump(bb))

	return bb, err
}

// ReverseBytes reverses the order of bytes in a byte slice.
//
// data: the byte slice to be reversed.
// []byte: the reversed byte slice.
func ReverseBytes(data []byte) []byte {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return data
}
