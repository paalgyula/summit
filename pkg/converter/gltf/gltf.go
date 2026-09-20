package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	GLBMagic = 0x46546C67 // "glTF"
	GLBJSON  = 0x4E4F534A // "JSON"
	GLBBIN   = 0x004E4942 // "BIN\x00"

	CompTypeByte          = 5120
	CompTypeUnsignedByte  = 5121
	CompTypeShort         = 5122
	CompTypeUnsignedShort = 5123
	CompTypeUnsignedInt   = 5125
	CompTypeFloat         = 5126

	TargetArrayBuffer        = 34962
	TargetElementArrayBuffer = 34963
)

type Document struct {
	Asset       Asset        `json:"asset"`
	Scene       *int         `json:"scene,omitempty"`
	Scenes      []Scene      `json:"scenes,omitempty"`
	Nodes       []Node       `json:"nodes,omitempty"`
	Meshes      []Mesh       `json:"meshes,omitempty"`
	Skins       []Skin       `json:"skins,omitempty"`
	Animations  []Animation  `json:"animations,omitempty"`
	Materials   []Material   `json:"materials,omitempty"`
	Textures    []Texture    `json:"textures,omitempty"`
	Images      []Image      `json:"images,omitempty"`
	Samplers    []Sampler    `json:"samplers,omitempty"`
	Accessors   []Accessor   `json:"accessors,omitempty"`
	BufferViews []BufferView `json:"bufferViews,omitempty"`
	Buffers     []Buffer     `json:"buffers,omitempty"`

	// Internal binary buffer accumulator
	binBuffer bytes.Buffer
}

type Asset struct {
	Version   string `json:"version"`
	Generator string `json:"generator,omitempty"`
}

type Scene struct {
	Name  string `json:"name,omitempty"`
	Nodes []int  `json:"nodes"`
}

type Node struct {
	Name        string      `json:"name,omitempty"`
	Mesh        *int        `json:"mesh,omitempty"`
	Skin        *int        `json:"skin,omitempty"`
	Children    []int       `json:"children,omitempty"`
	Translation *[3]float32 `json:"translation,omitempty"`
	Rotation    *[4]float32 `json:"rotation,omitempty"` // x, y, z, w
	Scale       *[3]float32 `json:"scale,omitempty"`
}

type Mesh struct {
	Name       string      `json:"name,omitempty"`
	Primitives []Primitive `json:"primitives"`
}

type Primitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices,omitempty"`
	Material   *int           `json:"material,omitempty"`
	Mode       int            `json:"mode,omitempty"` // default 4 = TRIANGLES
}

type Skin struct {
	Name                string `json:"name,omitempty"`
	InverseBindMatrices *int   `json:"inverseBindMatrices,omitempty"`
	Joints              []int  `json:"joints"`
	Skeleton            *int   `json:"skeleton,omitempty"`
}

type Animation struct {
	Name     string             `json:"name,omitempty"`
	Channels []AnimationChannel `json:"channels"`
	Samplers []AnimationSampler `json:"samplers"`
	Extras   any                `json:"extras,omitempty"`
}

type AnimationChannel struct {
	Sampler int                    `json:"sampler"`
	Target  AnimationChannelTarget `json:"target"`
}

type AnimationChannelTarget struct {
	Node int    `json:"node"`
	Path string `json:"path"` // "translation", "rotation", "scale"
}

type AnimationSampler struct {
	Input         int    `json:"input"`
	Interpolation string `json:"interpolation,omitempty"` // "LINEAR", "STEP"
	Output        int    `json:"output"`
}

type Material struct {
	Name                 string                `json:"name,omitempty"`
	PbrMetallicRoughness *PbrMetallicRoughness `json:"pbrMetallicRoughness,omitempty"`
	DoubleSided          bool                  `json:"doubleSided,omitempty"`
	Extras               any                   `json:"extras,omitempty"`
}

type PbrMetallicRoughness struct {
	BaseColorFactor  [4]float32   `json:"baseColorFactor,omitempty"`
	BaseColorTexture *TextureInfo `json:"baseColorTexture,omitempty"`
	MetallicFactor   float32      `json:"metallicFactor"`
	RoughnessFactor  float32      `json:"roughnessFactor"`
}

