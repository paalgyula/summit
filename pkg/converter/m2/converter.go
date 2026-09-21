package m2

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

// ConvertToGLTF converts an M2 model (and optional .skin file) into a glTF Document.
func ConvertToGLTF(model *Model, skin *Skin, options ...Option) (*gltf.Document, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}
	var opts ConvertOptions
	for _, o := range options {
		o(&opts)
	}

	doc := gltf.NewDocument()

	// Convert vertices
	positions := make([][3]float32, len(model.Vertices))
	normals := make([][3]float32, len(model.Vertices))
	uvs := make([][2]float32, len(model.Vertices))
	uvs2 := make([][2]float32, len(model.Vertices))
	joints := make([][4]uint16, len(model.Vertices))
	weights := make([][4]float32, len(model.Vertices))

	for i, v := range model.Vertices {
		positions[i] = gltf.ConvertM2ToGLTPosition(v.Pos[0], v.Pos[1], v.Pos[2])
		normals[i] = gltf.ConvertM2ToGLTPosition(v.Normal[0], v.Normal[1], v.Normal[2])
		uvs[i] = [2]float32{v.TexCoords[0], v.TexCoords[1]}
		uvs2[i] = [2]float32{v.TexCoords2[0], v.TexCoords2[1]}

		joints[i] = [4]uint16{
			uint16(v.BoneIndices[0]),
			uint16(v.BoneIndices[1]),
			uint16(v.BoneIndices[2]),
			uint16(v.BoneIndices[3]),
		}

		totalW := float32(v.BoneWeights[0]) + float32(v.BoneWeights[1]) + float32(v.BoneWeights[2]) + float32(v.BoneWeights[3])
		if totalW > 0 {
			weights[i] = [4]float32{
				float32(v.BoneWeights[0]) / totalW,
				float32(v.BoneWeights[1]) / totalW,
				float32(v.BoneWeights[2]) / totalW,
				float32(v.BoneWeights[3]) / totalW,
			}
		} else {
			weights[i] = [4]float32{1.0, 0, 0, 0}
		}
	}

	posAcc := doc.AddFloat32Vec3Accessor(positions, gltf.TargetArrayBuffer)
	normAcc := doc.AddFloat32Vec3Accessor(normals, gltf.TargetArrayBuffer)
	uvAcc := doc.AddFloat32Vec2Accessor(uvs, gltf.TargetArrayBuffer)
	uv2Acc := doc.AddFloat32Vec2Accessor(uvs2, gltf.TargetArrayBuffer)
	jAcc, wAcc := doc.AddJointsWeightsAccessor(joints, weights)

	attrMap := map[string]int{
		"POSITION":   posAcc,
		"NORMAL":     normAcc,
		"TEXCOORD_0": uvAcc,
		"TEXCOORD_1": uv2Acc,
	}

	if len(model.Bones) > 0 {
		attrMap["JOINTS_0"] = jAcc
		attrMap["WEIGHTS_0"] = wAcc
	}

	var primitives []gltf.Primitive

	if skin != nil && len(skin.Submeshes) > 0 {
		for subIdx, sub := range skin.Submeshes {
			var subIndices []uint16
			// Level carries the high bits of TriangleStart for large models
			triStart := int(sub.TriangleStart) + int(sub.Level)<<16
			triCount := int(sub.TriangleCount)

			if triStart+triCount <= len(skin.Triangles) {
				for t := 0; t < triCount; t++ {
					// triangles index the skin's global vertex lookup table,
					// which in turn maps to M2 vertices
					lookupIdx := int(skin.Triangles[triStart+t])
					if lookupIdx < len(skin.Indices) {
						subIndices = append(subIndices, skin.Indices[lookupIdx])
					}
				}
			}

			if len(subIndices) == 0 {
				continue
			}

			idxAcc := doc.AddUint16IndicesAccessor(subIndices)

			// One primitive per texture unit (render pass) of the submesh; a
			// unit's own textures are combined by its pixel shader
			for _, mat := range submeshMaterials(model, skin, subIdx, sub, opts) {
				matIdx := len(doc.Materials)
				doc.Materials = append(doc.Materials, mat)
				primitives = append(primitives, gltf.Primitive{
					Attributes: attrMap,
					Indices:    &idxAcc,
					Material:   &matIdx,
				})
			}
		}
	}

	// Fallback if no submeshes or no skin file: sequential triangle index generator
	if len(primitives) == 0 && len(positions) >= 3 {
		triCount := (len(positions) / 3) * 3
		seqIndices := make([]uint16, triCount)
		for i := 0; i < triCount; i++ {
			seqIndices[i] = uint16(i)
		}
		idxAcc := doc.AddUint16IndicesAccessor(seqIndices)
		matIdx := len(doc.Materials)
		doc.Materials = append(doc.Materials, gltf.Material{
			Name: "DefaultMaterial",
			PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
				BaseColorFactor: [4]float32{0.8, 0.8, 0.8, 1.0},
				MetallicFactor:  0.0,
				RoughnessFactor: 0.8,
			},
			DoubleSided: true,
		})
		primitives = append(primitives, gltf.Primitive{
			Attributes: attrMap,
			Indices:    &idxAcc,
			Material:   &matIdx,
		})
	}

	meshIdx := len(doc.Meshes)
	meshName := model.Name
	if meshName == "" {
		meshName = "WoW_Model"
	}
	doc.Meshes = append(doc.Meshes, gltf.Mesh{
		Name:       meshName,
		Primitives: primitives,
	})

	// Add joint nodes for bones. M2 vertices are already in model space and
	// bone pivots are absolute, so each node's translation is relative to its
	// parent pivot and the inverse bind matrix undoes the absolute pivot; the
	// composed bind pose is then the identity.
	jointNodeIndices := make([]int, len(model.Bones))
	inverseBind := make([][16]float32, len(model.Bones))
	restTranslation := make([][3]float32, len(model.Bones))
	var rootBones []int
	for bIdx, bone := range model.Bones {
		jNodeIdx := len(doc.Nodes)
		jointNodeIndices[bIdx] = jNodeIdx
		pivot := gltf.ConvertM2ToGLTPosition(bone.Pivot[0], bone.Pivot[1], bone.Pivot[2])

		translation := pivot
		if bone.ParentBone >= 0 && int(bone.ParentBone) < len(model.Bones) {
			pp := model.Bones[bone.ParentBone].Pivot
			parentPivot := gltf.ConvertM2ToGLTPosition(pp[0], pp[1], pp[2])
			for i := 0; i < 3; i++ {
				translation[i] -= parentPivot[i]
			}
		} else {
			rootBones = append(rootBones, jNodeIdx)
		}

		restTranslation[bIdx] = translation

		// Column-major translate(-pivot)
		inverseBind[bIdx] = [16]float32{
			1, 0, 0, 0,
			0, 1, 0, 0,
			0, 0, 1, 0,
			-pivot[0], -pivot[1], -pivot[2], 1,
		}

		doc.Nodes = append(doc.Nodes, gltf.Node{
			Name:        fmt.Sprintf("Bone_%d_Key_%d", bIdx, bone.KeyBoneID),
			Translation: &translation,
		})
	}

	// Link bone parent-child hierarchy
	for bIdx, bone := range model.Bones {
		if bone.ParentBone >= 0 && int(bone.ParentBone) < len(jointNodeIndices) {
			parentIdx := jointNodeIndices[bone.ParentBone]
			doc.Nodes[parentIdx].Children = append(doc.Nodes[parentIdx].Children, jointNodeIndices[bIdx])
		}
	}

	var skinIdx *int
	if len(jointNodeIndices) > 0 {
		sIdx := len(doc.Skins)
		skinIdx = &sIdx
		ibmAcc := doc.AddInverseBindMatrices(inverseBind)
		skin := gltf.Skin{
			Name:                "Armature",
			Joints:              jointNodeIndices,
			InverseBindMatrices: &ibmAcc,
		}
		if len(rootBones) > 0 {
			skin.Skeleton = &rootBones[0]
		}
		doc.Skins = append(doc.Skins, skin)
	}

	// Root mesh node. The skeleton roots hang off it so the joints are part of
	// the scene graph and inherit the node's transform; otherwise loaders leave
	// the bones at the world origin and the skinned mesh is drawn there.
	rootNodeIdx := len(doc.Nodes)
	doc.Nodes = append(doc.Nodes, gltf.Node{
		Name:     meshName + "_Root",
		Mesh:     &meshIdx,
		Skin:     skinIdx,
		Children: rootBones,
	})

	doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, rootNodeIdx)

	if len(model.Bones) > 0 {
		addAnimations(doc, model, jointNodeIndices, restTranslation, opts)
	}

	addCameras(doc, model, opts)
	addLights(doc, model, opts)

	return doc, nil
}

