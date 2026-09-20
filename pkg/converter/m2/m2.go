package m2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	MagicMD20 = 0x3032444D // "MD20"
	MagicMD21 = 0x3132444D // "MD21"

	VersionWotLK  = 208
	VersionModern = 274
)

var (
	ErrNotM2 = errors.New("not a valid M2 model file")
)

type M2Array struct {
	Count  uint32
	Offset uint32
}

type M2Sequence struct {
	ID             uint16
	VariationIndex uint16
	Duration       uint32
	MoveSpeed      float32
	Flags          uint32
	Frequency      int16
	Padding        uint16
	ReplayMin      uint32
	ReplayMax      uint32
	BlendTime      uint32
	BoundsMin      [3]float32
	BoundsMax      [3]float32
	BoundRadius    float32
	NextVariation  int16
	NextAlias      uint16
}

type M2Vertex struct {
	Pos         [3]float32
	BoneWeights [4]uint8
	BoneIndices [4]uint8
	Normal      [3]float32
	TexCoords   [2]float32
}

// ArrayRef is an M2Array header: element count and byte offset into the
// owning file (the .m2 itself, or an external .anim for that sequence).
type ArrayRef struct {
	Count  uint32
	Offset uint32
}

// M2Track is the WotLK (version >= 264) animation track header: one
// timestamp/value array pair per sequence.
type M2Track struct {
	Interpolation  uint16 // 0 none/step, 1 linear, 2 hermite, 3 bezier
	GlobalSequence int16  // >= 0: driven by a global loop instead of sequences
	Timestamps     []ArrayRef
	Values         []ArrayRef
}

type M2Bone struct {
	KeyBoneID   int32
	Flags       uint32
	ParentBone  int16
	SubmeshID   uint16
	Translation M2Track // C3Vector
	Rotation    M2Track // M2CompQuat (4 x int16)
	Scale       M2Track // C3Vector
	Pivot       [3]float32
}

// SequenceFlagInFile marks sequences whose animation data lives inside the
// .m2; otherwise it is stored in an external <Model><id>-<variation>.anim file.
const SequenceFlagInFile = 0x20

// SequenceFlagAlias marks sequences whose data is found in AliasNext.
const SequenceFlagAlias = 0x40

// AnimLoader returns the raw contents of the external .anim file for a sequence.
type AnimLoader func(sequenceID, variation uint16) ([]byte, error)

type M2Texture struct {
	Type  uint32
	Flags uint32
	Name  string
}

type Model struct {
	Version         uint32
	Name            string
	GlobalFlags     uint32
	GlobalSequences []uint32 // loop durations in ms
	Vertices        []M2Vertex
	Bones           []M2Bone
	Sequences       []M2Sequence
	Textures        []M2Texture

	raw []byte // file contents, needed to resolve animation tracks
}

