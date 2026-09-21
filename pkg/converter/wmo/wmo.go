package wmo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

const (
	MagicMVER = 0x4D564552 // "MVER" (stored reversed on disk)
	MagicMOHD = 0x4D4F4844 // "MOHD" (stored reversed on disk)
	MagicMOTX = 0x4D4F5458 // "MOTX" (stored reversed on disk)
	MagicMOMT = 0x4D4F4D54 // "MOMT" (stored reversed on disk)
	MagicMOGN = 0x4D4F474E // "MOGN" (stored reversed on disk)
	MagicMOGI = 0x4D4F4749 // "MOGI" (stored reversed on disk)
	MagicMODS = 0x4D4F4453 // "MODS" (stored reversed on disk)
	MagicMODN = 0x4D4F444E // "MODN" (stored reversed on disk)
	MagicMODD = 0x4D4F4444 // "MODD" (stored reversed on disk)
	MagicMOGP = 0x4D4F4750 // "MOGP" (stored reversed on disk)
	MagicMOVT = 0x4D4F5654 // "MOVT" (stored reversed on disk)
	MagicMONR = 0x4D4F4E52 // "MONR" (stored reversed on disk)
	MagicMOTV = 0x4D4F5456 // "MOTV" (stored reversed on disk)
	MagicMOVI = 0x4D4F5649 // "MOVI" (stored reversed on disk)
	MagicMOBA = 0x4D4F4241 // "MOBA" (stored reversed on disk)
	MagicMOCV = 0x4D4F4356 // "MOCV" (stored reversed on disk)
)

// Material flags (MOMT.flags).
const (
	MaterialFlagUnlit    = 0x01
	MaterialFlagUnfogged = 0x02
	MaterialFlagTwoSided = 0x04 // F_UNCULLED
	MaterialFlagExtLight = 0x08 // darkened by the exterior light
	MaterialFlagSIDN     = 0x10 // night glow (emissive)
	MaterialFlagWindow   = 0x20
	MaterialFlagClampS   = 0x40
	MaterialFlagClampT   = 0x80
)

// Material blend modes (MOMT.blendMode).
const (
	BlendOpaque   = 0
	BlendAlphaKey = 1
	BlendAlpha    = 2
)

// Group flags (MOGP.flags).
const (
	GroupFlagHasVertexColors = 0x04
	GroupFlagOutdoor         = 0x08
	GroupFlagExteriorLit     = 0x40
	GroupFlagIndoor          = 0x2000
)

var ErrNotWMO = errors.New("not a valid WMO file")

type Header struct {
	NMaterials   uint32
	NGroups      uint32
	NPortals     uint32
	NLights      uint32
	NDoodadNames uint32
	NDoodadDefs  uint32
	NDoodadSets  uint32
	AmbientColor [4]uint8
	WMOID        uint32
	BoundingBox  [2][3]float32
	Flags        uint16
}

type Material struct {
	Flags     uint32
	Shader    uint32
	BlendMode uint32
	Texture1  uint32 // byte offset into MOTX
	Color1    [4]uint8
	Texture2  uint32
	Color2    [4]uint8
	// Texture / Texture2Name are the offsets resolved against MOTX ("" when empty).
	Texture      string
	Texture2Name string
}

// DoodadSet is one MODS entry: a named range of MODD entries. Set 0 is
// always shown; an MODF placement picks one more.
type DoodadSet struct {
	Name  string
	Start uint32
	Count uint32
}

type DoodadPlacement struct {
	NameOffset uint32 // byte offset into MODN (24 bits on disk)
	Flags      uint8
	Pos        [3]float32
	Rot        [4]float32 // Quaternion x, y, z, w
	Scale      float32
	LightColor [4]uint8
	// Name is NameOffset resolved against MODN.
	Name string
}

type Batch struct {
	StartIndex  uint32
	IndexCount  uint16
	VertexStart uint16
	VertexEnd   uint16
	MaterialID  uint8
}

type Group struct {
	Name        string
	Flags       uint32
	Vertices    [][3]float32
	Normals     [][3]float32
	TexCoords   [][2]float32 // first MOTV set
	TexCoords2  [][2]float32 // second MOTV set (two-layer shaders)
	Colors      [][4]uint8   // MOCV (BGRA), pre-baked lighting of indoor groups
	Indices     []uint16
	Batches     []Batch
	BoundingBox [2][3]float32
}

type RootWMO struct {
	Header      Header
	Textures    []string
	Materials   []Material
	DoodadNames []string
	DoodadSets  []DoodadSet
	Doodads     []DoodadPlacement
	Groups      []*Group

	motx, modn []byte
}

// OpenRoot reads and parses a Root WMO file from disk.
func OpenRoot(filePath string) (*RootWMO, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadRoot(f)
}

