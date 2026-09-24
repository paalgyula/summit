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
	TexCoords2  [2]float32 // second UV set (multi-texture batches)
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

// GlobalFlagTextureCombiners (M2 header flag 0x08): batches' shader_id
// indexes the texture_combiner_combos table.
const GlobalFlagTextureCombiners = 0x08

// Texture types: 0 is a fixed file name, the others are replaced at runtime
// (character skin, hair, creature skins, ...).
const (
	TextureTypeFilename = 0
	TextureTypeSkin     = 1
	TextureTypeHair     = 6
)

// Texture flags.
const (
	TextureFlagWrapX = 0x1
	TextureFlagWrapY = 0x2
)

type M2Texture struct {
	Type  uint32
	Flags uint32
	Name  string // BLP path for TextureTypeFilename
}

// Render flags (M2Material.flags).
const (
	MaterialFlagUnlit     = 0x01
	MaterialFlagUnfogged  = 0x02
	MaterialFlagTwoSided  = 0x04
	MaterialFlagDepthTest = 0x08
	MaterialFlagNoDepthWr = 0x10
)

// Blend modes (M2Material.blending_mode).
const (
	BlendOpaque   = 0
	BlendAlphaKey = 1
	BlendAlpha    = 2
	BlendNoAlpha  = 3
	BlendAdd      = 4
	BlendMod      = 5
	BlendMod2x    = 6
)

type M2Material struct {
	Flags     uint16
	BlendMode uint16
}

// M2Color is a per-texture-unit colour and alpha animation (M2Color).
type M2Color struct {
	Color M2Track // C3Vector
	Alpha M2Track // fixed16
}

// M2TextureTransform animates a texture's UVs (M2TextureTransform, 60 bytes).
type M2TextureTransform struct {
	Translation M2Track // C3Vector
	Rotation    M2Track // C4Quaternion
	Scaling     M2Track // C3Vector
}

// M2Light is a light embedded in the model (M2Light record, 156 bytes).
type M2Light struct {
	Type             uint16 // 0 directional, 1 point
	Bone             int16
	Position         [3]float32
	AmbientColor     M2Track // C3Vector
	AmbientIntensity M2Track // float
	DiffuseColor     M2Track // C3Vector
	DiffuseIntensity M2Track // float
	AttenuationStart M2Track // float
	AttenuationEnd   M2Track // float
	Visibility       M2Track // uint8
}

// M2Camera is a camera embedded in the model (login/character screens,
// cinematics). Positions are in model space; FOV is the diagonal field of
// view in radians (WotLK).
type M2Camera struct {
	Type     int32 // -1 portrait, 0 character info, 1+ flyby
	FOV      float32
	FarClip  float32
	NearClip float32
	Position M2Track // M2SplineKey<C3Vector>
	PosBase  [3]float32
	Target   M2Track // M2SplineKey<C3Vector>
	TgtBase  [3]float32
	Roll     M2Track // M2SplineKey<float>
}

// M2Attachment is a named point on a bone that equipment, spell effects and
// the like are attached to (M2Attachment record, 40 bytes). Position is in
// model space, moved by the bone.
type M2Attachment struct {
	ID       uint32 // attachment id: 0 shield, 1 right hand, 2 left hand, 5/6 shoulders, 11 helm, 26+ sheaths, ...
	Bone     uint16
	Position [3]float32
}

// M2PartTrack is the "fake" animation block used by particle emitters: a flat
// list of fixed16 timestamps and values, unlike the per-sequence M2Track.
// Times are normalised 0..1 over a particle's lifetime (0x7FFF = 1.0).
type M2PartTrack struct {
	Times    []int16
	Values   []byte // raw value bytes, ElemSize per key
	ElemSize int
}

// Particle emitter types (M2Particle.emitterType).
const (
	ParticleEmitterPlane  = 1
	ParticleEmitterSphere = 2
	ParticleEmitterSpline = 3
	ParticleEmitterBone   = 4
)

// Particle flags (M2Particle.flags).
const (
	ParticleFlagUnshaded      = 0x01
	ParticleFlagUnfogged      = 0x02
	ParticleFlagUseModelSpace = 0x10
)

// Particle blending modes (M2Particle.blendingType). This is a distinct
// enum from M2Material.blending_mode: 3 and 4 are additive, not modulate.
const (
	ParticleBlendOpaque     = 0
	ParticleBlendAlphaKey   = 1
	ParticleBlendAlpha      = 2
	ParticleBlendAdd        = 3
	ParticleBlendAddAlpha   = 4
	ParticleBlendMod        = 5
	ParticleBlendModulate2x = 6
)