// Open reads and parses an M2 model from a file on disk.
func Open(filePath string) (*Model, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// Read parses an M2 model from an io.Reader.
func Read(r io.Reader) (*Model, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read m2 data: %w", err)
	}

	if len(data) < 64 {
		return nil, ErrNotM2
	}

	base := 0
	magic := binary.LittleEndian.Uint32(data[0:4])

	// If wrapped in MD21 chunk
	if magic == MagicMD21 {
		base = 8
		if len(data) < base+64 {
			return nil, ErrNotM2
		}
		magic = binary.LittleEndian.Uint32(data[base : base+4])
	}

	if magic != MagicMD20 {
		return nil, fmt.Errorf("%w: magic 0x%08X", ErrNotM2, magic)
	}

	version := binary.LittleEndian.Uint32(data[base+4 : base+8])
	nameCount := binary.LittleEndian.Uint32(data[base+8 : base+12])
	nameOffset := binary.LittleEndian.Uint32(data[base+12 : base+16])
	flags := binary.LittleEndian.Uint32(data[base+16 : base+20])

	name := ""
	if nameCount > 0 && int(nameOffset)+int(nameCount) <= len(data) {
		name = strings.TrimRight(string(data[nameOffset:nameOffset+nameCount]), "\x00")
	}

	m := &Model{
		Version:     version,
		Name:        name,
		GlobalFlags: flags,
		raw:         data,
	}

	// In MD20 header layout:
	// base+20: global_loops (8 bytes)
	// base+28: sequences (8 bytes)
	// base+36: sequence_lookups (8 bytes)
	// base+44: bones (8 bytes)
	// base+52: key_bone_lookup (8 bytes)
	// base+60: vertices (8 bytes)
	readArr := func(ofs int) (uint32, uint32) {
		if ofs+8 > len(data) {
			return 0, 0
		}
		cnt := binary.LittleEndian.Uint32(data[ofs : ofs+4])
		off := binary.LittleEndian.Uint32(data[ofs+4 : ofs+8])
		return cnt, off
	}

	// Global sequences
	gsCnt, gsOfs := readArr(base + 20)
	if int(gsOfs)+int(gsCnt)*4 <= len(data) {
		m.GlobalSequences = make([]uint32, gsCnt)
		for i := uint32(0); i < gsCnt; i++ {
			m.GlobalSequences[i] = binary.LittleEndian.Uint32(data[int(gsOfs)+int(i)*4:])
		}
	}

	// Sequences
	seqCnt, seqOfs := readArr(base + 28)
	seqStride := 64
	if int(seqOfs)+int(seqCnt)*seqStride <= len(data) {
		m.Sequences = make([]M2Sequence, seqCnt)
		for i := uint32(0); i < seqCnt; i++ {
			sOff := int(seqOfs) + int(i)*seqStride
			m.Sequences[i] = M2Sequence{
				ID:             binary.LittleEndian.Uint16(data[sOff : sOff+2]),
				VariationIndex: binary.LittleEndian.Uint16(data[sOff+2 : sOff+4]),
				Duration:       binary.LittleEndian.Uint32(data[sOff+4 : sOff+8]),
				MoveSpeed:      mathFloat32(data[sOff+8 : sOff+12]),
				Flags:          binary.LittleEndian.Uint32(data[sOff+12 : sOff+16]),
				Frequency:      int16(binary.LittleEndian.Uint16(data[sOff+16 : sOff+18])),
				ReplayMin:      binary.LittleEndian.Uint32(data[sOff+20 : sOff+24]),
				ReplayMax:      binary.LittleEndian.Uint32(data[sOff+24 : sOff+28]),
				BlendTime:      binary.LittleEndian.Uint32(data[sOff+28 : sOff+32]),
				NextVariation:  int16(binary.LittleEndian.Uint16(data[sOff+60 : sOff+62])),
				NextAlias:      binary.LittleEndian.Uint16(data[sOff+62 : sOff+64]),
			}
		}
	}

	// Bones
	boneCnt, boneOfs := readArr(base + 44)
	boneStride := 88 // standard M2Bone stride in WotLK/Modern
	if int(boneOfs)+int(boneCnt)*boneStride <= len(data) {
		m.Bones = make([]M2Bone, boneCnt)
		for i := uint32(0); i < boneCnt; i++ {
			bOff := int(boneOfs) + int(i)*boneStride
			m.Bones[i] = M2Bone{
				KeyBoneID:  int32(binary.LittleEndian.Uint32(data[bOff : bOff+4])),
				Flags:      binary.LittleEndian.Uint32(data[bOff+4 : bOff+8]),
				ParentBone: int16(binary.LittleEndian.Uint16(data[bOff+8 : bOff+10])),
				SubmeshID:  binary.LittleEndian.Uint16(data[bOff+10 : bOff+12]),
				// tracks follow boneNameCRC (4); each M2Track header is 20 bytes
				Translation: readTrack(data, bOff+16),
				Rotation:    readTrack(data, bOff+36),
				Scale:       readTrack(data, bOff+56),
				Pivot: [3]float32{
					mathFloat32(data[bOff+76 : bOff+80]),
					mathFloat32(data[bOff+80 : bOff+84]),
					mathFloat32(data[bOff+84 : bOff+88]),
				},
			}
		}
	}

	// Vertices
	vertCnt, vertOfs := readArr(base + 60)
	vertStride := 48 // 48 bytes per vertex in 3.3.5a and modern M2
	if int(vertOfs)+int(vertCnt)*vertStride <= len(data) {
		m.Vertices = make([]M2Vertex, vertCnt)
		for i := uint32(0); i < vertCnt; i++ {
			vOff := int(vertOfs) + int(i)*vertStride
			var v M2Vertex
			v.Pos[0] = mathFloat32(data[vOff : vOff+4])
			v.Pos[1] = mathFloat32(data[vOff+4 : vOff+8])
			v.Pos[2] = mathFloat32(data[vOff+8 : vOff+12])

			v.BoneWeights[0] = data[vOff+12]
			v.BoneWeights[1] = data[vOff+13]
			v.BoneWeights[2] = data[vOff+14]
			v.BoneWeights[3] = data[vOff+15]

			v.BoneIndices[0] = data[vOff+16]
			v.BoneIndices[1] = data[vOff+17]
			v.BoneIndices[2] = data[vOff+18]
			v.BoneIndices[3] = data[vOff+19]

			v.Normal[0] = mathFloat32(data[vOff+20 : vOff+24])
			v.Normal[1] = mathFloat32(data[vOff+24 : vOff+28])
			v.Normal[2] = mathFloat32(data[vOff+28 : vOff+32])

			v.TexCoords[0] = mathFloat32(data[vOff+32 : vOff+36])
			v.TexCoords[1] = mathFloat32(data[vOff+36 : vOff+40])

			m.Vertices[i] = v
		}
	}

	return m, nil
}