// ReadRoot parses a Root WMO from an io.Reader.
func ReadRoot(r io.Reader) (*RootWMO, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read wmo: %w", err)
	}

	wmo := &RootWMO{}
	pos := 0

	for pos+8 <= len(data) {
		fourCC := binary.LittleEndian.Uint32(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		chunkEnd := pos + 8 + size
		if chunkEnd > len(data) {
			break
		}
		chunkData := data[pos+8 : chunkEnd]

		switch fourCC {
		case MagicMOHD:
			if len(chunkData) >= 64 {
				wmo.Header.NMaterials = binary.LittleEndian.Uint32(chunkData[0:4])
				wmo.Header.NGroups = binary.LittleEndian.Uint32(chunkData[4:8])
				wmo.Header.NPortals = binary.LittleEndian.Uint32(chunkData[8:12])
				wmo.Header.NLights = binary.LittleEndian.Uint32(chunkData[12:16])
				wmo.Header.NDoodadNames = binary.LittleEndian.Uint32(chunkData[16:20])
				wmo.Header.NDoodadDefs = binary.LittleEndian.Uint32(chunkData[20:24])
				wmo.Header.NDoodadSets = binary.LittleEndian.Uint32(chunkData[24:28])
				copy(wmo.Header.AmbientColor[:], chunkData[28:32])
				wmo.Header.WMOID = binary.LittleEndian.Uint32(chunkData[32:36])
				for k := 0; k < 3; k++ {
					wmo.Header.BoundingBox[0][k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[36+k*4:]))
					wmo.Header.BoundingBox[1][k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[48+k*4:]))
				}
				wmo.Header.Flags = binary.LittleEndian.Uint16(chunkData[60:62])
			}

		case MagicMOTX:
			wmo.motx = chunkData
			wmo.Textures = parseStringList(chunkData)

		case MagicMOMT:
			count := len(chunkData) / 64
			wmo.Materials = make([]Material, count)
			for i := 0; i < count; i++ {
				off := i * 64
				wmo.Materials[i] = Material{
					Flags:     binary.LittleEndian.Uint32(chunkData[off : off+4]),
					Shader:    binary.LittleEndian.Uint32(chunkData[off+4 : off+8]),
					BlendMode: binary.LittleEndian.Uint32(chunkData[off+8 : off+12]),
					Texture1:  binary.LittleEndian.Uint32(chunkData[off+12 : off+16]),
					Texture2:  binary.LittleEndian.Uint32(chunkData[off+20 : off+24]),
				}
				copy(wmo.Materials[i].Color1[:], chunkData[off+16:off+20])
				copy(wmo.Materials[i].Color2[:], chunkData[off+24:off+28])
			}

		case MagicMOGN:
			// group names, unused

		case MagicMODS:
			count := len(chunkData) / 32
			wmo.DoodadSets = make([]DoodadSet, count)
			for i := 0; i < count; i++ {
				off := i * 32
				wmo.DoodadSets[i] = DoodadSet{
					Name:  string(bytes.TrimRight(chunkData[off:off+20], "\x00")),
					Start: binary.LittleEndian.Uint32(chunkData[off+20 : off+24]),
					Count: binary.LittleEndian.Uint32(chunkData[off+24 : off+28]),
				}
			}

		case MagicMODN:
			wmo.modn = chunkData
			wmo.DoodadNames = parseStringList(chunkData)

		case MagicMODD:
			count := len(chunkData) / 40
			wmo.Doodads = make([]DoodadPlacement, count)
			for i := 0; i < count; i++ {
				off := i * 40
				// nameIndex is 24 bits, the top byte holds the flags
				nameFlags := binary.LittleEndian.Uint32(chunkData[off : off+4])
				wmo.Doodads[i] = DoodadPlacement{
					NameOffset: nameFlags & 0xFFFFFF,
					Flags:      uint8(nameFlags >> 24),
					Pos: [3]float32{
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+4 : off+8])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+8 : off+12])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+12 : off+16])),
					},
					Rot: [4]float32{
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+16 : off+20])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+20 : off+24])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+24 : off+28])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+28 : off+32])),
					},
					Scale: math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+32 : off+36])),
				}
				copy(wmo.Doodads[i].LightColor[:], chunkData[off+36:off+40])
			}
		}

		pos = chunkEnd
	}

	for i := range wmo.Materials {
		wmo.Materials[i].Texture = cstringAt(wmo.motx, wmo.Materials[i].Texture1)
		wmo.Materials[i].Texture2Name = cstringAt(wmo.motx, wmo.Materials[i].Texture2)
	}
	for i := range wmo.Doodads {
		wmo.Doodads[i].Name = cstringAt(wmo.modn, wmo.Doodads[i].NameOffset)
	}

	return wmo, nil
}

// cstringAt returns the NUL-terminated string starting at ofs in block.
func cstringAt(block []byte, ofs uint32) string {
	if int(ofs) >= len(block) {
		return ""
	}
	end := int(ofs)
	for end < len(block) && block[end] != 0 {
		end++
	}
	return string(block[ofs:end])
}

