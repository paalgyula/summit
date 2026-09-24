package adt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"strings"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

const (
	MagicMHDR = 0x5244484D // "MHDR"
	MagicMCIN = 0x4E49434D // "MCIN"
	MagicMTEX = 0x5845544D // "MTEX"
	MagicMMDX = 0x58444D4D // "MMDX"
	MagicMMID = 0x44494D4D // "MMID"
	MagicMWID = 0x4449574D // "MWID"
	MagicMDDF = 0x4644444D // "MDDF"
	MagicMCNK = 0x4B4E434D // "MCNK"
	MagicMCVT = 0x5456434D // "MCVT"
	MagicMCNR = 0x524E434D // "MCNR"
	MagicMCLY = 0x594C434D // "MCLY"
	MagicMCAL = 0x4C41434D // "MCAL"
	MagicMH2O = 0x4F32484D // "MH2O"

	TileSize  = float32(533.33333)
	ChunkSize = TileSize / 16.0
	UnitSize  = ChunkSize / 8.0
)

var ErrNotADT = errors.New("not a valid ADT terrain file")

type ChunkHeader struct {
	Flags       uint32
	IndexX      uint32
	IndexY      uint32
	NLayers     uint32
	NDoodadRefs uint32
	OfsMCVT     uint32
	OfsMCNR     uint32
	OfsMCLY     uint32
	OfsMCRF     uint32
	OfsMCAL     uint32
	SizeMCAL    uint32
	OfsMCSH     uint32
	SizeMCSH    uint32
	AreaID      uint32
	NMapObjRefs uint32
	Holes       uint16
	Padding     uint16
	OfsMH2O     uint32
	Pos         [3]float32 // World pos X, Y, Z
}

type Chunk struct {
	Header    ChunkHeader
	Heights   [145]float32    // 17 interleaved rows of 9 outer / 8 inner vertices
	Normals   [145][3]float32 // normalized X, Y, Z
	Layers    []Layer
	AlphaMaps [][AlphaMapSize * AlphaMapSize]uint8 // one per layer after the first
	// Liquids are the MH2O liquid layers covering this chunk.
	Liquids []LiquidLayer
}

// LiquidKind classifies a LiquidType.dbc id into the family the client draws
// it with (texture set and tint). Mirrors the LiquidType names in 3.3.5a.
type LiquidKind string

const (
	LiquidWater LiquidKind = "water"
	LiquidOcean LiquidKind = "ocean"
	LiquidMagma LiquidKind = "magma"
	LiquidSlime LiquidKind = "slime"
)

// LiquidLayer is one MH2O water/river/lava/slime layer of a chunk. The surface
// covers the cell rectangle [X, X+W) x [Y, Y+H) in 1/8-chunk units, flat at
// Level (the per-vertex height field is not used; the client draws it flat).
type LiquidLayer struct {
	Type  uint16
	Kind  LiquidKind
	Level float32
	X, Y  int
	W, H  int
}

// liquidKind maps a LiquidType.dbc id to the drawn family.
func liquidKind(id uint16) LiquidKind {
	switch id {
	case 2, 6, 10, 14:
		return LiquidOcean
	case 3, 7, 11, 15, 19, 121, 141:
		return LiquidMagma
	case 4, 8, 12, 20, 21, 181:
		return LiquidSlime
	default:
		return LiquidWater
	}
}

// LiquidExtras is attached to every water material in the tile GLB so the
// client can build the animated liquid surface.
type LiquidExtras struct {
	Kind  LiquidKind `json:"kind"`
	Type  uint16     `json:"type"`
	Level float32    `json:"level"`
}

type Layer struct {
	TextureID  uint32 // index into ADT.Textures
	Flags      uint32
	OffsetMCAL uint32
	EffectID   uint32
}

type DoodadPlacement struct {
	NameID   uint32
	UniqueID uint32
	Pos      [3]float32
	Rot      [3]float32
	Scale    uint16
	Flags    uint16
}

type WMOPlacement struct {
	NameID    uint32
	UniqueID  uint32
	Pos       [3]float32
	Rot       [3]float32
	Extents   [2][3]float32
	Flags     uint16
	DoodadSet uint16
	NameSet   uint16
}

