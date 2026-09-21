package m2

import (
	"bytes"
	"os"
	"testing"
)

func TestM2ConversionSynthetic(t *testing.T) {
	model := &Model{
		Name: "TestModel",
		Vertices: []M2Vertex{
			{Pos: [3]float32{0, 0, 0}, Normal: [3]float32{0, 0, 1}, TexCoords: [2]float32{0, 0}, BoneIndices: [4]uint8{0, 0, 0, 0}, BoneWeights: [4]uint8{255, 0, 0, 0}},
			{Pos: [3]float32{1, 0, 0}, Normal: [3]float32{0, 0, 1}, TexCoords: [2]float32{1, 0}, BoneIndices: [4]uint8{0, 0, 0, 0}, BoneWeights: [4]uint8{255, 0, 0, 0}},
			{Pos: [3]float32{0, 1, 0}, Normal: [3]float32{0, 0, 1}, TexCoords: [2]float32{0, 1}, BoneIndices: [4]uint8{0, 0, 0, 0}, BoneWeights: [4]uint8{255, 0, 0, 0}},
		},
		Bones: []M2Bone{
			{KeyBoneID: 0, ParentBone: -1, Pivot: [3]float32{0, 0, 0}},
		},
	}

	skin := &Skin{
		Indices:   []uint16{0, 1, 2},
		Triangles: []uint16{0, 1, 2},
		Submeshes: []Submesh{
			{ID: 0, VertexStart: 0, VertexCount: 3, TriangleStart: 0, TriangleCount: 3},
		},
	}

	var buf bytes.Buffer
	if err := ConvertToGLB(model, skin, &buf); err != nil {
		t.Fatalf("ConvertToGLB failed: %v", err)
	}

	if buf.Len() < 50 {
		t.Fatalf("GLB buffer too small: %d", buf.Len())
	}
}

func TestM2ParseRealAsset(t *testing.T) {
	realModelPath := "../../../client/assets/models/npc/4218362.m2"
	if _, err := os.Stat(realModelPath); err != nil {
		t.Skip("skipping real asset test, file not found")
	}

	model, err := Open(realModelPath)
	if err != nil {
		t.Fatalf("failed to open real M2 asset: %v", err)
	}

	if len(model.Vertices) == 0 {
		t.Fatalf("expected vertices to be loaded from real M2 asset")
	}

	t.Logf("Successfully loaded real M2: %d vertices, %d bones, %d sequences",
		len(model.Vertices), len(model.Bones), len(model.Sequences))

	var buf bytes.Buffer
	if err := ConvertToGLB(model, nil, &buf); err != nil {
		t.Fatalf("failed to convert real M2 to GLB: %v", err)
	}

	t.Logf("Successfully converted real M2 to GLB: %d bytes", buf.Len())
}

func TestPixelShaderSelection(t *testing.T) {
	cases := []struct {
		count int
		id    uint16
		want  string
	}{
		{1, 0x00, "Combiners_Opaque"},
		{1, 0x10, "Combiners_Mod"},
		{2, 0x01, "Combiners_Opaque_Mod"},
		{2, 0x03, "Combiners_Opaque_Add"},
		{2, 0x16, "Combiners_Mod_Mod2xNA"},
		{2, 0x17, "Combiners_Mod_AddNA"},
		{2, 0x07, "Combiners_Opaque_AddAlpha"},
	}
	for _, c := range cases {
		if got := PixelShaderName(c.count, c.id); got != c.want {
			t.Errorf("PixelShaderName(%d, %#x) = %s, want %s", c.count, c.id, got, c.want)
		}
	}

	// WotLK batches carry shader_id 0: the blend mode picks Opaque or Mod
	model := &Model{}
	if id := unitShaderID(model, &TextureUnit{TextureCount: 1}, BlendAlphaKey); PixelShaderName(1, id) != "Combiners_Mod" {
		t.Errorf("alpha-keyed unit should use Combiners_Mod, got %s", PixelShaderName(1, id))
	}
	if id := unitShaderID(model, &TextureUnit{TextureCount: 1}, BlendOpaque); PixelShaderName(1, id) != "Combiners_Opaque" {
		t.Errorf("opaque unit should use Combiners_Opaque, got %s", PixelShaderName(1, id))
	}

	// With the combiner table, shader_id indexes it
	model = &Model{GlobalFlags: GlobalFlagTextureCombiners, TextureCombinerCombos: []uint16{1, 4}}
	if id := unitShaderID(model, &TextureUnit{TextureCount: 2, ShaderID: 0}, BlendOpaque); PixelShaderName(2, id) != "Combiners_Mod_Mod2x" {
		t.Errorf("combo table: got %s", PixelShaderName(2, id))
	}
}
