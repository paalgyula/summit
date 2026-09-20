package mpq

import (
	"bytes"
	"compress/bzip2"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	HeaderMagic         = 0x1A51504D // "MPQ\x1a"
	UserDataMagic       = 0x1B51504D // "MPQ\x1b"
	HashEntryEmpty      = 0xFFFFFFFF
	HashEntryDeleted    = 0xFFFFFFFE

	HashTypeTableOffset = 0
	HashTypeNameA       = 1
	HashTypeNameB       = 2
	HashTypeKey         = 3

	KeyHashTable = 0xC3AF3770 // HashString("(hash table)", HashTypeKey)
	KeyBlockTable = 0xEC83B3A3 // HashString("(block table)", HashTypeKey)

	FileImplode      = 0x00000100
	FileCompress     = 0x00000200
	FileEncrypted    = 0x00010000
	FileKeyAdjusted  = 0x00020000
	FilePatchFile    = 0x00100000
	FileSingleUnit   = 0x01000000
	FileDeleteMarker = 0x02000000
	FileExists       = 0x80000000

	CompressionHuffman = 0x01
	CompressionZlib    = 0x02
	CompressionPKWARE  = 0x08
	CompressionBzip2   = 0x10
	CompressionSparse  = 0x20
	CompressionADPCM_M = 0x40
	CompressionADPCM_S = 0x80
)

var (
	ErrNotMPQ          = errors.New("not a valid MPQ archive")
	ErrFileNotFound    = errors.New("file not found in MPQ archive")
	ErrUnsupportedComp = errors.New("unsupported compression type in MPQ")
	cryptTable         [0x500]uint32
)

func init() {
	seed := uint32(0x00100001)
	for i := 0; i < 0x100; i++ {
		idx := i
		for j := 0; j < 5; j++ {
			seed = (seed*125 + 3) % 0x2AAAAB
			temp1 := (seed & 0xFFFF) << 0x10
			seed = (seed*125 + 3) % 0x2AAAAB
			temp2 := seed & 0xFFFF
			cryptTable[idx] = temp1 | temp2
			idx += 0x100
		}
	}
}

// HashString calculates Blizzard's Storm hash of a string.
func HashString(s string, hashType uint32) uint32 {
	seed1 := uint32(0x7FED7FED)
	seed2 := uint32(0xEEEEEEEE)
	clean := strings.ReplaceAll(strings.ToUpper(s), "/", "\\")

	for i := 0; i < len(clean); i++ {
		ch := uint32(clean[i])
		seed1 = cryptTable[(hashType<<8)+ch] ^ (seed1 + seed2)
		seed2 = ch + seed1 + seed2 + (seed2 << 5) + 3
	}
	return seed1
}

// DecryptBlock decrypts a slice of uint32 in-place using Storm cryptography.
func DecryptBlock(data []uint32, key uint32) {
	seed := uint32(0xEEEEEEEE)
	for i := 0; i < len(data); i++ {
		seed += cryptTable[0x400+(key&0xFF)]
		ch := data[i] ^ (key + seed)
		key = ((^key << 0x15) + 0x11111111) | (key >> 0x0B)
		seed = ch + seed + (seed << 5) + 3
		data[i] = ch
	}
}

// EncryptBlock encrypts a slice of uint32 in-place using Storm cryptography.
func EncryptBlock(data []uint32, key uint32) {
	seed := uint32(0xEEEEEEEE)
	for i := 0; i < len(data); i++ {
		seed += cryptTable[0x400+(key&0xFF)]
		ch := data[i]
		data[i] = ch ^ (key + seed)
		key = ((^key << 0x15) + 0x11111111) | (key >> 0x0B)
		seed = ch + seed + (seed << 5) + 3
	}
}

// DecryptBytes decrypts a byte slice in-place by treating it as little-endian uint32s.
func DecryptBytes(data []byte, key uint32) {
	numWords := len(data) / 4
	words := make([]uint32, numWords)
	for i := 0; i < numWords; i++ {
		words[i] = binary.LittleEndian.Uint32(data[i*4:])
	}
	DecryptBlock(words, key)
	for i := 0; i < numWords; i++ {
		binary.LittleEndian.PutUint32(data[i*4:], words[i])
	}
}