type ADT struct {
	Textures   []string
	ModelNames []string // MMDX entries, indexed through MMID by DoodadPlacement.NameID
	WMONames   []string // MWMO entries, indexed through MWID by WMOPlacement.NameID
	DoodadDefs []DoodadPlacement
	WMODefs    []WMOPlacement
	Chunks     [16][16]*Chunk

	mmdx, mwmo []byte // raw name blocks, resolved once the offset tables are read
	mmid, mwid []uint32
	// mh2o is the raw MH2O chunk: 256 chunk headers followed by layer
	// information, vertex height fields and render bitmaps.
	mh2o []byte
}

// OpenADT reads and parses an ADT terrain file from disk.
func OpenADT(filePath string) (*ADT, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadADT(f)
}

// ReadADT parses an ADT terrain file from an io.Reader.
func ReadADT(r io.Reader) (*ADT, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read adt: %w", err)
	}

	adt := &ADT{}
	pos := 0

	for pos+8 <= len(data) {
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		chunkDataEnd := pos + 8 + size
		if chunkDataEnd > len(data) {
			break
		}

		chunkData := data[pos+8 : chunkDataEnd]

		tag := string(data[pos : pos+4])
		switch tag {
		case "MTEX", "XETM":
			adt.Textures = parseStringList(chunkData)

		case "MMDX", "XDMM":
			adt.mmdx = chunkData
		case "MMID", "DIMM":
			adt.mmid = parseUint32List(chunkData)
		case "MWMO", "OMWM":
			adt.mwmo = chunkData
		case "MWID", "DIWM":
			adt.mwid = parseUint32List(chunkData)

		case "MDDF", "FDDM":
			doodadCount := len(chunkData) / 36
			adt.DoodadDefs = make([]DoodadPlacement, doodadCount)
			for i := 0; i < doodadCount; i++ {
				off := i * 36
				adt.DoodadDefs[i] = DoodadPlacement{
					NameID:   binary.LittleEndian.Uint32(chunkData[off : off+4]),
					UniqueID: binary.LittleEndian.Uint32(chunkData[off+4 : off+8]),
					Pos: [3]float32{
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+8 : off+12])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+12 : off+16])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+16 : off+20])),
					},
					Rot: [3]float32{
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+20 : off+24])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+24 : off+28])),
						math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+28 : off+32])),
					},
					Scale: binary.LittleEndian.Uint16(chunkData[off+32 : off+34]),
					Flags: binary.LittleEndian.Uint16(chunkData[off+34 : off+36]),
				}
			}

		case "MODF", "FDOM":
			wmoCount := len(chunkData) / 64
			adt.WMODefs = make([]WMOPlacement, wmoCount)
			for i := 0; i < wmoCount; i++ {
				off := i * 64
				def := WMOPlacement{
					NameID:    binary.LittleEndian.Uint32(chunkData[off : off+4]),
					UniqueID:  binary.LittleEndian.Uint32(chunkData[off+4 : off+8]),
					Flags:     binary.LittleEndian.Uint16(chunkData[off+56 : off+58]),
					DoodadSet: binary.LittleEndian.Uint16(chunkData[off+58 : off+60]),
					NameSet:   binary.LittleEndian.Uint16(chunkData[off+60 : off+62]),
				}
				for k := 0; k < 3; k++ {
					def.Pos[k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+8+k*4:]))
					def.Rot[k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+20+k*4:]))
					def.Extents[0][k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+32+k*4:]))
					def.Extents[1][k] = math.Float32frombits(binary.LittleEndian.Uint32(chunkData[off+44+k*4:]))
				}
				adt.WMODefs[i] = def
			}

		case "MCNK", "KNCM":
			chunk := parseMCNK(chunkData)
			if chunk != nil && chunk.Header.IndexX < 16 && chunk.Header.IndexY < 16 {
				adt.Chunks[chunk.Header.IndexX][chunk.Header.IndexY] = chunk
			}

		case "MH2O", "O2HM":
			adt.mh2o = append([]byte(nil), chunkData...)
		}

		pos = chunkDataEnd
	}

	adt.ModelNames = resolveNames(adt.mmdx, adt.mmid)
	adt.WMONames = resolveNames(adt.mwmo, adt.mwid)
	adt.distributeLiquids()

	return adt, nil
}