// GroupFileName returns the file name of group index i for a root WMO path
// ("World\\wmo\\x\\Foo.wmo" -> "World\\wmo\\x\\Foo_000.wmo").
func GroupFileName(rootPath string, i int) string {
	base := rootPath
	if len(base) > 4 && strings.EqualFold(base[len(base)-4:], ".wmo") {
		base = base[:len(base)-4]
	}
	return fmt.Sprintf("%s_%03d.wmo", base, i)
}

// ReadGroup parses a Group WMO from an io.Reader.
func ReadGroup(r io.Reader) (*Group, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read wmo group: %w", err)
	}

	grp := &Group{}
	pos := 0

	// Check if starts with MVER
	for pos+8 <= len(data) {
		fourCC := binary.LittleEndian.Uint32(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		chunkEnd := pos + 8 + size
		if chunkEnd > len(data) {
			break
		}
		chunkData := data[pos+8 : chunkEnd]

		switch fourCC {
		case MagicMOGP:
			// Parse sub-chunks inside MOGP starting at offset 68 (after MOGP header)
			if len(chunkData) > 68 {
				grp.Flags = binary.LittleEndian.Uint32(chunkData[8:12])
				for k := 0; k < 3; k++ {
					grp.BoundingBox[0][k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[12+k*4:]))
					grp.BoundingBox[1][k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[24+k*4:]))
				}
				parseGroupSubchunks(chunkData[68:], grp)
			}
		default:
			parseGroupChunk(fourCC, chunkData, grp)
		}

		pos = chunkEnd
	}

	return grp, nil
}

func parseGroupChunk(fourCC uint32, chunkData []byte, grp *Group) {
	switch fourCC {
	case MagicMOVT:
		grp.Vertices = parseVec3Array(chunkData)
	case MagicMONR:
		grp.Normals = parseVec3Array(chunkData)
	case MagicMOTV:
		// a second MOTV holds the UVs of the second texture layer
		switch {
		case grp.TexCoords == nil:
			grp.TexCoords = parseVec2Array(chunkData)
		case grp.TexCoords2 == nil:
			grp.TexCoords2 = parseVec2Array(chunkData)
		}
	case MagicMOCV:
		if grp.Colors == nil {
			grp.Colors = make([][4]uint8, len(chunkData)/4)
			for i := range grp.Colors {
				copy(grp.Colors[i][:], chunkData[i*4:i*4+4])
			}
		}
	case MagicMOVI:
		grp.Indices = parseIndices(chunkData)
	case MagicMOBA:
		grp.Batches = parseBatches(chunkData)
	}
}

func parseGroupSubchunks(data []byte, grp *Group) {
	pos := 0
	for pos+8 <= len(data) {
		fourCC := binary.LittleEndian.Uint32(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		chunkEnd := pos + 8 + size
		if chunkEnd > len(data) {
			break
		}
		chunkData := data[pos+8 : chunkEnd]

		parseGroupChunk(fourCC, chunkData, grp)
		pos = chunkEnd
	}
}

func parseVec3Array(data []byte) [][3]float32 {
	cnt := len(data) / 12
	res := make([][3]float32, cnt)
	for i := 0; i < cnt; i++ {
		off := i * 12
		res[i] = [3]float32{
			math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4])),
			math.Float32frombits(binary.LittleEndian.Uint32(data[off+4 : off+8])),
			math.Float32frombits(binary.LittleEndian.Uint32(data[off+8 : off+12])),
		}
	}
	return res
}

func parseVec2Array(data []byte) [][2]float32 {
	cnt := len(data) / 8
	res := make([][2]float32, cnt)
	for i := 0; i < cnt; i++ {
		off := i * 8
		res[i] = [2]float32{
			math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4])),
			math.Float32frombits(binary.LittleEndian.Uint32(data[off+4 : off+8])),
		}
	}
	return res
}

func parseIndices(data []byte) []uint16 {
	cnt := len(data) / 2
	res := make([]uint16, cnt)
	for i := 0; i < cnt; i++ {
		res[i] = binary.LittleEndian.Uint16(data[i*2 : (i+1)*2])
	}
	return res
}

func parseBatches(data []byte) []Batch {
	cnt := len(data) / 24
	res := make([]Batch, cnt)
	for i := 0; i < cnt; i++ {
		off := i * 24
		// SMOBatch: int16 bounds[6], uint32 startIndex, uint16 count,
		// uint16 minIndex, uint16 maxIndex, uint8 flags, uint8 materialId
		res[i] = Batch{
			StartIndex:  binary.LittleEndian.Uint32(data[off+12 : off+16]),
			IndexCount:  binary.LittleEndian.Uint16(data[off+16 : off+18]),
			VertexStart: binary.LittleEndian.Uint16(data[off+18 : off+20]),
			VertexEnd:   binary.LittleEndian.Uint16(data[off+20 : off+22]),
			MaterialID:  data[off+23],
		}
	}
	return res
}

func parseStringList(data []byte) []string {
	var res []string
	start := 0
	for i, b := range data {
		if b == 0 {
			if i > start {
				res = append(res, string(data[start:i]))
			}
			start = i + 1
		}
	}
	return res
}
