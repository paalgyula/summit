package wmo

import (
	"fmt"
	"io"
	"strings"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

// MaterialExtras is attached to every WMO material so the client can fetch
// the texture and set up blending the way the WoW renderer would.
type MaterialExtras struct {
	// Texture is the asset-server path of the material's first texture (WebP).
	Texture string `json:"texture,omitempty"`
	// Texture2 is the second texture of the two-layer / env / emissive shaders.
	Texture2 string `json:"texture2,omitempty"`
	// Shader is the MOMT shader (0 Diffuse, 3 Env, 6 TwoLayerDiffuse, 9 DiffuseEmissive, ...).
	Shader uint32 `json:"shader"`
	// BlendMode is the MOMT blend mode (0 opaque, 1 alpha key, 2 alpha blend, ...).
	BlendMode uint32 `json:"blendMode"`
	Unlit     bool   `json:"unlit,omitempty"`
	Unfogged  bool   `json:"unfogged,omitempty"`
	TwoSided  bool   `json:"twoSided,omitempty"`
	ClampS    bool   `json:"clampS,omitempty"`
	ClampT    bool   `json:"clampT,omitempty"`
	Window    bool   `json:"window,omitempty"`
	// Emissive is the material's night-glow (SIDN) colour when the SIDN flag is set.
	Emissive [3]float32 `json:"emissive,omitempty"`
}

// GroupExtras is attached to every group mesh node.
type GroupExtras struct {
	Group int `json:"group"`
	// Indoor groups are lit by their baked vertex colours, not the sky.
	Indoor bool `json:"indoor,omitempty"`
	// ExteriorLit indoor groups still receive the exterior light.
	ExteriorLit  bool `json:"exteriorLit,omitempty"`
	VertexColors bool `json:"vertexColors,omitempty"`
}

// DoodadExtras is attached to every doodad placeholder node of the WMO. The
// node carries the doodad's transform relative to the WMO root.
type DoodadExtras struct {
	Kind  string `json:"kind"` // always "m2"
	Model string `json:"model"`
	// Set is the doodad set the entry belongs to; the client shows set 0 and
	// the set the map placement selects.
	Set int `json:"set"`
}

// ExportGLB converts the WMO into a glTF GLB binary: one mesh node per group
// (a primitive per render batch) plus one empty node per doodad.
func (wmo *RootWMO) ExportGLB(w io.Writer) error {
	doc := gltf.NewDocument()

	matIndices := make([]int, len(wmo.Materials))
	for i, m := range wmo.Materials {
		mIdx := len(doc.Materials)
		matIndices[i] = mIdx
		mat := gltf.Material{
			Name: fmt.Sprintf("WMOMaterial_%d", i),
			PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
				BaseColorFactor: [4]float32{1, 1, 1, 1},
				MetallicFactor:  0.0,
				RoughnessFactor: 1.0,
			},
			DoubleSided: m.Flags&MaterialFlagTwoSided != 0,
		}
		extras := MaterialExtras{
			Texture:   TextureAssetPath(m.Texture),
			Texture2:  secondTexture(m),
			Shader:    m.Shader,
			BlendMode: m.BlendMode,
			Unlit:     m.Flags&MaterialFlagUnlit != 0,
			Unfogged:  m.Flags&MaterialFlagUnfogged != 0,
			TwoSided:  m.Flags&MaterialFlagTwoSided != 0,
			ClampS:    m.Flags&MaterialFlagClampS != 0,
			ClampT:    m.Flags&MaterialFlagClampT != 0,
			Window:    m.Flags&MaterialFlagWindow != 0,
		}
		if m.Flags&MaterialFlagSIDN != 0 {
			// Color1 is BGRA
			extras.Emissive = [3]float32{float32(m.Color1[2]) / 255, float32(m.Color1[1]) / 255, float32(m.Color1[0]) / 255}
		}
		mat.Extras = extras
		switch m.BlendMode {
		case BlendAlphaKey:
			cutoff := float32(0.5)
			mat.AlphaMode = gltf.AlphaModeMask
			mat.AlphaCutoff = &cutoff
		case BlendOpaque:
		default:
			mat.AlphaMode = gltf.AlphaModeBlend
		}
		doc.Materials = append(doc.Materials, mat)
	}

	for gIdx, grp := range wmo.Groups {
		if grp == nil || len(grp.Vertices) == 0 {
			continue
		}

		positions := make([][3]float32, len(grp.Vertices))
		normals := make([][3]float32, len(grp.Vertices))
		uvs := make([][2]float32, len(grp.Vertices))
		var uvs2 [][2]float32
		if len(grp.TexCoords2) == len(grp.Vertices) {
			uvs2 = grp.TexCoords2
		}
		var colors [][4]float32
		if len(grp.Colors) == len(grp.Vertices) {
			colors = make([][4]float32, len(grp.Vertices))
		}

		for i, v := range grp.Vertices {
			positions[i] = gltf.ConvertWoWToGLTPosition(v[0], v[1], v[2])
			if i < len(grp.Normals) {
				n := grp.Normals[i]
				normals[i] = gltf.ConvertWoWToGLTPosition(n[0], n[1], n[2])
			}
			if i < len(grp.TexCoords) {
				uvs[i] = grp.TexCoords[i]
			}
			if colors != nil {
				c := grp.Colors[i] // BGRA
				colors[i] = [4]float32{float32(c[2]) / 255, float32(c[1]) / 255, float32(c[0]) / 255, float32(c[3]) / 255}
			}
		}

		attrs := map[string]int{
			"POSITION":   doc.AddFloat32Vec3Accessor(positions, gltf.TargetArrayBuffer),
			"NORMAL":     doc.AddFloat32Vec3Accessor(normals, gltf.TargetArrayBuffer),
			"TEXCOORD_0": doc.AddFloat32Vec2Accessor(uvs, gltf.TargetArrayBuffer),
		}
		if colors != nil {
			attrs["COLOR_0"] = doc.AddFloat32Vec4Accessor(colors, gltf.TargetArrayBuffer)
		}
		if uvs2 != nil {
			attrs["TEXCOORD_1"] = doc.AddFloat32Vec2Accessor(uvs2, gltf.TargetArrayBuffer)
		}

		var primitives []gltf.Primitive
		for _, b := range grp.Batches {
			start, count := int(b.StartIndex), int(b.IndexCount)
			if count == 0 || start+count > len(grp.Indices) {
				continue
			}
			idxAcc := doc.AddUint16IndicesAccessor(grp.Indices[start : start+count])

			var matRef *int
			if int(b.MaterialID) < len(matIndices) {
				matRef = &matIndices[b.MaterialID]
			}
			primitives = append(primitives, gltf.Primitive{
				Attributes: attrs,
				Indices:    &idxAcc,
				Material:   matRef,
			})
		}

		if len(primitives) == 0 && len(grp.Indices) > 0 {
			idxAcc := doc.AddUint16IndicesAccessor(grp.Indices)
			primitives = append(primitives, gltf.Primitive{Attributes: attrs, Indices: &idxAcc})
		}
		if len(primitives) == 0 {
			continue
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
			Extras: GroupExtras{
				Group:        gIdx,
				Indoor:       grp.Flags&GroupFlagIndoor != 0,
				ExteriorLit:  grp.Flags&GroupFlagExteriorLit != 0,
				VertexColors: colors != nil,
			},
		})
		doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)
	}

	wmo.addDoodadNodes(doc)

	return doc.ToGLB(w)
}