// distributeLiquids attaches the MH2O layers to their chunks. The header array
// is indexed by the chunk's file order, which is x within y (i = y*16 + x).
func (adt *ADT) distributeLiquids() {
	const headerSize = 12
	if len(adt.mh2o) < 256*headerSize {
		return
	}

	d := adt.mh2o
	for i := 0; i < 256; i++ {
		offInfo := int(binary.LittleEndian.Uint32(d[i*headerSize : i*headerSize+4]))
		layers := int(binary.LittleEndian.Uint32(d[i*headerSize+4 : i*headerSize+8]))
		if offInfo == 0 || layers <= 0 {
			continue
		}

		x, y := i%16, i/16
		chunk := adt.Chunks[x][y]
		if chunk == nil {
			continue
		}

		for l := 0; l < layers; l++ {
			o := offInfo + l*24
			if o+24 > len(d) {
				break
			}
			// min_x/min_y/max_x/max_y are zero in practice; the covered
			// rectangle is x_offset/y_offset + width/height in 1/8 units.
			xo := int(d[o+12])
			yo := int(d[o+13])
			w := int(d[o+14])
			h := int(d[o+15])
			if w <= 0 || h <= 0 || xo+1 > 8 || yo+1 > 8 {
				continue
			}
			id := binary.LittleEndian.Uint16(d[o : o+2])
			level := math.Float32frombits(binary.LittleEndian.Uint32(d[o+4 : o+8]))
			chunk.Liquids = append(chunk.Liquids, LiquidLayer{
				Type:  id,
				Kind:  liquidKind(id),
				Level: level,
				X:     xo,
				Y:     yo,
				W:     w,
				H:     h,
			})
		}
	}
}

func parseUint32List(data []byte) []uint32 {
	res := make([]uint32, len(data)/4)
	for i := range res {
		res[i] = binary.LittleEndian.Uint32(data[i*4:])
	}
	return res
}

// resolveNames reads the NUL-terminated string at every offset of an
// MMID/MWID table out of its MMDX/MWMO block.
func resolveNames(block []byte, offsets []uint32) []string {
	names := make([]string, len(offsets))
	for i, ofs := range offsets {
		if int(ofs) >= len(block) {
			continue
		}
		end := int(ofs)
		for end < len(block) && block[end] != 0 {
			end++
		}
		names[i] = string(block[ofs:end])
	}
	return names
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

func parseMCNK(data []byte) *Chunk {
	if len(data) < 128 {
		return nil
	}

	c := &Chunk{}
	c.Header.Flags = binary.LittleEndian.Uint32(data[0:4])
	c.Header.IndexX = binary.LittleEndian.Uint32(data[4:8])
	c.Header.IndexY = binary.LittleEndian.Uint32(data[8:12])
	c.Header.NLayers = binary.LittleEndian.Uint32(data[12:16])
	c.Header.OfsMCVT = binary.LittleEndian.Uint32(data[20:24])
	c.Header.OfsMCNR = binary.LittleEndian.Uint32(data[24:28])
	c.Header.OfsMCLY = binary.LittleEndian.Uint32(data[28:32])
	c.Header.OfsMCAL = binary.LittleEndian.Uint32(data[36:40])
	c.Header.SizeMCAL = binary.LittleEndian.Uint32(data[40:44])
	c.Header.AreaID = binary.LittleEndian.Uint32(data[52:56])
	c.Header.Holes = binary.LittleEndian.Uint16(data[60:62])
	// SMChunk.position lives at 0x68; 0x74 is ofsMCCV, so reading from 108
	// yields (Y, Z, ~0) and collapses every chunk onto a single line.
	c.Header.Pos = [3]float32{
		math.Float32frombits(binary.LittleEndian.Uint32(data[104:108])),
		math.Float32frombits(binary.LittleEndian.Uint32(data[108:112])),
		math.Float32frombits(binary.LittleEndian.Uint32(data[112:116])),
	}

	// Parse MCVT (heights: 145 float32s)
	if c.Header.OfsMCVT > 0 {
		ofs := subChunkOffset(c.Header.OfsMCVT)
		if ofs+4 <= len(data) {
			tag := string(data[ofs : ofs+4])
			if tag == "MCVT" || tag == "TVCM" {
				ofs += 8
			}
		}
		if ofs+145*4 <= len(data) {
			for i := 0; i < 145; i++ {
				c.Heights[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[ofs+i*4 : ofs+(i+1)*4]))
			}
		}
	}

	// Parse MCNR (normals: 145 3-byte vectors)
	if c.Header.OfsMCNR > 0 {
		ofs := subChunkOffset(c.Header.OfsMCNR)
		if ofs+4 <= len(data) {
			tag := string(data[ofs : ofs+4])
			if tag == "MCNR" || tag == "RNCM" {
				ofs += 8
			}
		}
		if ofs+145*3 <= len(data) {
			for i := 0; i < 145; i++ {
				// Signed int8 X, Y, Z in world axes (verified against the height field)
				c.Normals[i] = [3]float32{
					float32(int8(data[ofs+i*3+0])) / 127.0,
					float32(int8(data[ofs+i*3+1])) / 127.0,
					float32(int8(data[ofs+i*3+2])) / 127.0,
				}
			}
		}
	}

	parseLayers(c, data)

	return c
}