// LightExtras is one M2 light, static at its first keyframe.
type LightExtras struct {
	Type     uint16     `json:"type"` // 0 directional, 1 point
	Position [3]float32 `json:"position"`
	Ambient  [3]float32 `json:"ambient"`
	Diffuse  [3]float32 `json:"diffuse"`
	// AttenuationEnd is the point light's range (0 for directional).
	AttenuationEnd float32 `json:"attenuationEnd,omitempty"`
}

// SceneExtras is attached to the glTF scene.
type SceneExtras struct {
	Lights []LightExtras `json:"lights,omitempty"`
}

// addLights writes the model's lights into the scene extras: glue scenes
// (login, character select) light their pre-baked artwork with them.
func addLights(doc *gltf.Document, model *Model, opts ConvertOptions) {
	if len(model.Lights) == 0 {
		return
	}
	ext := sequenceExt(model, opts)
	var lights []LightExtras
	for _, l := range model.Lights {
		le := LightExtras{Type: l.Type, Position: gltf.ConvertM2ToGLTPosition(l.Position[0], l.Position[1], l.Position[2])}
		amb, ai := firstVec3(model, l.AmbientColor, ext, [3]float32{1, 1, 1}), firstFloat(model, l.AmbientIntensity, ext, 1)
		dif, di := firstVec3(model, l.DiffuseColor, ext, [3]float32{1, 1, 1}), firstFloat(model, l.DiffuseIntensity, ext, 1)
		for k := 0; k < 3; k++ {
			le.Ambient[k] = amb[k] * ai
			le.Diffuse[k] = dif[k] * di
		}
		le.AttenuationEnd = firstFloat(model, l.AttenuationEnd, ext, 0)
		lights = append(lights, le)
	}
	doc.Scenes[0].Extras = SceneExtras{Lights: lights}
}

