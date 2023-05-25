package dbc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type FieldType byte

const (
	FieldNA     FieldType = 'x' // Not used or unknown, 4 byte size
	FieldNAByte FieldType = 'X' // Not used or unknown, byte
	FieldString FieldType = 's' // char*
	FieldFloat  FieldType = 'f' // float
	FieldInt    FieldType = 'i' // uint32
	FieldByte   FieldType = 'b' // uint8
	FieldSort   FieldType = 'd' // Sorted by this field, field is not included
	FieldIndex  FieldType = 'n' // The same,but parsed to data
	FieldLogic  FieldType = 'l' // Logical (boolean)
)

func (f FieldType) Size() int {
	switch f {
	case FieldNA, FieldFloat, FieldInt, FieldIndex:
		return 4
	case FieldNAByte, FieldByte, FieldLogic:
		return 1
	}

	return 1 // Field string?
}

type RecordFormat []FieldType

func (r RecordFormat) Length() (i int) {
	for _, v := range r {
		i += v.Size()
	}

	return
}

type DataHeader struct {
	Magic           [4]byte // always 'WDBC'
	RecordCount     uint32  // records per file
	FieldCount      uint32  // fields per record
	RecordSize      uint32  // sum (sizeof (field_type_i)) | 0 <= i < field_count. field_type_i is NOT defined in the files.
	StringBlockSize uint32  // Block size of the string block at the end of file
}

type Reader[C any] struct {
	file   io.ReadSeekCloser
	header DataHeader

	current   []byte
	recordPos int

	recordFormat RecordFormat
}

// NewReader creates a DBC file reader with a format
func NewReader[C comparable](f io.ReadSeekCloser, format string) *Reader[C] {
	r := &Reader[C]{file: f}
	r.readHeader()

	r.recordFormat = RecordFormat(format)

	fmt.Printf("Record len: %d\n", r.recordFormat.Length())

	return r
}

// Header returns the DBC file header
func (r *Reader[C]) Header() DataHeader {
	return r.header
}

func (r *Reader[C]) readHeader() {
	_, _ = r.file.Seek(0, 0)

	binary.Read(r.file, binary.LittleEndian, &r.header)
	fmt.Printf("Header: %+v\n", r.header)

	r.current = make([]byte, r.header.RecordSize)
	r.recordPos = 0
}

func (r *Reader[C]) checkStringSize() {
	pos, _ := r.file.Seek(0, 1)
	endPos, _ := r.file.Seek(0, 2)
	distance := endPos - pos

	fmt.Printf("  Pos: %d End: %d, Distance: %d\n", pos, endPos, distance)
	fmt.Printf("  RecordCount: %d\n", r.header.RecordCount)
}

// HasNext is the iterator which is iterating through the records,
// and returns false if no more record found.
func (r *Reader[C]) HasNext() bool {
	return r.recordPos < int(r.header.RecordCount)
}

// Reduce and skips bytes based on format
func (r *Reader[C]) reduceWithFormat() {
	rreader := bytes.NewReader(r.current[:])
	reduced := &bytes.Buffer{}

	for _, v := range r.recordFormat {
		switch v {
		case FieldNA, FieldNAByte:
			rreader.Seek(int64(v.Size()), 1)
		default:
			buf := make([]byte, v.Size())
			io.CopyBuffer(reduced, rreader, buf)
		}
	}
}

func (r *Reader[C]) Next() *C {
	_, err := r.file.Read(r.current)

	if err != nil {
		panic(err)
	}

	r.reduceWithFormat()
	r.recordPos++

	var rec C
	dr := bytes.NewReader(r.current)
	binary.Read(dr, binary.LittleEndian, &rec)

	return &rec
}

func (r *Reader[C]) Current() []byte {
	return r.current
}

func (r *Reader[C]) Strings() []byte {
	dest := r.header.RecordCount*r.header.RecordSize + 20
	_, err := r.file.Seek(int64(dest), 0)
	if err != nil {
		panic(err)
	}
	bb := make([]byte, r.header.StringBlockSize)

	_, err = r.file.Read(bb)
	if err != nil {
		panic(err)
	}

	return bb
}

func (r *Reader[C]) ReadRecord() error {
	for i := 0; i < int(r.header.RecordCount); i++ {
		rec := make([]byte, r.header.RecordSize)
		_, err := r.file.Read(rec)

		if err != nil {
			return err
		}

		// fmt.Printf("readed: %d/%d\n", n, header.RecordSize)
		// fmt.Printf("%s", hex.Dump(rec))
	}

	return nil
}
