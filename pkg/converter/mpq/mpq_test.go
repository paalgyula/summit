package mpq

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"testing"
)

func TestStormHashes(t *testing.T) {
	hashTableKey := HashString("(hash table)", HashTypeKey)
	if hashTableKey != KeyHashTable {
		t.Fatalf("expected KeyHashTable 0x%08X, got 0x%08X", KeyHashTable, hashTableKey)
	}

	blockTableKey := HashString("(block table)", HashTypeKey)
	if blockTableKey != KeyBlockTable {
		t.Fatalf("expected KeyBlockTable 0x%08X, got 0x%08X", KeyBlockTable, blockTableKey)
	}
}

func TestMPQArchiveRoundtrip(t *testing.T) {
	// Build a mini in-memory MPQ archive
	testFileName := "test\\sample.txt"
	testContent := []byte("Hello, World of Warcraft Web Emulation with Summit Go!")

	// Compress content with zlib (prepend 0x02 for Zlib compression mask)
	var zlibBuf bytes.Buffer
	zw := zlib.NewWriter(&zlibBuf)
	_, err := zw.Write(testContent)
	if err != nil {
		t.Fatalf("zlib write failed: %v", err)
	}
	zw.Close()

	compressedData := append([]byte{CompressionZlib}, zlibBuf.Bytes()...)

	// Calculate tables
	filePos := uint32(32) // right after header
	fileSizeCompressed := uint32(len(compressedData))
	fileSizeUncompressed := uint32(len(testContent))

	hashTableSize := uint32(4)
	blockTableSize := uint32(1)

	hashPos := filePos + fileSizeCompressed
	blockPos := hashPos + hashTableSize*16

	h := Header{
		Magic:           HeaderMagic,
		HeaderSize:      32,
		ArchiveSize:     blockPos + blockTableSize*16,
		FormatVersion:   0,
		SectorSizeShift: 3, // 4096
		HashTablePos:    hashPos,
		BlockTablePos:   blockPos,
		HashTableSize:   hashTableSize,
		BlockTableSize:  blockTableSize,
	}

	// Prepare buffer
	archiveBuf := make([]byte, h.ArchiveSize)

	// Write header
	binary.LittleEndian.PutUint32(archiveBuf[0:4], h.Magic)
	binary.LittleEndian.PutUint32(archiveBuf[4:8], h.HeaderSize)
	binary.LittleEndian.PutUint32(archiveBuf[8:12], h.ArchiveSize)
	binary.LittleEndian.PutUint16(archiveBuf[12:14], h.FormatVersion)
	binary.LittleEndian.PutUint16(archiveBuf[14:16], h.SectorSizeShift)
	binary.LittleEndian.PutUint32(archiveBuf[16:20], h.HashTablePos)
	binary.LittleEndian.PutUint32(archiveBuf[20:24], h.BlockTablePos)
	binary.LittleEndian.PutUint32(archiveBuf[24:28], h.HashTableSize)
	binary.LittleEndian.PutUint32(archiveBuf[28:32], h.BlockTableSize)

	// Write file payload
	copy(archiveBuf[filePos:], compressedData)

	// Prepare hash table
	hashBuf := make([]byte, hashTableSize*16)
	for i := uint32(0); i < hashTableSize; i++ {
		binary.LittleEndian.PutUint32(hashBuf[i*16+12:], HashEntryEmpty)
	}

	// Insert testFileName
	hashIndex := HashString(testFileName, HashTypeTableOffset) & (hashTableSize - 1)
	binary.LittleEndian.PutUint32(hashBuf[hashIndex*16:hashIndex*16+4], HashString(testFileName, HashTypeNameA))
	binary.LittleEndian.PutUint32(hashBuf[hashIndex*16+4:hashIndex*16+8], HashString(testFileName, HashTypeNameB))
	binary.LittleEndian.PutUint16(hashBuf[hashIndex*16+8:hashIndex*16+10], 0)
	binary.LittleEndian.PutUint16(hashBuf[hashIndex*16+10:hashIndex*16+12], 0)
	binary.LittleEndian.PutUint32(hashBuf[hashIndex*16+12:hashIndex*16+16], 0) // block index 0

	// Encrypt hash table with KeyHashTable
	EncryptBytes(hashBuf, KeyHashTable)
	copy(archiveBuf[hashPos:], hashBuf)

	// Prepare block table
	blockBuf := make([]byte, blockTableSize*16)
	binary.LittleEndian.PutUint32(blockBuf[0:4], filePos)
	binary.LittleEndian.PutUint32(blockBuf[4:8], fileSizeCompressed)
	binary.LittleEndian.PutUint32(blockBuf[8:12], fileSizeUncompressed)
	binary.LittleEndian.PutUint32(blockBuf[12:16], FileExists|FileCompress|FileSingleUnit)

	EncryptBytes(blockBuf, KeyBlockTable)
	copy(archiveBuf[blockPos:], blockBuf)

	// Now open using MPQ reader
	r := bytes.NewReader(archiveBuf)
	arc, err := OpenReader(r)
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}

	if !arc.HasFile(testFileName) {
		t.Fatalf("file %s was not found in archive", testFileName)
	}

	data, err := arc.ReadFile(testFileName)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(testContent) {
		t.Fatalf("expected content %q, got %q", string(testContent), string(data))
	}
}