// sequenceExt returns the external .anim data of sequence 0, if any.
func sequenceExt(model *Model, opts ConvertOptions) []byte {
	if len(model.Sequences) == 0 {
		return nil
	}
	if _, data, ok := resolveSequenceData(model, 0, opts.AnimLoader); ok {
		return data
	}
	return nil
}

func firstVec3(model *Model, t M2Track, ext []byte, def [3]float32) [3]float32 {
	key := model.FirstKey(t, 0, 12, ext)
	if key == nil {
		return def
	}
	return [3]float32{mathFloat32(key[0:4]), mathFloat32(key[4:8]), mathFloat32(key[8:12])}
}

func firstFloat(model *Model, t M2Track, ext []byte, def float32) float32 {
	key := model.FirstKey(t, 0, 4, ext)
	if key == nil {
		return def
	}
	return mathFloat32(key)
}

// CameraExtras is attached to every camera node.
type CameraExtras struct {
	// Type is the M2 camera type (-1 portrait, 0 character info, 1+ flyby).
	Type int32 `json:"type"`
	// DiagonalFOV is the M2 field of view (diagonal, radians); the client
	// derives the vertical FOV from it and its aspect ratio.
	DiagonalFOV float32 `json:"diagonalFov"`
	// Target is the look-at point in glTF space.
	Target [3]float32 `json:"target"`
}