// EncryptBytes encrypts a byte slice in-place by treating it as little-endian uint32s.
func EncryptBytes(data []byte, key uint32) {
	numWords := len(data) / 4
	words := make([]uint32, numWords)
	for i := 0; i < numWords; i++ {
		words[i] = binary.LittleEndian.Uint32(data[i*4:])
	}
	EncryptBlock(words, key)
	for i := 0; i < numWords; i++ {
		binary.LittleEndian.PutUint32(data[i*4:], words[i])
	}
}

// Header represents the MPQ archive header.
type Header struct {
	Magic           uint32
	HeaderSize      uint32
	ArchiveSize     uint32
	FormatVersion   uint16
	SectorSizeShift uint16
	HashTablePos    uint32
	BlockTablePos   uint32
	HashTableSize   uint32
	BlockTableSize  uint32
}

// HashEntry represents an entry in the hash table.
type HashEntry struct {
	NameHashA  uint32
	NameHashB  uint32
	Locale     uint16
	Platform   uint16
	BlockIndex uint32
}

// BlockEntry represents an entry in the block table.
type BlockEntry struct {
	FilePos          uint32
	CompressedSize   uint32
	UncompressedSize uint32
	Flags            uint32
}

// FileInfo provides metadata about an entry in the archive.
type FileInfo struct {
	Name             string
	CompressedSize   uint32
	UncompressedSize uint32
	Flags            uint32
	BlockIndex       uint32
}

// Archive represents an open MPQ archive.
type Archive struct {
	r          io.ReaderAt
	header     Header
	baseOffset int64
	sectorSize uint32
	hashTable  []HashEntry
	blockTable []BlockEntry
	fileNames  map[string]uint32 // name -> block index
}

// Open reads and parses an MPQ archive from a file.
func Open(filePath string) (*Archive, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("mpq open file: %w", err)
	}
	return OpenReader(f)
}