// M2Particle is the WotLK (v264) particle emitter record (476 bytes). When the
// header flag 0x200 is set or the version is newer than 271 the record grows to
// 492 bytes with the Cataclysm multi-texture parameters.
type M2Particle struct {
	ID          uint32
	Flags       uint32
	Position    [3]float32 // relative to Bone
	Bone        uint16
	Texture     uint16
	Blending    uint8
	EmitterType uint8
	ColorIndex  uint16
	ParticleTyp uint8
	HeadOrTail  uint8

	TextureTileRotation uint16
	TextureRows         uint16
	TextureCols         uint16

	EmissionSpeed   M2Track // float
	SpeedVariation  M2Track // float
	VerticalRange   M2Track // float
	HorizontalRange M2Track // float
	Gravity         M2Track // float
	Lifespan        M2Track // float
	LifespanVary    float32
	EmissionRate    M2Track // float
	EmissionRateVar float32
	AreaLength      M2Track // float
	AreaWidth       M2Track // float
	ZSource         M2Track // float

	ColorTrack M2PartTrack // C3Vector
	AlphaTrack M2PartTrack // fixed16
	ScaleTrack M2PartTrack // C2Vector
	ScaleVary  [2]float32
	HeadCell   M2PartTrack // uint16
	TailCell   M2PartTrack // uint16

	TailLength      float32
	TwinkleSpeed    float32
	TwinklePercent  float32
	TwinkleScale    [2]float32
	BurstMultiplier float32
	Drag            float32
	BaseSpin        float32
	BaseSpinVary    float32
	Spin            float32
	SpinVary        float32
	Tumble          [2][3]float32
	WindVector      [3]float32
	WindTime        float32
	FollowSpeed1    float32
	FollowScale1    float32
	FollowSpeed2    float32
	FollowScale2    float32
	SplinePoints    []uint32 // byte offsets into raw, resolved to points
	EnabledIn       M2Track  // uint8
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
	Materials       []M2Material
	TextureLookup   []uint16 // texture_lookup_table: batch texture combo -> Textures index
	TexUnitLookup   []int16  // texture_unit_lookup_table: batch texcoord combo -> 0 uv0, 1 uv1, -1 env map
	Cameras         []M2Camera
	Lights          []M2Light
	Attachments     []M2Attachment
	Particles       []M2Particle
	// TextureCombinerCombos lists the per-texture combiner ops batches index
	// with shader_id when GlobalFlagTextureCombiners is set.
	TextureCombinerCombos  []uint16
	TextureTransforms      []M2TextureTransform
	TextureTransformLookup []int16 // batch transform combo -> TextureTransforms index (-1 none)
	Colors                 []M2Color
	TextureWeights         []M2Track // transparency animations (fixed16)
	TransparencyLookup     []uint16  // batch weight combo -> TextureWeights index

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
			v.TexCoords2[0] = mathFloat32(data[vOff+40 : vOff+44])
			v.TexCoords2[1] = mathFloat32(data[vOff+44 : vOff+48])

			m.Vertices[i] = v
		}
	}

	// Textures (base+80): M2Texture{type u32, flags u32, filename M2Array<char>}
	texCnt, texOfs := readArr(base + 80)
	if int(texOfs)+int(texCnt)*16 <= len(data) {
		m.Textures = make([]M2Texture, texCnt)
		for i := uint32(0); i < texCnt; i++ {
			tOff := int(texOfs) + int(i)*16
			tex := M2Texture{
				Type:  binary.LittleEndian.Uint32(data[tOff : tOff+4]),
				Flags: binary.LittleEndian.Uint32(data[tOff+4 : tOff+8]),
			}
			nCnt, nOfs := readArr(tOff + 8)
			if nCnt > 0 && int(nOfs)+int(nCnt) <= len(data) {
				tex.Name = strings.TrimRight(string(data[nOfs:nOfs+nCnt]), "\x00")
			}
			m.Textures[i] = tex
		}
	}

	// Materials (base+112): {flags u16, blending_mode u16}
	matCnt, matOfs := readArr(base + 112)
	if int(matOfs)+int(matCnt)*4 <= len(data) {
		m.Materials = make([]M2Material, matCnt)
		for i := uint32(0); i < matCnt; i++ {
			mOff := int(matOfs) + int(i)*4
			m.Materials[i] = M2Material{
				Flags:     binary.LittleEndian.Uint16(data[mOff : mOff+2]),
				BlendMode: binary.LittleEndian.Uint16(data[mOff+2 : mOff+4]),
			}
		}
	}

	// Colours (base+72): M2Color{color track, alpha track}, 40 bytes
	colCnt, colOfs := readArr(base + 72)
	if int(colOfs)+int(colCnt)*40 <= len(data) {
		m.Colors = make([]M2Color, colCnt)
		for i := uint32(0); i < colCnt; i++ {
			c := int(colOfs) + int(i)*40
			m.Colors[i] = M2Color{Color: readTrack(data, c), Alpha: readTrack(data, c+20)}
		}
	}

	// Texture weights (base+88): one fixed16 track each
	twCnt, twOfs := readArr(base + 88)
	if int(twOfs)+int(twCnt)*20 <= len(data) {
		m.TextureWeights = make([]M2Track, twCnt)
		for i := uint32(0); i < twCnt; i++ {
			m.TextureWeights[i] = readTrack(data, int(twOfs)+int(i)*20)
		}
	}

	// Transparency lookup table (base+144)
	trCnt, trOfs := readArr(base + 144)
	if int(trOfs)+int(trCnt)*2 <= len(data) {
		m.TransparencyLookup = make([]uint16, trCnt)
		for i := uint32(0); i < trCnt; i++ {
			m.TransparencyLookup[i] = binary.LittleEndian.Uint16(data[int(trOfs)+int(i)*2:])
		}
	}

	// Texture transforms (base+96) and their lookup table (base+152)
	ttCnt, ttOfs := readArr(base + 96)
	if int(ttOfs)+int(ttCnt)*60 <= len(data) {
		m.TextureTransforms = make([]M2TextureTransform, ttCnt)
		for i := uint32(0); i < ttCnt; i++ {
			o := int(ttOfs) + int(i)*60
			m.TextureTransforms[i] = M2TextureTransform{
				Translation: readTrack(data, o),
				Rotation:    readTrack(data, o+20),
				Scaling:     readTrack(data, o+40),
			}
		}
	}
	ttlCnt, ttlOfs := readArr(base + 152)
	if int(ttlOfs)+int(ttlCnt)*2 <= len(data) {
		m.TextureTransformLookup = make([]int16, ttlCnt)
		for i := uint32(0); i < ttlCnt; i++ {
			m.TextureTransformLookup[i] = int16(binary.LittleEndian.Uint16(data[int(ttlOfs)+int(i)*2:]))
		}
	}

	// Texture combiner combos (base+304), present when the flag is set
	if flags&GlobalFlagTextureCombiners != 0 {
		ccCnt, ccOfs := readArr(base + 304)
		if ccCnt < 4096 && int(ccOfs)+int(ccCnt)*2 <= len(data) {
			m.TextureCombinerCombos = make([]uint16, ccCnt)
			for i := uint32(0); i < ccCnt; i++ {
				m.TextureCombinerCombos[i] = binary.LittleEndian.Uint16(data[int(ccOfs)+int(i)*2:])
			}
		}
	}

	// Attachments (base+240): 40-byte M2Attachment records
	atCnt, atOfs := readArr(base + 240)
	if int(atOfs)+int(atCnt)*40 <= len(data) {
		m.Attachments = make([]M2Attachment, atCnt)
		for i := uint32(0); i < atCnt; i++ {
			a := int(atOfs) + int(i)*40
			att := M2Attachment{
				ID:   binary.LittleEndian.Uint32(data[a : a+4]),
				Bone: binary.LittleEndian.Uint16(data[a+4 : a+6]),
			}
			for k := 0; k < 3; k++ {
				att.Position[k] = mathFloat32(data[a+8+k*4 : a+12+k*4])
			}
			m.Attachments[i] = att
		}
	}

	// Lights (base+264): 156-byte M2Light records
	lCnt, lOfs := readArr(base + 264)
	if int(lOfs)+int(lCnt)*156 <= len(data) {
		m.Lights = make([]M2Light, lCnt)
		for i := uint32(0); i < lCnt; i++ {
			l := int(lOfs) + int(i)*156
			light := M2Light{
				Type:             binary.LittleEndian.Uint16(data[l : l+2]),
				Bone:             int16(binary.LittleEndian.Uint16(data[l+2 : l+4])),
				AmbientColor:     readTrack(data, l+16),
				AmbientIntensity: readTrack(data, l+36),
				DiffuseColor:     readTrack(data, l+56),
				DiffuseIntensity: readTrack(data, l+76),
				AttenuationStart: readTrack(data, l+96),
				AttenuationEnd:   readTrack(data, l+116),
				Visibility:       readTrack(data, l+136),
			}
			for k := 0; k < 3; k++ {
				light.Position[k] = mathFloat32(data[l+4+k*4 : l+8+k*4])
			}
			m.Lights[i] = light
		}
	}

	// Cameras (base+272): 100-byte M2Camera records; Cataclysm (271+) turned
	// the FOV into a track and changed the record size
	camCnt, camOfs := readArr(base + 272)
	if version < 271 && int(camOfs)+int(camCnt)*100 <= len(data) {
		m.Cameras = make([]M2Camera, camCnt)
		for i := uint32(0); i < camCnt; i++ {
			c := int(camOfs) + int(i)*100
			cam := M2Camera{
				Type:     int32(binary.LittleEndian.Uint32(data[c : c+4])),
				FOV:      mathFloat32(data[c+4 : c+8]),
				FarClip:  mathFloat32(data[c+8 : c+12]),
				NearClip: mathFloat32(data[c+12 : c+16]),
				Position: readTrack(data, c+16),
				Target:   readTrack(data, c+48),
				Roll:     readTrack(data, c+80),
			}
			for k := 0; k < 3; k++ {
				cam.PosBase[k] = mathFloat32(data[c+36+k*4 : c+40+k*4])
				cam.TgtBase[k] = mathFloat32(data[c+68+k*4 : c+72+k*4])
			}
			m.Cameras[i] = cam
		}
	}

	// Particle emitters (base+296): 476-byte WotLK records, 492 bytes from
	// Cataclysm or when the header's new-particle flag (0x200) is set.
	peCnt, peOfs := readArr(base + 296)
	peStride := 476
	if version > 271 || flags&0x200 != 0 {
		peStride = 492
	}
	if int(peOfs)+int(peCnt)*peStride <= len(data) {
		m.Particles = make([]M2Particle, peCnt)
		for i := uint32(0); i < peCnt; i++ {
			m.Particles[i] = readParticle(data, int(peOfs)+int(i)*peStride)
		}
	}

	// Texture unit (texcoord) lookup table (base+136)
	tuCnt, tuOfs := readArr(base + 136)
	if int(tuOfs)+int(tuCnt)*2 <= len(data) {
		m.TexUnitLookup = make([]int16, tuCnt)
		for i := uint32(0); i < tuCnt; i++ {
			m.TexUnitLookup[i] = int16(binary.LittleEndian.Uint16(data[int(tuOfs)+int(i)*2:]))
		}
	}

	// Texture lookup table (base+128)
	tlCnt, tlOfs := readArr(base + 128)
	if int(tlOfs)+int(tlCnt)*2 <= len(data) {
		m.TextureLookup = make([]uint16, tlCnt)
		for i := uint32(0); i < tlCnt; i++ {
			m.TextureLookup[i] = binary.LittleEndian.Uint16(data[int(tlOfs)+int(i)*2:])
		}
	}

	return m, nil
}

