package m2

import (
	"encoding/binary"
	"fmt"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

// DefaultSequences lists the AnimationData.dbc IDs exported by default, with
// the glTF clip names the client plays them by.
var DefaultSequences = map[uint16]string{
	0:  "Stand",
	1:  "Death",
	4:  "Walk",
	5:  "Run",
	6:  "Dead",
	13: "WalkBackwards",
	16: "AttackUnarmed",
	17: "Attack1H",
	18: "Attack2H",
	25: "ReadyUnarmed",
	26: "Ready1H",
	27: "Ready2H",
	37: "JumpStart",
	38: "Jump",
	39: "JumpEnd",
	40: "Fall",
	41: "SwimIdle",
	42: "Swim",
}

// ConvertOptions tunes ConvertToGLTF.
type ConvertOptions struct {
	// Sequences maps AnimationData IDs to clip names. Nil means DefaultSequences.
	Sequences map[uint16]string
	// AnimLoader fetches external .anim files. Nil skips sequences stored externally.
	AnimLoader AnimLoader
}

// Option customises the conversion.
type Option func(*ConvertOptions)

// WithSequences selects which animation sequences are exported.
func WithSequences(seqs map[uint16]string) Option {
	return func(o *ConvertOptions) { o.Sequences = seqs }
}

// WithAnimLoader enables sequences stored in external .anim files.
func WithAnimLoader(l AnimLoader) Option {
	return func(o *ConvertOptions) { o.AnimLoader = l }
}

// AnimationExtras is attached to each exported glTF animation.
type AnimationExtras struct {
	SequenceID uint16  `json:"sequenceId"`
	MoveSpeed  float32 `json:"moveSpeed"` // yards/s the clip was authored for, 0 if stationary
	DurationMs uint32  `json:"durationMs"`
}

// boneChannel is one resolved TRS track of a bone for a sequence.
type boneChannel struct {
	path          string
	interpolation string
	times         []float32
	vec3          [][3]float32
	vec4          [][4]float32
}

// addAnimations appends one glTF animation per selected sequence.
func addAnimations(doc *gltf.Document, model *Model, jointNodes []int, restTranslation [][3]float32, opts ConvertOptions) {
	seqs := opts.Sequences
	if seqs == nil {
		seqs = DefaultSequences
	}

	for seqIdx, seq := range model.Sequences {
		name, wanted := seqs[seq.ID]
		if !wanted || seq.VariationIndex != 0 {
			continue
		}

		dataIdx, ext, ok := resolveSequenceData(model, seqIdx, opts.AnimLoader)
		if !ok {
			continue
		}

		// AnimationExtras carries sequence metadata the client uses to match
		// clip playback rate to the entity's actual velocity.
		anim := gltf.Animation{Name: name, Extras: AnimationExtras{
			SequenceID: seq.ID,
			MoveSpeed:  seq.MoveSpeed,
			DurationMs: seq.Duration,
		}}
		for bIdx, bone := range model.Bones {
			for _, ch := range boneChannels(model, bone, dataIdx, ext, restTranslation[bIdx]) {
				sampler := gltf.AnimationSampler{
					Input:         doc.AddFloat32ScalarAccessor(ch.times),
					Interpolation: ch.interpolation,
				}
				if ch.vec4 != nil {
					sampler.Output = doc.AddFloat32Vec4Accessor(ch.vec4, 0)
				} else {
					sampler.Output = doc.AddFloat32Vec3Accessor(ch.vec3, 0)
				}
				anim.Samplers = append(anim.Samplers, sampler)
				anim.Channels = append(anim.Channels, gltf.AnimationChannel{
					Sampler: len(anim.Samplers) - 1,
					Target:  gltf.AnimationChannelTarget{Node: jointNodes[bIdx], Path: ch.path},
				})
			}
		}

		if len(anim.Channels) > 0 {
			doc.Animations = append(doc.Animations, anim)
		}
	}

	// Tracks driven by a global sequence loop independently of the played
	// sequence (waving banners, drifting clouds): one always-on clip each.
	for gs, duration := range model.GlobalSequences {
		anim := gltf.Animation{Name: fmt.Sprintf("%s%d", GlobalSequencePrefix, gs), Extras: AnimationExtras{
			SequenceID: 0xFFFF,
			DurationMs: duration,
		}}
		for bIdx, bone := range model.Bones {
			for _, ch := range boneChannelsFor(model, bone, 0, nil, restTranslation[bIdx], gs) {
				sampler := gltf.AnimationSampler{
					Input:         doc.AddFloat32ScalarAccessor(ch.times),
					Interpolation: ch.interpolation,
				}
				if ch.vec4 != nil {
					sampler.Output = doc.AddFloat32Vec4Accessor(ch.vec4, 0)
				} else {
					sampler.Output = doc.AddFloat32Vec3Accessor(ch.vec3, 0)
				}
				anim.Samplers = append(anim.Samplers, sampler)
				anim.Channels = append(anim.Channels, gltf.AnimationChannel{
					Sampler: len(anim.Samplers) - 1,
					Target:  gltf.AnimationChannelTarget{Node: jointNodes[bIdx], Path: ch.path},
				})
			}
		}
		if len(anim.Channels) > 0 {
			doc.Animations = append(doc.Animations, anim)
		}
	}
}

// GlobalSequencePrefix names the always-looping global sequence clips.
const GlobalSequencePrefix = "GlobalSequence_"

// resolveSequenceData follows alias links and loads external data when needed.
// It returns the sequence index holding the keyframes and the .anim bytes (nil = in-file).
func resolveSequenceData(model *Model, seqIdx int, loader AnimLoader) (int, []byte, bool) {
	for hops := 0; hops < 16; hops++ {
		seq := model.Sequences[seqIdx]
		if seq.Flags&SequenceFlagAlias == 0 {
			break
		}
		if int(seq.NextAlias) >= len(model.Sequences) || int(seq.NextAlias) == seqIdx {
			return 0, nil, false
		}
		seqIdx = int(seq.NextAlias)
	}

	seq := model.Sequences[seqIdx]
	if seq.Flags&SequenceFlagInFile != 0 {
		return seqIdx, nil, true
	}
	if loader == nil {
		return 0, nil, false
	}
	ext, err := loader(seq.ID, seq.VariationIndex)
	if err != nil || len(ext) == 0 {
		return 0, nil, false
	}
	return seqIdx, ext, true
}

// boneChannels converts a bone's tracks for one sequence into glTF channels.
//
// M2 composes a bone as parent * T(pivot) * T(t) * R * S * T(-pivot) with the
// pivot absolute. The joint node already carries T(pivot - parentPivot) as its
// rest translation, so the animated node transform is
// translation = (pivot - parentPivot) + t, rotation = R, scale = S.
func boneChannels(model *Model, bone M2Bone, seq int, ext []byte, rest [3]float32) []boneChannel {
	return boneChannelsFor(model, bone, seq, ext, rest, -1)
}

// boneChannelsFor extracts the channels of a bone that are driven either by
// the sequence (gs < 0) or by global sequence gs, whose keys live at index 0.
func boneChannelsFor(model *Model, bone M2Bone, seq int, ext []byte, rest [3]float32, gs int) []boneChannel {
	var out []boneChannel
	if gs >= 0 {
		seq, ext = 0, nil
	}
	driven := func(t M2Track) bool { return int(t.GlobalSequence) == gs || (gs < 0 && t.GlobalSequence < 0) }

	if ch, ok := vec3Channel(model, bone.Translation, seq, ext); ok && driven(bone.Translation) {
		for i := range ch.vec3 {
			v := gltf.ConvertM2ToGLTPosition(ch.vec3[i][0], ch.vec3[i][1], ch.vec3[i][2])
			ch.vec3[i] = [3]float32{rest[0] + v[0], rest[1] + v[1], rest[2] + v[2]}
		}
		ch.path = "translation"
		out = append(out, ch)
	}

	if times, raw := model.TrackKeys(bone.Rotation, seq, 8, ext); len(times) > 0 && driven(bone.Rotation) {
		stride := 8
		if bone.Rotation.Interpolation >= 2 {
			stride *= 3
		}
		ch := boneChannel{path: "rotation", interpolation: interpolationName(bone.Rotation.Interpolation)}
		for i, t := range times {
			if i > 0 && t <= times[i-1] {
				continue // glTF requires strictly increasing input
			}
			q := decodeCompQuat(raw[i*stride:])
			// Same axis permutation as positions: (x, y, z) -> (-y, z, -x)
			ch.times = append(ch.times, float32(t)/1000)
			ch.vec4 = append(ch.vec4, [4]float32{-q[1], q[2], -q[0], q[3]})
		}
		if len(ch.times) > 0 {
			out = append(out, ch)
		}
	}

	if ch, ok := vec3Channel(model, bone.Scale, seq, ext); ok && driven(bone.Scale) {
		for i := range ch.vec3 {
			s := ch.vec3[i]
			ch.vec3[i] = [3]float32{s[1], s[2], s[0]}
		}
		ch.path = "scale"
		out = append(out, ch)
	}

	return out
}

func vec3Channel(model *Model, track M2Track, seq int, ext []byte) (boneChannel, bool) {
	times, raw := model.TrackKeys(track, seq, 12, ext)
	if len(times) == 0 {
		return boneChannel{}, false
	}
	stride := 12
	if track.Interpolation >= 2 {
		stride *= 3
	}
	ch := boneChannel{interpolation: interpolationName(track.Interpolation)}
	for i, t := range times {
		if i > 0 && t <= times[i-1] {
			continue
		}
		o := i * stride
		ch.times = append(ch.times, float32(t)/1000)
		ch.vec3 = append(ch.vec3, [3]float32{
			mathFloat32(raw[o : o+4]),
			mathFloat32(raw[o+4 : o+8]),
			mathFloat32(raw[o+8 : o+12]),
		})
	}
	return ch, len(ch.times) > 0
}

// decodeCompQuat unpacks an M2CompQuat (4 x int16, x y z w).
func decodeCompQuat(b []byte) [4]float32 {
	var q [4]float32
	for i := 0; i < 4; i++ {
		v := int16(binary.LittleEndian.Uint16(b[i*2:]))
		if v < 0 {
			q[i] = float32(int32(v)+32768) / 32767
		} else {
			q[i] = float32(int32(v)-32767) / 32767
		}
	}
	return q
}

func interpolationName(interp uint16) string {
	if interp == 0 {
		return "STEP"
	}
	// Hermite/bezier tangents are dropped; linear is a close enough fit.
	return "LINEAR"
}
