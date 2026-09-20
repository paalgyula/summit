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