// readParticle parses one WotLK M2Particle record starting at byte offset o.
func readParticle(data []byte, o int) M2Particle {
	p := M2Particle{
		ID:                  binary.LittleEndian.Uint32(data[o : o+4]),
		Flags:               binary.LittleEndian.Uint32(data[o+4 : o+8]),
		Bone:                binary.LittleEndian.Uint16(data[o+20 : o+22]),
		Texture:             binary.LittleEndian.Uint16(data[o+22 : o+24]),
		Blending:            data[o+40],
		EmitterType:         data[o+41],
		ColorIndex:          binary.LittleEndian.Uint16(data[o+42 : o+44]),
		ParticleTyp:         data[o+44],
		HeadOrTail:          data[o+45],
		TextureTileRotation: binary.LittleEndian.Uint16(data[o+46 : o+48]),
		TextureRows:         binary.LittleEndian.Uint16(data[o+48 : o+50]),
		TextureCols:         binary.LittleEndian.Uint16(data[o+50 : o+52]),

		EmissionSpeed:   readTrack(data, o+52),
		SpeedVariation:  readTrack(data, o+72),
		VerticalRange:   readTrack(data, o+92),
		HorizontalRange: readTrack(data, o+112),
		Gravity:         readTrack(data, o+132),
		Lifespan:        readTrack(data, o+152),
		LifespanVary:    mathFloat32(data[o+172 : o+176]),
		EmissionRate:    readTrack(data, o+176),
		EmissionRateVar: mathFloat32(data[o+196 : o+200]),
		AreaLength:      readTrack(data, o+200),
		AreaWidth:       readTrack(data, o+220),
		ZSource:         readTrack(data, o+240),

		ColorTrack: readPartTrack(data, o+260, 12),
		AlphaTrack: readPartTrack(data, o+276, 2),
		ScaleTrack: readPartTrack(data, o+292, 8),
		HeadCell:   readPartTrack(data, o+316, 2),
		TailCell:   readPartTrack(data, o+332, 2),

		TailLength:      mathFloat32(data[o+348 : o+352]),
		TwinkleSpeed:    mathFloat32(data[o+352 : o+356]),
		TwinklePercent:  mathFloat32(data[o+356 : o+360]),
		BurstMultiplier: mathFloat32(data[o+368 : o+372]),
		Drag:            mathFloat32(data[o+372 : o+376]),
		BaseSpin:        mathFloat32(data[o+376 : o+380]),
		BaseSpinVary:    mathFloat32(data[o+380 : o+384]),
		Spin:            mathFloat32(data[o+384 : o+388]),
		SpinVary:        mathFloat32(data[o+388 : o+392]),

		WindTime:     mathFloat32(data[o+428 : o+432]),
		FollowSpeed1: mathFloat32(data[o+432 : o+436]),
		FollowScale1: mathFloat32(data[o+436 : o+440]),
		FollowSpeed2: mathFloat32(data[o+440 : o+444]),
		FollowScale2: mathFloat32(data[o+444 : o+448]),

		EnabledIn: readTrack(data, o+456),
	}

	for k := 0; k < 3; k++ {
		p.Position[k] = mathFloat32(data[o+8+k*4 : o+12+k*4])
		p.Tumble[0][k] = mathFloat32(data[o+392+k*4 : o+396+k*4])
		p.Tumble[1][k] = mathFloat32(data[o+404+k*4 : o+408+k*4])
		p.WindVector[k] = mathFloat32(data[o+416+k*4 : o+420+k*4])
	}
	for k := 0; k < 2; k++ {
		p.ScaleVary[k] = mathFloat32(data[o+308+k*4 : o+312+k*4])
		p.TwinkleScale[k] = mathFloat32(data[o+360+k*4 : o+364+k*4])
	}

	return p
}

