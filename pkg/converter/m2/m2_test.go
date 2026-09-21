package m2

import (
	"bytes"
	"os"
	"testing"

	"github.com/paalgyula/summit/pkg/converter/gltf"
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
		{2, 0x8001, "Combiners_Opaque_Mod2xNA_Alpha"},
		{2, 0x8002, "Combiners_Opaque_AddAlpha"},
		{2, 0x8003, "Combiners_Opaque_AddAlpha_Alpha"},
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

func TestAttachmentsExported(t *testing.T) {
	model := &Model{
		Name: "Attached",
		Vertices: []M2Vertex{
			{Pos: [3]float32{0, 0, 0}, BoneWeights: [4]uint8{255}},
			{Pos: [3]float32{1, 0, 0}, BoneWeights: [4]uint8{255}},
			{Pos: [3]float32{0, 1, 0}, BoneWeights: [4]uint8{255}},
		},
		Bones: []M2Bone{
			{ParentBone: -1, Pivot: [3]float32{0, 0, 1}},
			{ParentBone: 0, Pivot: [3]float32{0, 0.5, 1.5}},
		},
		Attachments: []M2Attachment{
			{ID: 1, Bone: 1, Position: [3]float32{0.2, 0.5, 1.5}},
			{ID: 99, Bone: 7}, // dangling bone: skipped
		},
	}
	doc, err := ConvertToGLTF(model, nil)
	if err != nil {
		t.Fatal(err)
	}
	var found *gltf.Node
	for i := range doc.Nodes {
		if doc.Nodes[i].Name == AttachmentNodePrefix+"1" {
			found = &doc.Nodes[i]
		}
		if doc.Nodes[i].Name == AttachmentNodePrefix+"99" {
			t.Error("attachment on a missing bone must be skipped")
		}
	}
	if found == nil {
		t.Fatal("Attach_1 node missing")
	}
	// (0.2, 0.5, 1.5) - pivot (0, 0.5, 1.5) = (0.2, 0, 0) in M2 space -> (0, 0, -0.2) in glTF space
	if *found.Translation != [3]float32{0, 0, -0.2} {
		t.Errorf("attachment translation: %v", *found.Translation)
	}
	// Parented to bone 1 (node index 1)
	parented := false
	for _, c := range doc.Nodes[1].Children {
		if doc.Nodes[c].Name == AttachmentNodePrefix+"1" {
			parented = true
		}
	}
	if !parented {
		t.Error("attachment must be a child of its bone node")
	}
}

func TestRealCharacterAttachments(t *testing.T) {
	path := "../../../client/assets/Character/Human/Male/HumanMale.m2"
	if _, err := os.Stat(path); err != nil {
		t.Skip("HumanMale.m2 not available")
	}
	model, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[uint32]bool{}
	for _, a := range model.Attachments {
		ids[a.ID] = true
	}
	// Right hand, left hand, shoulders, helm and the sheath points
	for _, want := range []uint32{0, 1, 2, 5, 6, 11, 26, 27} {
		if !ids[want] {
			t.Errorf("attachment %d missing (have %v)", want, ids)
		}
	}
}

func TestInspectTreeModel(t *testing.T) {
	m2Path := "../../../client/assets/world/AZEROTH/STRANGLETHORN/PASSIVEDOODADS/TREES/STRANGLETHORNTREE01/STRANGLETHORNTREE01.m2"
	skinPath := "../../../client/assets/world/AZEROTH/STRANGLETHORN/PASSIVEDOODADS/TREES/STRANGLETHORNTREE01/STRANGLETHORNTREE0100.skin"
	model, err := Open(m2Path)
	if err != nil {
		t.Fatalf("open m2: %v", err)
	}
	skin, err := OpenSkin(skinPath)
	if err != nil {
		t.Fatalf("open skin: %v", err)
	}
	t.Logf("Model Name: %s, GlobalFlags: %#x, Sequences: %d", model.Name, model.GlobalFlags, len(model.Sequences))
	for i, s := range model.Sequences {
		t.Logf("Sequence %d: ID=%d, Duration=%d, Flags=%#x", i, s.ID, s.Duration, s.Flags)
	}
	t.Logf("Vertices: %d, Bones: %d, Materials: %d, Textures: %d", len(model.Vertices), len(model.Bones), len(model.Materials), len(model.Textures))
	for i, b := range model.Bones {
		t.Logf("Bone %d: Flags=%#x, Parent=%d, KeyBone=%d, Pivot=%v, TransTracks=%d, RotTracks=%d",
			i, b.Flags, b.ParentBone, b.KeyBoneID, b.Pivot, len(b.Translation.Timestamps), len(b.Rotation.Timestamps))
	}
	for i, mat := range model.Materials {
		t.Logf("Material %d: Flags=%#x, BlendMode=%d", i, mat.Flags, mat.BlendMode)
	}
	for i, tex := range model.Textures {
		t.Logf("Texture %d: Type=%d, Flags=%#x, Name=%s", i, tex.Type, tex.Flags, tex.Name)
	}
	t.Logf("Submeshes: %d, TextureUnits: %d", len(skin.Submeshes), len(skin.TextureUnits))
	for i, sub := range skin.Submeshes {
		t.Logf("Submesh %d: ID=%d, VertStart=%d, VertCount=%d, TriStart=%d, TriCount=%d", i, sub.ID, sub.VertexStart, sub.VertexCount, sub.TriangleStart, sub.TriangleCount)
	}
	for i, u := range skin.TextureUnits {
		t.Logf("TextureUnit %d: Flags=%#x, PriorityPlane=%d, ShaderID=%#x, SubmeshIdx=%d, MatIdx=%d, Layer=%d, TexCount=%d, TexComboIdx=%d",
			i, u.Flags, u.PriorityPlane, u.ShaderID, u.SubmeshIndex, u.MaterialIndex, u.MaterialLayer, u.TextureCount, u.TextureComboIndex)
	}
	doc, err := ConvertToGLTF(model, skin)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	for i, mat := range doc.Materials {
		extras := mat.Extras.(MaterialExtras)
		t.Logf("GLTF Mat %d: Name=%s, DoubleSided=%v, AlphaMode=%s, Extras=%+v", i, mat.Name, mat.DoubleSided, mat.AlphaMode, extras)
	}
}