// subChunkOffset converts an MCNK header offset (relative to the MCNK magic)
// into an index into the chunk body, which starts 8 bytes later.
func subChunkOffset(ofs uint32) int {
	if ofs < 8 {
		return 0
	}
	return int(ofs) - 8
}

// MCVT/MCNR hold 145 entries as 17 interleaved rows: 9 outer, 8 inner,
// 9 outer, ... (not 81 outer followed by 64 inner).
func outerIdx(row, col int) int { return row*17 + col }
func innerIdx(row, col int) int { return row*17 + 9 + col }

// GetHeight returns the terrain height at local tile coordinate (x, y).
func (adt *ADT) GetHeight(tileX, tileY float32) float32 {
	if tileX < 0 || tileX >= TileSize || tileY < 0 || tileY >= TileSize {
		return 0
	}

	chunkX := int(tileX / ChunkSize)
	chunkY := int(tileY / ChunkSize)
	if chunkX >= 16 {
		chunkX = 15
	}
	if chunkY >= 16 {
		chunkY = 15
	}

	chunk := adt.Chunks[chunkX][chunkY]
	if chunk == nil {
		return 0
	}

	localX := tileX - float32(chunkX)*ChunkSize
	localY := tileY - float32(chunkY)*ChunkSize

	unitX := int(localX / UnitSize)
	unitY := int(localY / UnitSize)
	if unitX >= 8 {
		unitX = 7
	}
	if unitY >= 8 {
		unitY = 7
	}

	// Bilinear center approximation:
	return chunk.Header.Pos[2] + chunk.Heights[innerIdx(unitY, unitX)]
}

// ChunkExtras is attached to every chunk material as glTF extras so the client
// can build the texture-splatting material. Layer textures are asset-server
// paths (WebP); the alpha atlas is the material's baseColorTexture, with the
// chunk's 64x64 block at (chunk[0]*64, chunk[1]*64) and layers 1..3 in R, G, B.
type ChunkExtras struct {
	Chunk  [2]int   `json:"chunk"`
	Layers []string `json:"layers"`
	AreaID uint32   `json:"areaId"`
}