type TextureInfo struct {
	Index    int `json:"index"`
	TexCoord int `json:"texCoord,omitempty"`
}

type Texture struct {
	Name    string `json:"name,omitempty"`
	Sampler *int   `json:"sampler,omitempty"`
	Source  int    `json:"source"`
}

type Image struct {
	Name       string `json:"name,omitempty"`
	MimeType   string `json:"mimeType"`
	BufferView int    `json:"bufferView"`
}

type Sampler struct {
	MagFilter int `json:"magFilter,omitempty"`
	MinFilter int `json:"minFilter,omitempty"`
	WrapS     int `json:"wrapS,omitempty"`
	WrapT     int `json:"wrapT,omitempty"`
}

const (
	FilterLinear    = 9729
	WrapClampToEdge = 33071
	WrapRepeat      = 10497
)

type Accessor struct {
	BufferView    int       `json:"bufferView"`
	ByteOffset    int       `json:"byteOffset,omitempty"`
	ComponentType int       `json:"componentType"`
	Count         int       `json:"count"`
	Type          string    `json:"type"` // "SCALAR", "VEC2", "VEC3", "VEC4", "MAT4"
	Min           []float32 `json:"min,omitempty"`
	Max           []float32 `json:"max,omitempty"`
}

type BufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	ByteStride int `json:"byteStride,omitempty"`
	Target     int `json:"target,omitempty"`
}

type Buffer struct {
	ByteLength int `json:"byteLength"`
}

func NewDocument() *Document {
	defScene := 0
	return &Document{
		Asset: Asset{
			Version:   "2.0",
			Generator: "Summit WoW glTF Converter",
		},
		Scene:  &defScene,
		Scenes: []Scene{{Name: "Scene", Nodes: []int{}}},
	}
}

// AddBufferView writes data into the document's binary buffer aligned to 4 bytes and creates a BufferView.
func (doc *Document) AddBufferView(data []byte, target int, byteStride int) int {
	// 4-byte alignment
	pad := (4 - (doc.binBuffer.Len() % 4)) % 4
	for i := 0; i < pad; i++ {
		doc.binBuffer.WriteByte(0)
	}

	offset := doc.binBuffer.Len()
	doc.binBuffer.Write(data)
	length := len(data)

	bv := BufferView{
		Buffer:     0,
		ByteOffset: offset,
		ByteLength: length,
		ByteStride: byteStride,
		Target:     target,
	}

	idx := len(doc.BufferViews)
	doc.BufferViews = append(doc.BufferViews, bv)
	return idx
}

// AddFloat32Vec3Accessor adds an accessor for a slice of [3]float32 (e.g. POSITION or NORMAL).
func (doc *Document) AddFloat32Vec3Accessor(vecs [][3]float32, target int) int {
	if len(vecs) == 0 {
		return -1
	}

	buf := new(bytes.Buffer)
	minVal := [3]float32{vecs[0][0], vecs[0][1], vecs[0][2]}
	maxVal := [3]float32{vecs[0][0], vecs[0][1], vecs[0][2]}

	for _, v := range vecs {
		for i := 0; i < 3; i++ {
			if v[i] < minVal[i] {
				minVal[i] = v[i]
			}
			if v[i] > maxVal[i] {
				maxVal[i] = v[i]
			}
			_ = binary.Write(buf, binary.LittleEndian, v[i])
		}
	}

	bvIdx := doc.AddBufferView(buf.Bytes(), target, 0)
	accIdx := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    bvIdx,
		ComponentType: CompTypeFloat,
		Count:         len(vecs),
		Type:          "VEC3",
		Min:           []float32{minVal[0], minVal[1], minVal[2]},
		Max:           []float32{maxVal[0], maxVal[1], maxVal[2]},
	})

	return accIdx
}

