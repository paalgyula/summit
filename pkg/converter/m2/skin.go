package m2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const MagicSKIN = 0x4E494B53 // "SKIN"

var ErrNotSkin = errors.New("not a valid .skin file")

type Submesh struct {
	ID            uint16
	Level         uint16
	VertexStart   uint16
	VertexCount   uint16
	TriangleStart uint16
	TriangleCount uint16
}

// TextureUnit (M2Batch) binds a submesh to a material and texture layer.
type TextureUnit struct {
	Flags              uint8
	PriorityPlane      int8
	ShaderID           uint16
	SubmeshIndex       uint16
	GeosetIndex        uint16
	ColorIndex         uint16
	MaterialIndex      uint16
	MaterialLayer      uint16
	TextureCount       uint16
	TextureComboIndex  uint16 // into Model.TextureLookup
	TexCoordComboIndex uint16
	WeightComboIndex   uint16
	TransformCombo     uint16
}

type Skin struct {
	Indices      []uint16
	Triangles    []uint16
	Submeshes    []Submesh
	TextureUnits []TextureUnit
}

// OpenSkin reads and parses a .skin file from disk.
func OpenSkin(filePath string) (*Skin, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadSkin(f)
}

// ReadSkin parses a .skin file from an io.Reader.
func ReadSkin(r io.Reader) (*Skin, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read skin data: %w", err)
	}

	if len(data) < 48 {
		return nil, ErrNotSkin
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != MagicSKIN {
		return nil, fmt.Errorf("%w: magic 0x%08X", ErrNotSkin, magic)
	}

	// Layout:
	// 0: magic (4)
	// 4: indices (8)
	// 12: triangles (8)
	// 20: properties (8)
	// 28: submeshes (8)
	// 36: textureUnits (8)
	// 44: bones (4)

	indicesCount := binary.LittleEndian.Uint32(data[4:8])
	indicesOffset := binary.LittleEndian.Uint32(data[8:12])

	trianglesCount := binary.LittleEndian.Uint32(data[12:16])
	trianglesOffset := binary.LittleEndian.Uint32(data[16:20])

	submeshCount := binary.LittleEndian.Uint32(data[28:32])
	submeshOffset := binary.LittleEndian.Uint32(data[32:36])

	s := &Skin{}

	if int(indicesOffset)+int(indicesCount)*2 <= len(data) {
		s.Indices = make([]uint16, indicesCount)
		for i := uint32(0); i < indicesCount; i++ {
			off := indicesOffset + i*2
			s.Indices[i] = binary.LittleEndian.Uint16(data[off : off+2])
		}
	}

	if int(trianglesOffset)+int(trianglesCount)*2 <= len(data) {
		s.Triangles = make([]uint16, trianglesCount)
		for i := uint32(0); i < trianglesCount; i++ {
			off := trianglesOffset + i*2
			s.Triangles[i] = binary.LittleEndian.Uint16(data[off : off+2])
		}
	}

	submeshStride := 48
	if int(submeshOffset)+int(submeshCount)*submeshStride <= len(data) {
		s.Submeshes = make([]Submesh, submeshCount)
		for i := uint32(0); i < submeshCount; i++ {
			off := int(submeshOffset) + int(i)*submeshStride
			s.Submeshes[i] = Submesh{
				ID:            binary.LittleEndian.Uint16(data[off : off+2]),
				Level:         binary.LittleEndian.Uint16(data[off+2 : off+4]),
				VertexStart:   binary.LittleEndian.Uint16(data[off+4 : off+6]),
				VertexCount:   binary.LittleEndian.Uint16(data[off+6 : off+8]),
				TriangleStart: binary.LittleEndian.Uint16(data[off+8 : off+10]),
				TriangleCount: binary.LittleEndian.Uint16(data[off+10 : off+12]),
			}
		}
	}

	tuCount := binary.LittleEndian.Uint32(data[36:40])
	tuOffset := binary.LittleEndian.Uint32(data[40:44])
	if int(tuOffset)+int(tuCount)*24 <= len(data) {
		s.TextureUnits = make([]TextureUnit, tuCount)
		for i := uint32(0); i < tuCount; i++ {
			off := int(tuOffset) + int(i)*24
			s.TextureUnits[i] = TextureUnit{
				Flags:              data[off],
				PriorityPlane:      int8(data[off+1]),
				ShaderID:           binary.LittleEndian.Uint16(data[off+2:]),
				SubmeshIndex:       binary.LittleEndian.Uint16(data[off+4:]),
				GeosetIndex:        binary.LittleEndian.Uint16(data[off+6:]),
				ColorIndex:         binary.LittleEndian.Uint16(data[off+8:]),
				MaterialIndex:      binary.LittleEndian.Uint16(data[off+10:]),
				MaterialLayer:      binary.LittleEndian.Uint16(data[off+12:]),
				TextureCount:       binary.LittleEndian.Uint16(data[off+14:]),
				TextureComboIndex:  binary.LittleEndian.Uint16(data[off+16:]),
				TexCoordComboIndex: binary.LittleEndian.Uint16(data[off+18:]),
				WeightComboIndex:   binary.LittleEndian.Uint16(data[off+20:]),
				TransformCombo:     binary.LittleEndian.Uint16(data[off+22:]),
			}
		}
	}

	return s, nil
}