// addCameras emits the model's cameras as glTF cameras at their first
// keyframe (sequence 0), looking at their target: the login and character
// screen scenes define their view this way.
func addCameras(doc *gltf.Document, model *Model, opts ConvertOptions) {
	var ext []byte
	if len(model.Sequences) > 0 {
		if _, data, ok := resolveSequenceData(model, 0, opts.AnimLoader); ok {
			ext = data
		}
	}

	for i, cam := range model.Cameras {
		pos := cam.PosBase
		if key, ok := model.SplineKeyValue(cam.Position, 0, ext); ok {
			for k := range pos {
				pos[k] += key[k]
			}
		}
		tgt := cam.TgtBase
		if key, ok := model.SplineKeyValue(cam.Target, 0, ext); ok {
			for k := range tgt {
				tgt[k] += key[k]
			}
		}

		position := gltf.ConvertM2ToGLTPosition(pos[0], pos[1], pos[2])
		target := gltf.ConvertM2ToGLTPosition(tgt[0], tgt[1], tgt[2])

		// Vertical FOV for a 4:3 view; the client recomputes from the diagonal
		aspect := 4.0 / 3.0
		yfov := 2 * math.Atan(math.Tan(float64(cam.FOV)/2)/math.Sqrt(1+aspect*aspect))
		if math.IsNaN(yfov) || yfov <= 0 || yfov >= math.Pi {
			yfov = math.Pi / 3
		}

		camIdx := len(doc.Cameras)
		doc.Cameras = append(doc.Cameras, gltf.Camera{
			Name: fmt.Sprintf("Camera_%d", i),
			Type: "perspective",
			Perspective: &gltf.Perspective{
				YFov:  float32(yfov),
				ZNear: max(cam.NearClip, 0.01),
				ZFar:  max(cam.FarClip, 1),
			},
		})

		rotation := gltf.LookAtRotation(position, target)
		nodeIdx := len(doc.Nodes)
		doc.Nodes = append(doc.Nodes, gltf.Node{
			Name:        fmt.Sprintf("Camera_%d", i),
			Camera:      &camIdx,
			Translation: &position,
			Rotation:    &rotation,
			Extras:      CameraExtras{Type: cam.Type, DiagonalFOV: cam.FOV, Target: target},
		})
		doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)
	}
}

// LayerExtras is one texture of a render pass.
type LayerExtras struct {
	// Texture is the asset-server path (WebP) when the model names it; empty
	// for runtime-replaced textures (skins, hair).
	Texture     string `json:"texture,omitempty"`
	TextureType uint32 `json:"textureType"`
	// TexCoord selects the UV set: 0, 1, or -1 for spherical environment mapping.
	TexCoord int `json:"texCoord"`
	// Wrap: the texture repeats (M2 texture flags); clamps otherwise.
	Wrap bool `json:"wrap,omitempty"`
	// UVKeys scroll the texture: [time ms, u, v] over UVLoopMs.
	UVKeys   [][3]float32 `json:"uvKeys,omitempty"`
	UVLoopMs uint32       `json:"uvLoopMs,omitempty"`
}

// MaterialExtras describes the WoW render state of one render pass (texture
// unit) of a submesh so the client can reproduce the fixed-function combiner.
type MaterialExtras struct {
	// Texture / TextureType / UVKeys / UVLoopMs mirror Layers[0] for clients
	// that only draw the first texture.
	Texture     string       `json:"texture,omitempty"`
	TextureType uint32       `json:"textureType"`
	UVKeys      [][3]float32 `json:"uvKeys,omitempty"`
	UVLoopMs    uint32       `json:"uvLoopMs,omitempty"`
	// Layers are the pass's textures, combined by PixelShader.
	Layers []LayerExtras `json:"layers"`
	// PixelShader names the combiner (Combiners_Opaque, Combiners_Mod_Add, ...).
	PixelShader  string `json:"pixelShader"`
	BlendMode    uint16 `json:"blendMode"`
	Unlit        bool   `json:"unlit,omitempty"`
	Unfogged     bool   `json:"unfogged,omitempty"`
	TwoSided     bool   `json:"twoSided,omitempty"`
	NoDepthTest  bool   `json:"noDepthTest,omitempty"`
	NoDepthWrite bool   `json:"noDepthWrite,omitempty"`
	// Layer is the pass index within the submesh (draw order), Priority the
	// unit's priority plane.
	Layer    uint16 `json:"layer"`
	Priority int8   `json:"priority,omitempty"`
	// ColorKeys / AlphaKeys animate the pass over sequence 0 (or its global
	// loop): [time ms, r, g, b] and [time ms, alpha]. The static value in the
	// material is the first key.
	ColorKeys [][4]float32 `json:"colorKeys,omitempty"`
	AlphaKeys [][2]float32 `json:"alphaKeys,omitempty"`
	// LoopMs is the length of the timeline the keys are on.
	LoopMs uint32 `json:"loopMs,omitempty"`
}