// AddFloat32Vec2Accessor adds an accessor for a slice of [2]float32 (e.g. TEXCOORD_0).
func (doc *Document) AddFloat32Vec2Accessor(vecs [][2]float32, target int) int {
	if len(vecs) == 0 {
		return -1
	}

	buf := new(bytes.Buffer)
	for _, v := range vecs {
		_ = binary.Write(buf, binary.LittleEndian, v[0])
		_ = binary.Write(buf, binary.LittleEndian, v[1])
	}

	bvIdx := doc.AddBufferView(buf.Bytes(), target, 0)
	accIdx := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    bvIdx,
		ComponentType: CompTypeFloat,
		Count:         len(vecs),
		Type:          "VEC2",
	})

	return accIdx
}

// AddUint16IndicesAccessor adds an accessor for triangle indices.
func (doc *Document) AddUint16IndicesAccessor(indices []uint16) int {
	if len(indices) == 0 {
		return -1
	}

	buf := new(bytes.Buffer)
	for _, idx := range indices {
		_ = binary.Write(buf, binary.LittleEndian, idx)
	}

	bvIdx := doc.AddBufferView(buf.Bytes(), TargetElementArrayBuffer, 0)
	accIdx := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    bvIdx,
		ComponentType: CompTypeUnsignedShort,
		Count:         len(indices),
		Type:          "SCALAR",
	})

	return accIdx
}

// AddJointsWeightsAccessor adds JOINTS_0 (VEC4 of uint16) and WEIGHTS_0 (VEC4 of float32).
func (doc *Document) AddJointsWeightsAccessor(joints [][4]uint16, weights [][4]float32) (int, int) {
	if len(joints) != len(weights) || len(joints) == 0 {
		return -1, -1
	}

	jBuf := new(bytes.Buffer)
	wBuf := new(bytes.Buffer)

	for i := range joints {
		for k := 0; k < 4; k++ {
			_ = binary.Write(jBuf, binary.LittleEndian, joints[i][k])
			_ = binary.Write(wBuf, binary.LittleEndian, weights[i][k])
		}
	}

	jBv := doc.AddBufferView(jBuf.Bytes(), TargetArrayBuffer, 0)
	jAcc := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    jBv,
		ComponentType: CompTypeUnsignedShort,
		Count:         len(joints),
		Type:          "VEC4",
	})

	wBv := doc.AddBufferView(wBuf.Bytes(), TargetArrayBuffer, 0)
	wAcc := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    wBv,
		ComponentType: CompTypeFloat,
		Count:         len(weights),
		Type:          "VEC4",
	})

	return jAcc, wAcc
}

// AddFloat32ScalarAccessor adds a SCALAR float32 accessor with min/max (e.g. animation input times).
func (doc *Document) AddFloat32ScalarAccessor(values []float32) int {
	if len(values) == 0 {
		return -1
	}
	buf := new(bytes.Buffer)
	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		minVal = min(minVal, v)
		maxVal = max(maxVal, v)
		_ = binary.Write(buf, binary.LittleEndian, v)
	}
	bvIdx := doc.AddBufferView(buf.Bytes(), 0, 0)
	accIdx := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    bvIdx,
		ComponentType: CompTypeFloat,
		Count:         len(values),
		Type:          "SCALAR",
		Min:           []float32{minVal},
		Max:           []float32{maxVal},
	})
	return accIdx
}

// AddFloat32Vec4Accessor adds a VEC4 float32 accessor (e.g. rotation quaternions).
func (doc *Document) AddFloat32Vec4Accessor(vecs [][4]float32, target int) int {
	if len(vecs) == 0 {
		return -1
	}
	buf := new(bytes.Buffer)
	for _, v := range vecs {
		for i := 0; i < 4; i++ {
			_ = binary.Write(buf, binary.LittleEndian, v[i])
		}
	}
	bvIdx := doc.AddBufferView(buf.Bytes(), target, 0)
	accIdx := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    bvIdx,
		ComponentType: CompTypeFloat,
		Count:         len(vecs),
		Type:          "VEC4",
	})
	return accIdx
}

