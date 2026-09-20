package adt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	MagicMVER = 0x5245564D // "MVER"
	MagicMPHD = 0x4448504D // "MPHD"
	MagicMAIN = 0x4E49414D // "MAIN"
	MagicMWMO = 0x4F4D574D // "MWMO"
	MagicMODF = 0x46444F4D // "MODF"
)

var ErrNotWDT = errors.New("not a valid WDT file")

type WDT struct {
	Flags uint32
	Tiles [64][64]bool
	MWMO  string
}

// OpenWDT reads and parses a .wdt file from disk.
func OpenWDT(filePath string) (*WDT, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadWDT(f)
}

// ReadWDT parses a WDT file from an io.Reader.
func ReadWDT(r io.Reader) (*WDT, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read wdt: %w", err)
	}

	w := &WDT{}
	pos := 0

	for pos+8 <= len(data) {
		fourCC := binary.LittleEndian.Uint32(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		chunkDataEnd := pos + 8 + size
		if chunkDataEnd > len(data) {
			break
		}

		chunkData := data[pos+8 : chunkDataEnd]

		switch fourCC {
		case MagicMPHD:
			if len(chunkData) >= 4 {
				w.Flags = binary.LittleEndian.Uint32(chunkData[0:4])
			}

		case MagicMAIN:
			// 64x64 entries of 8 bytes (flags uint32, asyncId uint32)
			if len(chunkData) >= 64*64*8 {
				for y := 0; y < 64; y++ {
					for x := 0; x < 64; x++ {
						off := (y*64 + x) * 8
						flags := binary.LittleEndian.Uint32(chunkData[off : off+4])
						if (flags & 1) != 0 {
							w.Tiles[x][y] = true
						}
					}
				}
			}

		case MagicMWMO:
			w.MWMO = string(chunkData)
		}

		pos = chunkDataEnd
	}

	return w, nil
}

// HasTile returns true if the ADT tile exists at grid coordinate (x, y).
func (w *WDT) HasTile(x, y int) bool {
	if x < 0 || x >= 64 || y < 0 || y >= 64 {
		return false
	}
	return w.Tiles[x][y]
}

// TileCoords returns a list of all existing (x, y) tile coordinates.
func (w *WDT) TileCoords() [][2]int {
	var res [][2]int
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			if w.Tiles[x][y] {
				res = append(res, [2]int{x, y})
			}
		}
	}
	return res
}