// submeshMaterials builds one glTF material per texture unit targeting the
// submesh, in draw order.
func submeshMaterials(model *Model, skin *Skin, subIdx int, sub Submesh, opts ConvertOptions) []gltf.Material {
	var units []*TextureUnit
	for i := range skin.TextureUnits {
		if int(skin.TextureUnits[i].SubmeshIndex) == subIdx {
			units = append(units, &skin.TextureUnits[i])
		}
	}
	sort.SliceStable(units, func(i, j int) bool { return units[i].MaterialLayer < units[j].MaterialLayer })

	if len(units) == 0 {
		return []gltf.Material{{
			Name: fmt.Sprintf("Submesh_%d_Geoset_%d", subIdx, sub.ID),
			PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
				BaseColorFactor: [4]float32{1, 1, 1, 1},
				MetallicFactor:  0.0,
				RoughnessFactor: 1.0,
			},
			DoubleSided: true,
		}}
	}

	ext := sequenceExt(model, opts)
	mats := make([]gltf.Material, 0, len(units))
	for _, unit := range units {
		mats = append(mats, unitMaterial(model, unit, subIdx, sub, ext))
	}
	return mats
}

// unitMaterial builds the glTF material of one texture unit.
func unitMaterial(model *Model, unit *TextureUnit, subIdx int, sub Submesh, ext []byte) gltf.Material {
	mat := gltf.Material{
		Name: fmt.Sprintf("Submesh_%d_Geoset_%d", subIdx, sub.ID),
		PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
			BaseColorFactor: [4]float32{1, 1, 1, 1},
			MetallicFactor:  0.0,
			RoughnessFactor: 1.0,
		},
		DoubleSided: true,
	}
	if unit.MaterialLayer > 0 {
		mat.Name = fmt.Sprintf("Submesh_%d_Geoset_%d_Pass_%d", subIdx, sub.ID, unit.MaterialLayer)
	}

	extras := MaterialExtras{Layer: unit.MaterialLayer, Priority: unit.PriorityPlane}
	if int(unit.MaterialIndex) < len(model.Materials) {
		m := model.Materials[unit.MaterialIndex]
		extras.BlendMode = m.BlendMode
		extras.Unlit = m.Flags&MaterialFlagUnlit != 0
		extras.Unfogged = m.Flags&MaterialFlagUnfogged != 0
		extras.TwoSided = m.Flags&MaterialFlagTwoSided != 0
		extras.NoDepthTest = m.Flags&MaterialFlagDepthTest != 0
		extras.NoDepthWrite = m.Flags&MaterialFlagNoDepthWr != 0
		mat.DoubleSided = extras.TwoSided
		switch m.BlendMode {
		case BlendOpaque:
		case BlendAlphaKey:
			cutoff := float32(0.5)
			mat.AlphaMode = gltf.AlphaModeMask
			mat.AlphaCutoff = &cutoff
		default:
			mat.AlphaMode = gltf.AlphaModeBlend
		}
	}

	// Colour and opacity of the pass from the colour/alpha and transparency
	// animations: the first keyframe is the static value, the rest the timeline
	color := [4]float32{1, 1, 1, 1}
	if int(unit.ColorIndex) < len(model.Colors) {
		mc := model.Colors[unit.ColorIndex]
		if times, vals := model.TrackKeys(mc.Color, 0, 12, ext); len(times) > 0 {
			color[0], color[1], color[2] = mathFloat32(vals[0:4]), mathFloat32(vals[4:8]), mathFloat32(vals[8:12])
			if len(times) > 1 {
				stride := len(vals) / len(times)
				for i, t := range times {
					v := vals[i*stride:]
					extras.ColorKeys = append(extras.ColorKeys, [4]float32{float32(t), mathFloat32(v[0:4]), mathFloat32(v[4:8]), mathFloat32(v[8:12])})
				}
				extras.LoopMs = trackLoop(model, mc.Color)
			}
		}
		if times, vals := model.TrackKeys(mc.Alpha, 0, 2, ext); len(times) > 0 {
			color[3] *= fixed16(vals)
			if len(times) > 1 {
				stride := len(vals) / len(times)
				for i, t := range times {
					extras.AlphaKeys = append(extras.AlphaKeys, [2]float32{float32(t), fixed16(vals[i*stride:])})
				}
				extras.LoopMs = trackLoop(model, mc.Alpha)
			}
		}
	}
	if int(unit.WeightComboIndex) < len(model.TransparencyLookup) {
		if wi := int(model.TransparencyLookup[unit.WeightComboIndex]); wi < len(model.TextureWeights) {
			tw := model.TextureWeights[wi]
			if times, vals := model.TrackKeys(tw, 0, 2, ext); len(times) > 0 {
				color[3] *= fixed16(vals)
				if len(times) > 1 && extras.AlphaKeys == nil {
					stride := len(vals) / len(times)
					for i, t := range times {
						extras.AlphaKeys = append(extras.AlphaKeys, [2]float32{float32(t), fixed16(vals[i*stride:])})
					}
					extras.LoopMs = trackLoop(model, tw)
				}
			}
		}
	}

	// The pass's textures: texture, UV set and scroll animation each
	count := int(unit.TextureCount)
	if count < 1 {
		count = 1
	}
	if count > 2 {
		count = 2 // WotLK combines at most two textures per pass
	}
	for i := 0; i < count; i++ {
		layer := LayerExtras{}
		if ti := int(unit.TextureComboIndex) + i; ti < len(model.TextureLookup) {
			if tex := int(model.TextureLookup[ti]); tex < len(model.Textures) {
				t := model.Textures[tex]
				layer.TextureType = t.Type
				layer.Wrap = t.Flags&(TextureFlagWrapX|TextureFlagWrapY) != 0
				if t.Type == TextureTypeFilename {
					layer.Texture = TextureAssetPath(t.Name)
				}
			}
		}
		if ci := int(unit.TexCoordComboIndex) + i; ci < len(model.TexUnitLookup) {
			layer.TexCoord = int(model.TexUnitLookup[ci])
			if layer.TexCoord > 1 {
				layer.TexCoord = 0
			}
		}
		if xi := int(unit.TransformCombo) + i; xi < len(model.TextureTransformLookup) {
			if ti := int(model.TextureTransformLookup[xi]); ti >= 0 && ti < len(model.TextureTransforms) {
				tt := model.TextureTransforms[ti]
				if times, vals := model.TrackKeys(tt.Translation, 0, 12, ext); len(times) > 1 {
					stride := len(vals) / len(times)
					for k, t := range times {
						v := vals[k*stride:]
						layer.UVKeys = append(layer.UVKeys, [3]float32{float32(t), mathFloat32(v[0:4]), mathFloat32(v[4:8])})
					}
					layer.UVLoopMs = trackLoop(model, tt.Translation)
				}
			}
		}
		extras.Layers = append(extras.Layers, layer)
	}
	extras.Texture = extras.Layers[0].Texture
	extras.TextureType = extras.Layers[0].TextureType
	extras.UVKeys = extras.Layers[0].UVKeys
	extras.UVLoopMs = extras.Layers[0].UVLoopMs
	extras.PixelShader = PixelShaderName(count, unitShaderID(model, unit, extras.BlendMode))

	mat.PbrMetallicRoughness.BaseColorFactor = color
	if (color[3] < 1 || extras.AlphaKeys != nil) && mat.AlphaMode == "" {
		mat.AlphaMode = gltf.AlphaModeBlend
	}
	mat.Extras = extras
	return mat
}