// addDoodadNodes emits an empty, transformed node per MODD entry. Doodad
// positions and orientations are in the WMO's own frame, so the same axis
// conversion as for the vertices applies.
func (wmo *RootWMO) addDoodadNodes(doc *gltf.Document) {
	setOf := make([]int, len(wmo.Doodads))
	for i := range setOf {
		setOf[i] = -1
	}
	for si, set := range wmo.DoodadSets {
		for i := set.Start; i < set.Start+set.Count && int(i) < len(setOf); i++ {
			if setOf[i] < 0 {
				setOf[i] = si
			}
		}
	}

	for i, d := range wmo.Doodads {
		if d.Name == "" || setOf[i] < 0 {
			continue
		}
		t := gltf.ConvertWoWToGLTPosition(d.Pos[0], d.Pos[1], d.Pos[2])
		r := gltf.ConvertWoWQuaternion(d.Rot)
		s := [3]float32{d.Scale, d.Scale, d.Scale}

		nodeIdx := len(doc.Nodes)
		node := gltf.Node{
			Name:        fmt.Sprintf("doodad_%d", i),
			Translation: &t,
			Rotation:    &r,
			Extras:      DoodadExtras{Kind: "m2", Model: ModelAssetPath(d.Name), Set: setOf[i]},
		}
		if d.Scale != 1 {
			node.Scale = &s
		}
		doc.Nodes = append(doc.Nodes, node)
		doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)
	}
}

// secondTexture returns the material's second texture for the shaders that
// use one; unused slots point at offset 0, i.e. the first name in MOTX.
func secondTexture(m Material) string {
	switch m.Shader {
	case 0, 1, 2, 4, 10, 21: // Diffuse, Specular, Metal, Opaque, WaterWindow, Lod
		return ""
	}
	return TextureAssetPath(m.Texture2Name)
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

// ModelAssetPath maps an M2/MDX path to the asset-server path of its GLB.
func ModelAssetPath(name string) string {
	p := strings.ReplaceAll(name, "\\", "/")
	lower := strings.ToLower(p)
	for _, ext := range []string{".mdx", ".mdl", ".m2"} {
		if strings.HasSuffix(lower, ext) {
			p = p[:len(p)-len(ext)]
			break
		}
	}
	return p + ".glb"
}