// AddEmbeddedTexture stores encoded image bytes in the binary buffer and
// returns the index of a texture referencing it.
func (doc *Document) AddEmbeddedTexture(name, mimeType string, data []byte, sampler Sampler) int {
	bvIdx := doc.AddBufferView(data, 0, 0)
	imgIdx := len(doc.Images)
	doc.Images = append(doc.Images, Image{Name: name, MimeType: mimeType, BufferView: bvIdx})
	sIdx := len(doc.Samplers)
	doc.Samplers = append(doc.Samplers, sampler)
	texIdx := len(doc.Textures)
	doc.Textures = append(doc.Textures, Texture{Name: name, Sampler: &sIdx, Source: imgIdx})
	return texIdx
}

// AddInverseBindMatrices adds a MAT4 float32 accessor for skin inverse bind matrices.
func (doc *Document) AddInverseBindMatrices(matrices [][16]float32) int {
	buf := new(bytes.Buffer)
	for _, m := range matrices {
		for i := 0; i < 16; i++ {
			_ = binary.Write(buf, binary.LittleEndian, m[i])
		}
	}
	bvIdx := doc.AddBufferView(buf.Bytes(), 0, 0)
	accIdx := len(doc.Accessors)
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView:    bvIdx,
		ComponentType: CompTypeFloat,
		Count:         len(matrices),
		Type:          "MAT4",
	})
	return accIdx
}

// ToGLB serializes the entire glTF document into a binary .glb container.
func (doc *Document) ToGLB(w io.Writer) error {
	doc.Buffers = []Buffer{{ByteLength: doc.binBuffer.Len()}}

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal gltf json: %w", err)
	}

	// 4-byte padding for JSON chunk with spaces (0x20)
	jsonPad := (4 - (len(jsonBytes) % 4)) % 4
	for i := 0; i < jsonPad; i++ {
		jsonBytes = append(jsonBytes, 0x20)
	}

	// 4-byte padding for BIN chunk with zeros (0x00)
	binBytes := doc.binBuffer.Bytes()
	binPad := (4 - (len(binBytes) % 4)) % 4
	for i := 0; i < binPad; i++ {
		binBytes = append(binBytes, 0x00)
	}

	totalLength := 12 + (8 + len(jsonBytes)) + (8 + len(binBytes))

	// Write 12-byte GLB header
	header := make([]byte, 12)
	binary.LittleEndian.PutUint32(header[0:4], GLBMagic)
	binary.LittleEndian.PutUint32(header[4:8], 2)
	binary.LittleEndian.PutUint32(header[8:12], uint32(totalLength))
	if _, err := w.Write(header); err != nil {
		return err
	}

	// Write JSON chunk header + content
	jsonHeader := make([]byte, 8)
	binary.LittleEndian.PutUint32(jsonHeader[0:4], uint32(len(jsonBytes)))
	binary.LittleEndian.PutUint32(jsonHeader[4:8], GLBJSON)
	if _, err := w.Write(jsonHeader); err != nil {
		return err
	}
	if _, err := w.Write(jsonBytes); err != nil {
		return err
	}

	// Write BIN chunk header + content
	binHeader := make([]byte, 8)
	binary.LittleEndian.PutUint32(binHeader[0:4], uint32(len(binBytes)))
	binary.LittleEndian.PutUint32(binHeader[4:8], GLBBIN)
	if _, err := w.Write(binHeader); err != nil {
		return err
	}
	if _, err := w.Write(binBytes); err != nil {
		return err
	}

	return nil
}

// SaveGLB writes the document to a file on disk as .glb.
func (doc *Document) SaveGLB(destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return doc.ToGLB(f)
}

// ConvertWoWToGLTPosition converts WoW coordinates to glTF/Three.js coordinates.
// WoW: +X North, +Y West, +Z Up
// Three.js: +X Right, +Y Up, -Z Forward
func ConvertWoWToGLTPosition(wowX, wowY, wowZ float32) [3]float32 {
	return [3]float32{-wowY, wowZ, -wowX}
}

// ConvertM2ToGLTPosition converts local M2 model coordinates to glTF coordinates.
// M2 model space uses the same axes as the world: +X Forward (facing), +Y Left, +Z Up
// glTF: +X Right, +Y Up, -Z Forward
func ConvertM2ToGLTPosition(m2X, m2Y, m2Z float32) [3]float32 {
	return [3]float32{-m2Y, m2Z, -m2X}
}