// unitShaderID resolves a batch's combiner ops the way the WotLK client
// does: from the texture_combiner_combos table when the model has one,
// otherwise the first texture is Opaque for opaque materials and Mod (so the
// texture alpha reaches the alpha test / blend) for every other blend mode,
// and a second texture is Mod.
func unitShaderID(model *Model, unit *TextureUnit, blendMode uint16) uint16 {
	if model.GlobalFlags&GlobalFlagTextureCombiners != 0 && len(model.TextureCombinerCombos) > 0 {
		idx := int(unit.ShaderID)
		if idx < len(model.TextureCombinerCombos) {
			op0 := model.TextureCombinerCombos[idx] & 7
			op1 := uint16(1)
			if unit.TextureCount > 1 && idx+1 < len(model.TextureCombinerCombos) {
				op1 = model.TextureCombinerCombos[idx+1] & 7
			}
			return op0<<4 | op1
		}
	}
	if unit.ShaderID != 0 {
		return unit.ShaderID
	}
	op0 := uint16(0)
	if blendMode != BlendOpaque {
		op0 = 1
	}
	return op0<<4 | 1
}

// PixelShaderName maps a texture unit's shader_id to its combiner: the
// high nibble is the first texture's operation, the low one the second's
// (0 opaque, 1 mod, 3 add, 4 mod2x, 6 mod2xNA, 7 addAlpha).
func PixelShaderName(textureCount int, shaderID uint16) string {
	op0 := (shaderID >> 4) & 7
	op1 := shaderID & 7
	if textureCount < 2 {
		switch op0 {
		case 1:
			return "Combiners_Mod"
		case 2:
			return "Combiners_Decal"
		case 3:
			return "Combiners_Add"
		case 4:
			return "Combiners_Mod2x"
		case 5:
			return "Combiners_Fade"
		default:
			return "Combiners_Opaque"
		}
	}
	second := map[uint16]string{0: "Opaque", 1: "Mod", 3: "Add", 4: "Mod2x", 6: "Mod2xNA", 7: "AddAlpha"}
	s, ok := second[op1]
	if !ok {
		s = "Mod"
	}
	if op0 == 1 {
		if op1 == 7 {
			s = "AddNA"
		}
		return "Combiners_Mod_" + s
	}
	return "Combiners_Opaque_" + s
}