// readPartTrack parses an M2PartTrack (a flat fixed16 timestamp array plus a
// flat value array) whose element size is elemSize.
func readPartTrack(data []byte, off, elemSize int) M2PartTrack {
	pt := M2PartTrack{ElemSize: elemSize}

	tCnt, tOfs := readArrAt(data, off)
	if tCnt < 4096 && int(tOfs)+int(tCnt)*2 <= len(data) {
		pt.Times = make([]int16, tCnt)
		for i := uint32(0); i < tCnt; i++ {
			pt.Times[i] = int16(binary.LittleEndian.Uint16(data[int(tOfs)+int(i)*2:]))
		}
	}

	vCnt, vOfs := readArrAt(data, off+8)
	if vCnt < 4096 && int(vOfs)+int(vCnt)*elemSize <= len(data) {
		pt.Values = data[int(vOfs) : int(vOfs)+int(vCnt)*elemSize]
	}

	return pt
}

// readArrAt reads an M2Array header (count, offset) at the given byte offset.
func readArrAt(data []byte, ofs int) (uint32, uint32) {
	if ofs+8 > len(data) {
		return 0, 0
	}
	return binary.LittleEndian.Uint32(data[ofs : ofs+4]), binary.LittleEndian.Uint32(data[ofs+4 : ofs+8])
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

// FirstKey returns the raw bytes of the first keyframe value of a track in
// the given sequence, or nil. ext is the sequence's external .anim contents.
func (m *Model) FirstKey(t M2Track, seq int, elemSize int, ext []byte) []byte {
	_, vals := m.TrackKeys(t, seq, elemSize, ext)
	if len(vals) < elemSize {
		return nil
	}
	return vals[:elemSize]
}

// SplineKeyValue returns the value part of the first M2SplineKey<C3Vector>
// of a track in the given sequence (the key's in/out tangents are ignored).
// ext is the sequence's external .anim contents, or nil when in-file.
func (m *Model) SplineKeyValue(t M2Track, seq int, ext []byte) ([3]float32, bool) {
	if seq < 0 || seq >= len(t.Values) || t.Values[seq].Count == 0 {
		return [3]float32{}, false
	}
	src := m.raw
	if ext != nil {
		src = ext
	}
	ofs := int(t.Values[seq].Offset)
	if ofs+12 > len(src) {
		return [3]float32{}, false
	}
	return [3]float32{mathFloat32(src[ofs : ofs+4]), mathFloat32(src[ofs+4 : ofs+8]), mathFloat32(src[ofs+8 : ofs+12])}, true
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
