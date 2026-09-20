package wmo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

const (
	MagicMVER = 0x5245564D // "MVER"
	MagicMOHD = 0x44484F4D // "MOHD"
	MagicMOTX = 0x58544F4D // "MOTX"
	MagicMOMT = 0x544D4F4D // "MOMT"
	MagicMOGN = 0x4E474F4D // "MOGN"
	MagicMOGI = 0x49474F4D // "MOGI"
	MagicMODS = 0x53444F4D // "MODS"
	MagicMODN = 0x4E444F4D // "MODN"
	MagicMODD = 0x44444F4D // "MODD"
	MagicMOGP = 0x50474F4D // "MOGP"
	MagicMOVT = 0x54564F4D // "MOVT"
	MagicMONR = 0x524E4F4D // "MONR"
	MagicMOTV = 0x56544F4D // "MOTV"
	MagicMOVI = 0x49564F4D // "MOVI"
	MagicMOBA = 0x41424F4D // "MOBA"
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
	Texture1  uint32
	Color1    [4]uint8
}

type DoodadPlacement struct {
	NameOffset  uint32
	Pos         [3]float32
	Rot         [4]float32 // Quaternion
	Scale       float32
	LightColor  [4]uint8
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
	Vertices    [][3]float32
	Normals     [][3]float32
	TexCoords   [][2]float32
	Indices     []uint16
	Batches     []Batch
	BoundingBox [2][3]float32
}

type RootWMO struct {
	Header      Header
	Textures    []string
	Materials   []Material
	DoodadNames []string
	Doodads     []DoodadPlacement
	Groups      []*Group
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
			}

		case MagicMOTX:
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
				}
			}

		case MagicMODN:
			wmo.DoodadNames = parseStringList(chunkData)

		case MagicMODD:
			count := len(chunkData) / 40
			wmo.Doodads = make([]DoodadPlacement, count)
			for i := 0; i < count; i++ {
				off := i * 40
				wmo.Doodads[i] = DoodadPlacement{
					NameOffset: binary.LittleEndian.Uint32(chunkData[off : off+4]),
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
			}
		}

		pos = chunkEnd
	}

	return wmo, nil
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
				parseGroupSubchunks(chunkData[68:], grp)
			}
		case MagicMOVT:
			grp.Vertices = parseVec3Array(chunkData)
		case MagicMONR:
			grp.Normals = parseVec3Array(chunkData)
		case MagicMOTV:
			grp.TexCoords = parseVec2Array(chunkData)
		case MagicMOVI:
			grp.Indices = parseIndices(chunkData)
		case MagicMOBA:
			grp.Batches = parseBatches(chunkData)
		}

		pos = chunkEnd
	}

	return grp, nil
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

		switch fourCC {
		case MagicMOVT:
			grp.Vertices = parseVec3Array(chunkData)
		case MagicMONR:
			grp.Normals = parseVec3Array(chunkData)
		case MagicMOTV:
			grp.TexCoords = parseVec2Array(chunkData)
		case MagicMOVI:
			grp.Indices = parseIndices(chunkData)
		case MagicMOBA:
			grp.Batches = parseBatches(chunkData)
		}
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
		res[i] = Batch{
			StartIndex:  binary.LittleEndian.Uint32(data[off+4 : off+8]),
			IndexCount:  binary.LittleEndian.Uint16(data[off+8 : off+10]),
			VertexStart: binary.LittleEndian.Uint16(data[off+10 : off+12]),
			VertexEnd:   binary.LittleEndian.Uint16(data[off+12 : off+14]),
			MaterialID:  data[off+14],
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

// ExportGLB converts the WMO into a glTF GLB binary.
func (wmo *RootWMO) ExportGLB(w io.Writer) error {
	doc := gltf.NewDocument()

	matIndices := make([]int, len(wmo.Materials))
	for i := range wmo.Materials {
		mIdx := len(doc.Materials)
		matIndices[i] = mIdx
		doc.Materials = append(doc.Materials, gltf.Material{
			Name: fmt.Sprintf("WMOMaterial_%d", i),
			PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
				BaseColorFactor: [4]float32{0.7, 0.7, 0.7, 1.0},
				MetallicFactor:  0.0,
				RoughnessFactor: 0.8,
			},
			DoubleSided: true,
		})
	}

	for gIdx, grp := range wmo.Groups {
		if len(grp.Vertices) == 0 {
			continue
		}

		positions := make([][3]float32, len(grp.Vertices))
		normals := make([][3]float32, len(grp.Vertices))
		uvs := make([][2]float32, len(grp.Vertices))

		for i, v := range grp.Vertices {
			positions[i] = gltf.ConvertWoWToGLTPosition(v[0], v[1], v[2])
			if i < len(grp.Normals) {
				normals[i] = [3]float32{grp.Normals[i][0], grp.Normals[i][2], -grp.Normals[i][1]}
			}
			if i < len(grp.TexCoords) {
				uvs[i] = grp.TexCoords[i]
			}
		}

		posAcc := doc.AddFloat32Vec3Accessor(positions, gltf.TargetArrayBuffer)
		normAcc := doc.AddFloat32Vec3Accessor(normals, gltf.TargetArrayBuffer)
		uvAcc := doc.AddFloat32Vec2Accessor(uvs, gltf.TargetArrayBuffer)

		var primitives []gltf.Primitive

		if len(grp.Batches) > 0 {
			for _, b := range grp.Batches {
				start := int(b.StartIndex)
				count := int(b.IndexCount)
				if start+count <= len(grp.Indices) {
					subIndices := grp.Indices[start : start+count]
					idxAcc := doc.AddUint16IndicesAccessor(subIndices)

					var matRef *int
					if int(b.MaterialID) < len(matIndices) {
						matRef = &matIndices[b.MaterialID]
					}

					primitives = append(primitives, gltf.Primitive{
						Attributes: map[string]int{
							"POSITION":   posAcc,
							"NORMAL":     normAcc,
							"TEXCOORD_0": uvAcc,
						},
						Indices:  &idxAcc,
						Material: matRef,
					})
				}
			}
		}

		if len(primitives) == 0 && len(grp.Indices) > 0 {
			idxAcc := doc.AddUint16IndicesAccessor(grp.Indices)
			primitives = append(primitives, gltf.Primitive{
				Attributes: map[string]int{
					"POSITION":   posAcc,
					"NORMAL":     normAcc,
					"TEXCOORD_0": uvAcc,
				},
				Indices: &idxAcc,
			})
		}

		meshIdx := len(doc.Meshes)
		doc.Meshes = append(doc.Meshes, gltf.Mesh{
			Name:       fmt.Sprintf("WMOGroup_%d", gIdx),
			Primitives: primitives,
		})

		nodeIdx := len(doc.Nodes)
		doc.Nodes = append(doc.Nodes, gltf.Node{
			Name: fmt.Sprintf("WMONode_%d", gIdx),
			Mesh: &meshIdx,
		})
		doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)
	}

	return doc.ToGLB(w)
}
