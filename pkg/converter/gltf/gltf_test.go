package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestGLBRoundtrip(t *testing.T) {
	doc := NewDocument()

	// Simple triangle
	positions := [][3]float32{
		{0, 0, 0},
		{1, 0, 0},
		{0, 1, 0},
	}
	normals := [][3]float32{
		{0, 0, 1},
		{0, 0, 1},
		{0, 0, 1},
	}
	uvs := [][2]float32{
		{0, 0},
		{1, 0},
		{0, 1},
	}
	indices := []uint16{0, 1, 2}

	posAcc := doc.AddFloat32Vec3Accessor(positions, TargetArrayBuffer)
	normAcc := doc.AddFloat32Vec3Accessor(normals, TargetArrayBuffer)
	uvAcc := doc.AddFloat32Vec2Accessor(uvs, TargetArrayBuffer)
	idxAcc := doc.AddUint16IndicesAccessor(indices)

	matIdx := len(doc.Materials)
	doc.Materials = append(doc.Materials, Material{
		Name: "TestMaterial",
		PbrMetallicRoughness: &PbrMetallicRoughness{
			BaseColorFactor: [4]float32{1, 0, 0, 1},
			MetallicFactor:  0.1,
			RoughnessFactor: 0.9,
		},
	})

	meshIdx := len(doc.Meshes)
	doc.Meshes = append(doc.Meshes, Mesh{
		Name: "TestMesh",
		Primitives: []Primitive{
			{
				Attributes: map[string]int{
					"POSITION":   posAcc,
					"NORMAL":     normAcc,
					"TEXCOORD_0": uvAcc,
				},
				Indices:  &idxAcc,
				Material: &matIdx,
			},
		},
	})

	nodeIdx := len(doc.Nodes)
	doc.Nodes = append(doc.Nodes, Node{
		Name: "TestNode",
		Mesh: &meshIdx,
	})
	doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)

	var buf bytes.Buffer
	if err := doc.ToGLB(&buf); err != nil {
		t.Fatalf("ToGLB failed: %v", err)
	}

	data := buf.Bytes()
	if len(data) < 20 {
		t.Fatalf("GLB output too short: %d bytes", len(data))
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != GLBMagic {
		t.Fatalf("expected GLBMagic 0x%08X, got 0x%08X", GLBMagic, magic)
	}

	version := binary.LittleEndian.Uint32(data[4:8])
	if version != 2 {
		t.Fatalf("expected version 2, got %d", version)
	}

	totalLen := binary.LittleEndian.Uint32(data[8:12])
	if int(totalLen) != len(data) {
		t.Fatalf("expected total length %d, got %d", totalLen, len(data))
	}

	jsonLen := binary.LittleEndian.Uint32(data[12:16])
	jsonType := binary.LittleEndian.Uint32(data[16:20])
	if jsonType != GLBJSON {
		t.Fatalf("expected JSON chunk type, got 0x%08X", jsonType)
	}

	jsonBytes := data[20 : 20+jsonLen]
	var parsedDoc Document
	if err := json.Unmarshal(jsonBytes, &parsedDoc); err != nil {
		t.Fatalf("failed to unmarshal parsed JSON: %v", err)
	}

	if len(parsedDoc.Meshes) != 1 || len(parsedDoc.Nodes) != 1 {
		t.Fatalf("expected 1 mesh and 1 node, got %d meshes, %d nodes", len(parsedDoc.Meshes), len(parsedDoc.Nodes))
	}
}