// readTrack parses an M2Track header and its per-sequence array headers.
func readTrack(data []byte, off int) M2Track {
	t := M2Track{
		Interpolation:  binary.LittleEndian.Uint16(data[off : off+2]),
		GlobalSequence: int16(binary.LittleEndian.Uint16(data[off+2 : off+4])),
	}
	t.Timestamps = readArrayRefs(data, off+4)
	t.Values = readArrayRefs(data, off+12)
	return t
}

func readArrayRefs(data []byte, off int) []ArrayRef {
	cnt := binary.LittleEndian.Uint32(data[off : off+4])
	ofs := binary.LittleEndian.Uint32(data[off+4 : off+8])
	if cnt > 4096 || int(ofs)+int(cnt)*8 > len(data) {
		return nil
	}
	refs := make([]ArrayRef, cnt)
	for i := range refs {
		o := int(ofs) + i*8
		refs[i] = ArrayRef{
			Count:  binary.LittleEndian.Uint32(data[o : o+4]),
			Offset: binary.LittleEndian.Uint32(data[o+4 : o+8]),
		}
	}
	return refs
}

// TrackKeys resolves the keyframes of a track for one sequence index. It
// returns the timestamps (ms) and the raw value bytes (elemSize per key; three
// times that for hermite/bezier tracks, which store value/inTan/outTan).
// ext is the sequence's external .anim contents, or nil when the data is in-file.
func (m *Model) TrackKeys(t M2Track, seq int, elemSize int, ext []byte) ([]uint32, []byte) {
	if seq < 0 || seq >= len(t.Timestamps) || seq >= len(t.Values) {
		return nil, nil
	}
	src := m.raw
	if ext != nil {
		src = ext
	}
	ts, vs := t.Timestamps[seq], t.Values[seq]
	n := int(min(ts.Count, vs.Count))
	if n == 0 {
		return nil, nil
	}
	stride := elemSize
	if t.Interpolation >= 2 {
		stride *= 3
	}
	if int(ts.Offset)+n*4 > len(src) || int(vs.Offset)+n*stride > len(src) {
		return nil, nil
	}
	times := make([]uint32, n)
	for i := range times {
		times[i] = binary.LittleEndian.Uint32(src[int(ts.Offset)+i*4:])
	}
	return times, src[int(vs.Offset) : int(vs.Offset)+n*stride]
}

func mathFloat32(b []byte) float32 {
	bits := binary.LittleEndian.Uint32(b)
	var f float32
	binary.LittleEndian.PutUint32(b, bits)
	_ = binary.Read(binaryLittleEndianReader(b), binary.LittleEndian, &f)
	return f
}

type binaryLittleEndianReader []byte

func (b binaryLittleEndianReader) Read(p []byte) (n int, err error) {
	copy(p, b)
	return len(b), io.EOF
}