// OpenReader reads and parses an MPQ archive from any io.ReaderAt.
func OpenReader(r io.ReaderAt) (*Archive, error) {
	// Find MPQ header. Archives can have an offset (e.g. if embedded in an exe or with user data).
	var baseOffset int64 = 0
	buf := make([]byte, 512)

	for offset := int64(0); offset < 1024*1024*16; offset += 512 {
		n, err := r.ReadAt(buf[:4], offset)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n < 4 {
			break
		}
		magic := binary.LittleEndian.Uint32(buf[:4])
		if magic == HeaderMagic {
			baseOffset = offset
			break
		}
	}

	headerBuf := make([]byte, 32)
	if _, err := r.ReadAt(headerBuf, baseOffset); err != nil {
		return nil, fmt.Errorf("read mpq header: %w", err)
	}

	var h Header
	h.Magic = binary.LittleEndian.Uint32(headerBuf[0:4])
	if h.Magic != HeaderMagic {
		return nil, ErrNotMPQ
	}
	h.HeaderSize = binary.LittleEndian.Uint32(headerBuf[4:8])
	h.ArchiveSize = binary.LittleEndian.Uint32(headerBuf[8:12])
	h.FormatVersion = binary.LittleEndian.Uint16(headerBuf[12:14])
	h.SectorSizeShift = binary.LittleEndian.Uint16(headerBuf[14:16])
	h.HashTablePos = binary.LittleEndian.Uint32(headerBuf[16:20])
	h.BlockTablePos = binary.LittleEndian.Uint32(headerBuf[20:24])
	h.HashTableSize = binary.LittleEndian.Uint32(headerBuf[24:28])
	h.BlockTableSize = binary.LittleEndian.Uint32(headerBuf[28:32])

	a := &Archive{
		r:          r,
		header:     h,
		baseOffset: baseOffset,
		sectorSize: 512 << h.SectorSizeShift,
		fileNames:  make(map[string]uint32),
	}

	// Read and decrypt hash table
	hashBuf := make([]byte, h.HashTableSize*16)
	if _, err := r.ReadAt(hashBuf, baseOffset+int64(h.HashTablePos)); err != nil {
		return nil, fmt.Errorf("read hash table: %w", err)
	}
	DecryptBytes(hashBuf, KeyHashTable)

	a.hashTable = make([]HashEntry, h.HashTableSize)
	for i := uint32(0); i < h.HashTableSize; i++ {
		off := i * 16
		a.hashTable[i] = HashEntry{
			NameHashA:  binary.LittleEndian.Uint32(hashBuf[off : off+4]),
			NameHashB:  binary.LittleEndian.Uint32(hashBuf[off+4 : off+8]),
			Locale:     binary.LittleEndian.Uint16(hashBuf[off+8 : off+10]),
			Platform:   binary.LittleEndian.Uint16(hashBuf[off+10 : off+12]),
			BlockIndex: binary.LittleEndian.Uint32(hashBuf[off+12 : off+16]),
		}
	}

	// Read and decrypt block table
	blockBuf := make([]byte, h.BlockTableSize*16)
	if _, err := r.ReadAt(blockBuf, baseOffset+int64(h.BlockTablePos)); err != nil {
		return nil, fmt.Errorf("read block table: %w", err)
	}
	DecryptBytes(blockBuf, KeyBlockTable)

	a.blockTable = make([]BlockEntry, h.BlockTableSize)
	for i := uint32(0); i < h.BlockTableSize; i++ {
		off := i * 16
		a.blockTable[i] = BlockEntry{
			FilePos:          binary.LittleEndian.Uint32(blockBuf[off : off+4]),
			CompressedSize:   binary.LittleEndian.Uint32(blockBuf[off+4 : off+8]),
			UncompressedSize: binary.LittleEndian.Uint32(blockBuf[off+8 : off+12]),
			Flags:            binary.LittleEndian.Uint32(blockBuf[off+12 : off+16]),
		}
	}

	// Read (listfile) if present to populate file names
	if listData, err := a.ReadFile("(listfile)"); err == nil {
		lines := strings.Split(string(listData), "\r\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				if blockIdx, err := a.findBlockIndex(line); err == nil {
					a.fileNames[strings.ToLower(line)] = blockIdx
				}
			}
		}
	}

	return a, nil
}

func (a *Archive) findBlockIndex(name string) (uint32, error) {
	if len(a.hashTable) == 0 {
		return 0, ErrFileNotFound
	}

	hashIndex := HashString(name, HashTypeTableOffset) & (a.header.HashTableSize - 1)
	hashA := HashString(name, HashTypeNameA)
	hashB := HashString(name, HashTypeNameB)

	for i := hashIndex; ; i = (i + 1) & (a.header.HashTableSize - 1) {
		entry := a.hashTable[i]
		if entry.BlockIndex == HashEntryEmpty {
			return 0, ErrFileNotFound
		}
		if entry.NameHashA == hashA && entry.NameHashB == hashB && entry.BlockIndex < a.header.BlockTableSize {
			return entry.BlockIndex, nil
		}
		if (i+1)&(a.header.HashTableSize-1) == hashIndex {
			break
		}
	}

	return 0, ErrFileNotFound
}

// HasFile checks if a file exists in the archive.
func (a *Archive) HasFile(name string) bool {
	_, err := a.findBlockIndex(name)
	return err == nil
}

// ListFiles returns all known file names in the archive (from listfile).
func (a *Archive) ListFiles() []string {
	res := make([]string, 0, len(a.fileNames))
	for name := range a.fileNames {
		res = append(res, name)
	}
	return res
}

// ReadFile extracts and returns the contents of a file by name.
func (a *Archive) ReadFile(name string) ([]byte, error) {
	blockIdx, err := a.findBlockIndex(name)
	if err != nil {
		return nil, err
	}

	return a.ReadBlock(blockIdx, name)
}