// trackLoop returns the timeline length of a track: its global loop when it
// is driven by one, otherwise sequence 0's duration.
func trackLoop(model *Model, t M2Track) uint32 {
	if t.GlobalSequence >= 0 && int(t.GlobalSequence) < len(model.GlobalSequences) {
		return model.GlobalSequences[t.GlobalSequence]
	}
	if len(model.Sequences) > 0 {
		return model.Sequences[0].Duration
	}
	return 0
}

// fixed16 decodes an M2 fixed-point alpha (0x7FFF = 1.0).
func fixed16(b []byte) float32 {
	v := float32(int16(uint16(b[0])|uint16(b[1])<<8)) / 32767
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// TextureAssetPath maps a BLP path to the asset-server path of its WebP.
func TextureAssetPath(blpPath string) string {
	if blpPath == "" {
		return ""
	}
	p := strings.ReplaceAll(blpPath, "\\", "/")
	if strings.HasSuffix(strings.ToLower(p), ".blp") {
		p = p[:len(p)-4]
	}
	return p + ".webp"
}

// ConvertToGLB converts an M2 model and skin directly to a GLB file stream.
func ConvertToGLB(model *Model, skin *Skin, w io.Writer, options ...Option) error {
	doc, err := ConvertToGLTF(model, skin, options...)
	if err != nil {
		return err
	}
	return doc.ToGLB(w)
}
