package adt

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWDTRoundtrip(t *testing.T) {
	// Build synthetic WDT
	var buf bytes.Buffer

	// MPHD chunk
	buf.Write([]byte("DHPMPHD\x00")[:0]) // clear
	binary.Write(&buf, binary.LittleEndian, uint32(MagicMPHD))
	binary.Write(&buf, binary.LittleEndian, uint32(32))
	mphdData := make([]byte, 32)
	buf.Write(mphdData)

	// MAIN chunk: 64x64 entries
	binary.Write(&buf, binary.LittleEndian, uint32(MagicMAIN))
	binary.Write(&buf, binary.LittleEndian, uint32(64*64*8))
	mainData := make([]byte, 64*64*8)

	// Mark tile (30, 48) as existing
	tileIdx := (48*64 + 30) * 8
	binary.LittleEndian.PutUint32(mainData[tileIdx:tileIdx+4], 1) // Flag 1 = exists
	buf.Write(mainData)

	wdt, err := ReadWDT(&buf)
	if err != nil {
		t.Fatalf("ReadWDT failed: %v", err)
	}

	if !wdt.HasTile(30, 48) {
		t.Fatalf("expected tile (30, 48) to exist")
	}

	if wdt.HasTile(0, 0) {
		t.Fatalf("expected tile (0, 0) not to exist")
	}

	tiles := wdt.TileCoords()
	if len(tiles) != 1 || tiles[0][0] != 30 || tiles[0][1] != 48 {
		t.Fatalf("expected 1 tile at (30, 48), got %+v", tiles)
	}
}

func TestADTRoundtrip(t *testing.T) {
	var buf bytes.Buffer

	// MTEX chunk
	texList := "Tileset\\Elwynn\\Elwynn_Grass.blp\x00"
	binary.Write(&buf, binary.LittleEndian, uint32(MagicMTEX))
	binary.Write(&buf, binary.LittleEndian, uint32(len(texList)))
	buf.WriteString(texList)

	// MCNK chunk at index (0, 0)
	var mcnkBuf bytes.Buffer
	mcnkHeader := make([]byte, 128)
	binary.LittleEndian.PutUint32(mcnkHeader[4:8], 0)  // indexX = 0
	binary.LittleEndian.PutUint32(mcnkHeader[8:12], 0) // indexY = 0
	// Position X, Y, Z at 0x68 (SMChunk layout)
	binary.LittleEndian.PutUint32(mcnkHeader[104:108], binary.LittleEndian.Uint32([]byte{0, 0, 0x80, 0x43})) // 256.0
	binary.LittleEndian.PutUint32(mcnkHeader[108:112], binary.LittleEndian.Uint32([]byte{0, 0, 0x80, 0x43})) // 256.0
	binary.LittleEndian.PutUint32(mcnkHeader[112:116], binary.LittleEndian.Uint32([]byte{0, 0, 0x20, 0x41})) // 10.0 (ground base)

	mcnkBuf.Write(mcnkHeader)

	// MCVT sub-chunk inside MCNK
	binary.Write(&mcnkBuf, binary.LittleEndian, uint32(MagicMCVT))
	binary.Write(&mcnkBuf, binary.LittleEndian, uint32(145*4))
	for i := 0; i < 145; i++ {
		h := float32(5.0) // +5.0 height
		if i == 9 {
			h = 7.0 // first inner vertex (row 0) in the interleaved 9/8 layout
		}
		binary.Write(&mcnkBuf, binary.LittleEndian, h)
	}

	// ofsMCVT is relative to the MCNK magic: 8 (magic+size) + 128 (header)
	binary.LittleEndian.PutUint32(mcnkBuf.Bytes()[20:24], 136)

	// Write MCNK to main buf
	binary.Write(&buf, binary.LittleEndian, uint32(MagicMCNK))
	binary.Write(&buf, binary.LittleEndian, uint32(mcnkBuf.Len()))
	buf.Write(mcnkBuf.Bytes())

	adt, err := ReadADT(&buf)
	if err != nil {
		t.Fatalf("ReadADT failed: %v", err)
	}

	if len(adt.Textures) != 1 || adt.Textures[0] != "Tileset\\Elwynn\\Elwynn_Grass.blp" {
		t.Fatalf("unexpected textures: %+v", adt.Textures)
	}

	chunk := adt.Chunks[0][0]
	if chunk == nil {
		t.Fatalf("expected chunk (0, 0) to be parsed")
	}

	if chunk.Header.Pos != [3]float32{256, 256, 10} {
		t.Fatalf("unexpected chunk position: %v", chunk.Header.Pos)
	}

	if chunk.Heights[0] != 5.0 {
		t.Fatalf("expected height 5.0, got %f", chunk.Heights[0])
	}

	// Unit (0,0) maps to inner vertex index 9, not 81
	h := adt.GetHeight(2.0, 2.0)
	if h != 17.0 { // 10.0 base + 7.0 height
		t.Fatalf("expected height 17.0, got %f", h)
	}

	// Test GLB export
	var glbBuf bytes.Buffer
	if err := adt.ExportGLB(&glbBuf); err != nil {
		t.Fatalf("ExportGLB failed: %v", err)
	}

	if glbBuf.Len() < 100 {
		t.Fatalf("GLB buffer too small: %d", glbBuf.Len())
	}
}