// ExportGLB converts the ADT terrain tile into a glTF GLB binary: one mesh
// with a primitive per MCNK chunk, each with its own vertex accessors so
// loaders get per-chunk bounds. TEXCOORD_0 is chunk-local (0..1), which is
// the alpha map's parametrisation; detail textures repeat 8x per chunk.
func (adt *ADT) ExportGLB(w io.Writer) error {
	doc := gltf.NewDocument()
	atlas := image.NewNRGBA(image.Rect(0, 0, 16*AlphaMapSize, 16*AlphaMapSize))

	type chunkPrim struct {
		cx, cy    int
		positions [][3]float32
		normals   [][3]float32
		uvs       [][2]float32
		indices   []uint16
	}
	var prims []chunkPrim

	for cy := 0; cy < 16; cy++ {
		for cx := 0; cx < 16; cx++ {
			chunk := adt.Chunks[cx][cy]
			if chunk == nil {
				continue
			}
			prim := chunkPrim{cx: cx, cy: cy}

			addVertex := func(idx int, row, col float32) {
				posX := chunk.Header.Pos[0] - row*UnitSize
				posY := chunk.Header.Pos[1] - col*UnitSize
				posZ := chunk.Header.Pos[2] + chunk.Heights[idx]
				n := chunk.Normals[idx]
				prim.positions = append(prim.positions, gltf.ConvertWoWToGLTPosition(posX, posY, posZ))
				prim.normals = append(prim.normals, gltf.ConvertWoWToGLTPosition(n[0], n[1], n[2]))
				prim.uvs = append(prim.uvs, [2]float32{col / 8.0, row / 8.0})
			}

			// Output vertices are laid out as 81 outer (9x9) followed by 64 inner (8x8)
			for y := 0; y < 9; y++ {
				for x := 0; x < 9; x++ {
					addVertex(outerIdx(y, x), float32(y), float32(x))
				}
			}
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					addVertex(innerIdx(y, x), float32(y)+0.5, float32(x)+0.5)
				}
			}

			// 4 triangles per unit around the inner vertex
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					// Holes: one bit per 2x2 unit block in a 4x4 grid
					if chunk.Header.Holes&(1<<uint((y/2)*4+x/2)) != 0 {
						continue
					}
					tl := uint16(y*9 + x)
					tr := uint16(y*9 + (x + 1))
					bl := uint16((y+1)*9 + x)
					br := uint16((y+1)*9 + (x + 1))
					center := uint16(81 + y*8 + x)

					prim.indices = append(prim.indices, tl, tr, center)
					prim.indices = append(prim.indices, tr, br, center)
					prim.indices = append(prim.indices, br, bl, center)
					prim.indices = append(prim.indices, bl, tl, center)
				}
			}
			prims = append(prims, prim)

			// Alpha maps into the atlas: layer i+1 -> channel i
			for row := 0; row < AlphaMapSize; row++ {
				for col := 0; col < AlphaMapSize; col++ {
					o := atlas.PixOffset(cx*AlphaMapSize+col, cy*AlphaMapSize+row)
					for i := 0; i < 3 && i < len(chunk.AlphaMaps); i++ {
						atlas.Pix[o+i] = chunk.AlphaMaps[i][row*AlphaMapSize+col]
					}
					atlas.Pix[o+3] = 255
				}
			}
		}
	}

	if len(prims) == 0 {
		return errors.New("empty ADT terrain: no chunks found")
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, atlas); err != nil {
		return fmt.Errorf("encode alpha atlas: %w", err)
	}
	atlasTex := doc.AddEmbeddedTexture("AlphaAtlas", "image/png", pngBuf.Bytes(), gltf.Sampler{
		MagFilter: gltf.FilterLinear,
		MinFilter: gltf.FilterLinear,
		WrapS:     gltf.WrapClampToEdge,
		WrapT:     gltf.WrapClampToEdge,
	})

	mesh := gltf.Mesh{Name: "TerrainMesh"}
	for _, prim := range prims {
		if len(prim.indices) == 0 {
			continue
		}
		chunk := adt.Chunks[prim.cx][prim.cy]
		extras := ChunkExtras{Chunk: [2]int{prim.cx, prim.cy}, AreaID: chunk.Header.AreaID}
		for _, layer := range chunk.Layers {
			if int(layer.TextureID) < len(adt.Textures) {
				extras.Layers = append(extras.Layers, textureAssetPath(adt.Textures[layer.TextureID]))
			}
		}

		matIdx := len(doc.Materials)
		doc.Materials = append(doc.Materials, gltf.Material{
			Name: fmt.Sprintf("chunk_%d_%d", prim.cx, prim.cy),
			PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
				BaseColorFactor:  [4]float32{1, 1, 1, 1},
				BaseColorTexture: &gltf.TextureInfo{Index: atlasTex},
				MetallicFactor:   0.0,
				RoughnessFactor:  1.0,
			},
			Extras: extras,
		})

		idxAcc := doc.AddUint16IndicesAccessor(prim.indices)
		mesh.Primitives = append(mesh.Primitives, gltf.Primitive{
			Attributes: map[string]int{
				"POSITION":   doc.AddFloat32Vec3Accessor(prim.positions, gltf.TargetArrayBuffer),
				"NORMAL":     doc.AddFloat32Vec3Accessor(prim.normals, gltf.TargetArrayBuffer),
				"TEXCOORD_0": doc.AddFloat32Vec2Accessor(prim.uvs, gltf.TargetArrayBuffer),
			},
			Indices:  &idxAcc,
			Material: &matIdx,
		})
	}

	meshIdx := len(doc.Meshes)
	doc.Meshes = append(doc.Meshes, mesh)

	nodeIdx := len(doc.Nodes)
	doc.Nodes = append(doc.Nodes, gltf.Node{
		Name: "TerrainTile",
		Mesh: &meshIdx,
	})
	doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)

	adt.addLiquidMesh(doc)

	adt.addPlacementNodes(doc)

	return doc.ToGLB(w)
}