// ReadBlock extracts and returns the contents of a file by its block index.
func (a *Archive) ReadBlock(blockIdx uint32, nameHint string) ([]byte, error) {
	if blockIdx >= uint32(len(a.blockTable)) {
		return nil, errors.New("block index out of bounds")
	}

	b := a.blockTable[blockIdx]
	if (b.Flags & FileExists) == 0 {
		return nil, errors.New("file block marked as deleted or does not exist")
	}
	if b.UncompressedSize == 0 {
		return []byte{}, nil
	}

	fileOffset := a.baseOffset + int64(b.FilePos)
	raw := make([]byte, b.CompressedSize)
	if _, err := a.r.ReadAt(raw, fileOffset); err != nil {
		return nil, fmt.Errorf("read file data: %w", err)
	}

	// Decrypt if necessary
	if (b.Flags & FileEncrypted) != 0 {
		key := HashString(filepath.Base(nameHint), HashTypeKey)
		if (b.Flags & FileKeyAdjusted) != 0 {
			key = (key + b.FilePos) ^ b.UncompressedSize
		}
		DecryptBytes(raw, key)
	}

	// Uncompressed directly
	if (b.Flags & (FileCompress | FileImplode)) == 0 {
		return raw[:b.UncompressedSize], nil
	}

	// Single-unit file
	if (b.Flags & FileSingleUnit) != 0 {
		return decompressSingleUnit(raw, b.UncompressedSize)
	}

	// Sector-based compressed file
	return a.decompressSectors(raw, b)
}

func decompressSingleUnit(data []byte, uncompressedSize uint32) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	compMask := data[0]
	switch compMask {
	case CompressionZlib:
		zr, err := zlib.NewReader(bytes.NewReader(data[1:]))
		if err != nil {
			return nil, fmt.Errorf("zlib open: %w", err)
		}
		defer zr.Close()
		buf := make([]byte, uncompressedSize)
		_, err = io.ReadFull(zr, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("zlib read: %w", err)
		}
		return buf, nil

	case CompressionBzip2:
		br := bzip2.NewReader(bytes.NewReader(data[1:]))
		buf := make([]byte, uncompressedSize)
		_, err := io.ReadFull(br, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("bzip2 read: %w", err)
		}
		return buf, nil

	default:
		// Attempt direct zlib without compression mask
		zr, err := zlib.NewReader(bytes.NewReader(data))
		if err == nil {
			defer zr.Close()
			buf := make([]byte, uncompressedSize)
			if _, rErr := io.ReadFull(zr, buf); rErr == nil || rErr == io.EOF {
				return buf, nil
			}
		}
		return nil, fmt.Errorf("%w: 0x%02x", ErrUnsupportedComp, compMask)
	}
}

func (a *Archive) decompressSectors(data []byte, b BlockEntry) ([]byte, error) {
	numSectors := (b.UncompressedSize + a.sectorSize - 1) / a.sectorSize
	hasCrc := false // standard MPQ v1 does not have CRC sector

	tableSize := (numSectors + 1) * 4
	if hasCrc {
		tableSize = (numSectors + 2) * 4
	}

	if uint32(len(data)) < tableSize {
		return nil, errors.New("corrupt sector offset table")
	}

	sectorOffsets := make([]uint32, numSectors+1)
	for i := uint32(0); i <= numSectors; i++ {
		sectorOffsets[i] = binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
	}

	out := make([]byte, 0, b.UncompressedSize)

	for i := uint32(0); i < numSectors; i++ {
		start := sectorOffsets[i]
		end := sectorOffsets[i+1]
		if end > uint32(len(data)) || start > end {
			return nil, errors.New("invalid sector offset")
		}

		sectorData := data[start:end]
		expectedSize := a.sectorSize
		if i == numSectors-1 {
			rem := b.UncompressedSize % a.sectorSize
			if rem != 0 {
				expectedSize = rem
			}
		}

		if uint32(len(sectorData)) == expectedSize {
			out = append(out, sectorData...)
			continue
		}

		decomp, err := decompressSingleUnit(sectorData, expectedSize)
		if err != nil {
			return nil, fmt.Errorf("sector %d: %w", i, err)
		}
		out = append(out, decomp...)
	}

	return out, nil
}
