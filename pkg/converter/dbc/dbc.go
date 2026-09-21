// Package dbc reads the client's WDBC database files: fixed-size records of
// 4-byte columns followed by a string block the string columns index into.
package dbc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// ErrNotDBC is returned for data that is not a WDBC file.
var ErrNotDBC = errors.New("not a WDBC file")

const headerSize = 20

// File is a parsed DBC: every column is read on demand by row and index.
type File struct {
	Records    int
	Fields     int
	RecordSize int
	data       []byte
	strings    []byte
}

// Read parses a DBC file.
func Read(r io.Reader) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read dbc: %w", err)
	}
	if len(data) < headerSize || string(data[:4]) != "WDBC" {
		return nil, ErrNotDBC
	}
	f := &File{
		Records:    int(binary.LittleEndian.Uint32(data[4:])),
		Fields:     int(binary.LittleEndian.Uint32(data[8:])),
		RecordSize: int(binary.LittleEndian.Uint32(data[12:])),
	}
	stringSize := int(binary.LittleEndian.Uint32(data[16:]))
	end := headerSize + f.Records*f.RecordSize
	if f.RecordSize < f.Fields*4 || end+stringSize > len(data) {
		return nil, fmt.Errorf("%w: truncated", ErrNotDBC)
	}
	f.data = data[headerSize:end]
	f.strings = data[end : end+stringSize]
	return f, nil
}

// Uint32 returns column col of record row (0 when out of range).
func (f *File) Uint32(row, col int) uint32 {
	if row < 0 || row >= f.Records || col < 0 || col >= f.Fields {
		return 0
	}
	return binary.LittleEndian.Uint32(f.data[row*f.RecordSize+col*4:])
}

// Int32 returns a signed column.
func (f *File) Int32(row, col int) int32 {
	return int32(f.Uint32(row, col))
}

// Float32 returns a float column.
func (f *File) Float32(row, col int) float32 {
	return math.Float32frombits(f.Uint32(row, col))
}

// String returns a string column: the NUL-terminated string at the offset the
// column holds into the string block ("" when missing).
func (f *File) String(row, col int) string {
	ofs := int(f.Uint32(row, col))
	if ofs <= 0 || ofs >= len(f.strings) {
		return ""
	}
	end := ofs
	for end < len(f.strings) && f.strings[end] != 0 {
		end++
	}
	return string(f.strings[ofs:end])
}