// addLiquidMesh appends the MH2O liquid surfaces as a "WaterTile" mesh: one
// primitive per liquid kind so the client can pick the animated material and
// tint. Every covered 1/8-chunk cell becomes a quad flat at the layer's level
// (the per-vertex height field is not used; sloped liquids are rare).
func (adt *ADT) addLiquidMesh(doc *gltf.Document) {
	type sink struct {
		positions [][3]float32
		normals   [][3]float32
		uvs       [][2]float32
		indices   []uint16
	}

	groups := map[LiquidKind][]*sink{}
	up := gltf.ConvertWoWToGLTPosition(0, 0, 1)
	// Indices are uint16, so a primitive holds at most 65535 vertices; a quad
	// that would overflow starts a new group.
	current := func(kind LiquidKind) *sink {
		gs := groups[kind]
		if len(gs) == 0 || len(gs[len(gs)-1].positions)+4 > 65535 {
			gs = append(gs, &sink{})
			groups[kind] = gs
		}
		return gs[len(gs)-1]
	}

	for cy := 0; cy < 16; cy++ {
		for cx := 0; cx < 16; cx++ {
			chunk := adt.Chunks[cx][cy]
			if chunk == nil {
				continue
			}
			for _, liq := range chunk.Liquids {
				for j := liq.Y; j < liq.Y+liq.H; j++ {
					for i := liq.X; i < liq.X+liq.W; i++ {
						s := current(liq.Kind)
						add := func(a, b float32) {
							posX := chunk.Header.Pos[0] - b*UnitSize
							posY := chunk.Header.Pos[1] - a*UnitSize
							s.positions = append(s.positions, gltf.ConvertWoWToGLTPosition(posX, posY, liq.Level))
							s.normals = append(s.normals, up)
							s.uvs = append(s.uvs, [2]float32{posX / 64, posY / 64})
						}
						base := uint16(len(s.positions))
						add(float32(i), float32(j))
						add(float32(i+1), float32(j))
						add(float32(i), float32(j+1))
						add(float32(i+1), float32(j+1))
						s.indices = append(s.indices, base, base+1, base+3, base, base+3, base+2)
					}
				}
			}
		}
	}

	total := 0
	for _, gs := range groups {
		for _, s := range gs {
			total += len(s.indices)
		}
	}
	if total == 0 {
		return
	}

	meshIdx := len(doc.Meshes)
	doc.Meshes = append(doc.Meshes, gltf.Mesh{Name: "WaterMesh"})
	mesh := &doc.Meshes[meshIdx]

	for kind, gs := range groups {
		matIdx := -1
		for _, s := range gs {
			if len(s.indices) == 0 {
				continue
			}
			if matIdx < 0 {
				matIdx = len(doc.Materials)
				doc.Materials = append(doc.Materials, gltf.Material{
					Name: "liquid_" + string(kind),
					PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
						BaseColorFactor: [4]float32{1, 1, 1, 1},
						MetallicFactor:  0,
						RoughnessFactor: 1,
					},
					DoubleSided: true,
					AlphaMode:   gltf.AlphaModeBlend,
					Extras:      LiquidExtras{Kind: kind},
				})
			}

			idxAcc := doc.AddUint16IndicesAccessor(s.indices)
			m := matIdx
			mesh.Primitives = append(mesh.Primitives, gltf.Primitive{
				Attributes: map[string]int{
					"POSITION":   doc.AddFloat32Vec3Accessor(s.positions, gltf.TargetArrayBuffer),
					"NORMAL":     doc.AddFloat32Vec3Accessor(s.normals, gltf.TargetArrayBuffer),
					"TEXCOORD_0": doc.AddFloat32Vec2Accessor(s.uvs, gltf.TargetArrayBuffer),
				},
				Indices:  &idxAcc,
				Material: &m,
			})
		}
	}

	nodeIdx := len(doc.Nodes)
	doc.Nodes = append(doc.Nodes, gltf.Node{Name: "WaterTile", Mesh: &meshIdx})
	doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)
}

// textureAssetPath turns an MTEX entry ("Tileset\\Durotar\\DurotarBase.blp")
// into the asset-server path of its WebP conversion.
func textureAssetPath(blpPath string) string {
	p := strings.ReplaceAll(blpPath, "\\", "/")
	if strings.HasSuffix(strings.ToLower(p), ".blp") {
		p = p[:len(p)-4]
	}
	return p + ".webp"
}

func parseString(s string) string {
	return strings.TrimSpace(s)
}
