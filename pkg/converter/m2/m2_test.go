package m2

import (
	"bytes"
	"fmt"
	"os"
	"strings"
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
		t.Logf("Attachment %d: Bone=%d, Pos=%v", a.ID, a.Bone, a.Position)
	}
	for _, bIdx := range []uint16{100, 101, 102, 123, 125, 126, 84, 85} {
		b := model.Bones[bIdx]
		t.Logf("Bone %d: Key=%d, Parent=%d, Pivot=%v, TransTracks=%d, RotTracks=%d",
			bIdx, b.KeyBoneID, b.ParentBone, b.Pivot, len(b.Translation.Timestamps), len(b.Rotation.Timestamps))
		for seq := 0; seq < len(model.Sequences) && seq < 2; seq++ {
			times, raw := model.TrackKeys(b.Rotation, seq, 8, nil)
			if len(times) > 0 {
				q := decodeCompQuat(raw[0:8])
				t.Logf("  Bone %d seq %d (id=%d): M2Quat=[%.3f, %.3f, %.3f, %.3f], glTFQuat=[%.3f, %.3f, %.3f, %.3f]",
					bIdx, seq, model.Sequences[seq].ID, q[0], q[1], q[2], q[3], -q[1], q[2], -q[0], q[3])
			}
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

func TestInspectItemModels(t *testing.T) {
	for _, p := range []string{
		"../../../client/assets/ITEM/OBJECTCOMPONENTS/SHIELD/Shield_Round_A_01.m2",
		"../../../client/assets/ITEM/OBJECTCOMPONENTS/Weapon/Sword_1H_Short_A_02.m2",
	} {
		m, err := Open(p)
		if err != nil {
			t.Logf("open %s error: %v", p, err)
			continue
		}
		var minX, maxX, minY, maxY, minZ, maxZ float32
		if len(m.Vertices) > 0 {
			minX, maxX = m.Vertices[0].Pos[0], m.Vertices[0].Pos[0]
			minY, maxY = m.Vertices[0].Pos[1], m.Vertices[0].Pos[1]
			minZ, maxZ = m.Vertices[0].Pos[2], m.Vertices[0].Pos[2]
			for _, v := range m.Vertices {
				if v.Pos[0] < minX { minX = v.Pos[0] }
				if v.Pos[0] > maxX { maxX = v.Pos[0] }
				if v.Pos[1] < minY { minY = v.Pos[1] }
				if v.Pos[1] > maxY { maxY = v.Pos[1] }
				if v.Pos[2] < minZ { minZ = v.Pos[2] }
				if v.Pos[2] > maxZ { maxZ = v.Pos[2] }
			}
		}
		t.Logf("ITEM %s: Verts=%d, Bones=%d, Attachments=%d", p, len(m.Vertices), len(m.Bones), len(m.Attachments))
		t.Logf("  Bounds M2: X=[%.2f, %.2f], Y=[%.2f, %.2f], Z=[%.2f, %.2f]", minX, maxX, minY, maxY, minZ, maxZ)
		if len(m.Vertices) > 0 {
			for vi := 0; vi < len(m.Vertices) && vi < 5; vi++ {
				t.Logf("  Vert %d: Pos=%v, Normal=%v", vi, m.Vertices[vi].Pos, m.Vertices[vi].Normal)
			}
		}
		for i, b := range m.Bones {
			t.Logf("  Bone %d: Key=%d, Parent=%d, Pivot=%v", i, b.KeyBoneID, b.ParentBone, b.Pivot)
		}
		for i, a := range m.Attachments {
			t.Logf("  Attachment %d: ID=%d, Bone=%d, Pos=%v", i, a.ID, a.Bone, a.Position)
		}
	}
}

func TestInspectHumanMaleConversion(t *testing.T) {
	path := "../../../client/assets/Character/Human/Male/HumanMale.m2"
	skinPath := "../../../client/assets/Character/Human/Male/HumanMale00.skin"
	model, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	skin, err := OpenSkin(skinPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ConvertToGLTF(model, skin)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("GlobalSequences: %v", model.GlobalSequences)
	sheathBoneIds := map[uint8]bool{84: true, 85: true, 100: true, 101: true, 102: true}
	var weightedCount int
	for _, v := range model.Vertices {
		for i := 0; i < 4; i++ {
			if v.BoneWeights[i] > 0 && sheathBoneIds[v.BoneIndices[i]] {
				weightedCount++
			}
		}
	}
	t.Logf("Vertices skinned to sheath bones: %d", weightedCount)
	for _, bIdx := range []int{84, 85, 100, 101, 102} {
		b := model.Bones[bIdx]
		times, raw := model.TrackKeys(b.Rotation, 0, 8, nil)
		var q [4]float32
		if len(raw) >= 8 {
			q = decodeCompQuat(raw[0:8])
		}
		t.Logf("Bone %d: RotGS=%d, Times=%v, Quat=%v, glTFQuat=[%.3f, %.3f, %.3f, %.3f]",
			bIdx, b.Rotation.GlobalSequence, times, q, -q[1], q[2], -q[0], q[3])
	}
	for _, anim := range doc.Animations {
		if anim.Name == "Stand" {
			t.Logf("Stand channels: %d", len(anim.Channels))
			for _, ch := range anim.Channels {
				targetNode := doc.Nodes[ch.Target.Node]
				if strings.Contains(targetNode.Name, "102") || strings.Contains(targetNode.Name, "84") || strings.Contains(targetNode.Name, "100") {
					t.Logf("  Channel targeting %s: path=%s", targetNode.Name, ch.Target.Path)
				}
			}
		}
	}
}

func TestInspectAllRacesSheaths(t *testing.T) {
	for _, raceDir := range []string{"Human", "Orc", "Dwarf", "NightElf", "Scourge", "Tauren", "Gnome", "Troll"} {
		for _, sex := range []string{"Male", "Female"} {
			p := fmt.Sprintf("../../../client/assets/Character/%s/%s/%s%s.m2", raceDir, sex, raceDir, sex)
			m, err := Open(p)
			if err != nil {
				continue
			}
			t.Logf("=== %s %s ===", raceDir, sex)
			for _, att := range m.Attachments {
				if att.ID == 28 || att.ID == 26 || att.ID == 32 {
					b := m.Bones[att.Bone]
					times, raw := m.TrackKeys(b.Rotation, 0, 8, nil)
					var q [4]float32
					if len(raw) >= 8 {
						q = decodeCompQuat(raw[0:8])
					}
					t.Logf("  Att %d (bone %d): RotGS=%d, Times=%v, Quat=%v, glTFQuat=[%.3f, %.3f, %.3f, %.3f]",
						att.ID, att.Bone, b.Rotation.GlobalSequence, times, q, -q[1], q[2], -q[0], q[3])
				}
			}
		}
	}
}
