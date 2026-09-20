package wmo

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWMORoundtrip(t *testing.T) {
	// Build synthetic Root WMO
	var rootBuf bytes.Buffer

	// MOHD chunk
	mohdData := make([]byte, 64)
	binary.LittleEndian.PutUint32(mohdData[0:4], 1) // 1 material
	binary.LittleEndian.PutUint32(mohdData[4:8], 1) // 1 group
	binary.Write(&rootBuf, binary.LittleEndian, uint32(MagicMOHD))
	binary.Write(&rootBuf, binary.LittleEndian, uint32(len(mohdData)))
	rootBuf.Write(mohdData)

	// MOTX chunk
	texStr := "Textures\\Wood.blp\x00"
	binary.Write(&rootBuf, binary.LittleEndian, uint32(MagicMOTX))
	binary.Write(&rootBuf, binary.LittleEndian, uint32(len(texStr)))
	rootBuf.WriteString(texStr)

	// MOMT chunk
	momtData := make([]byte, 64)
	binary.Write(&rootBuf, binary.LittleEndian, uint32(MagicMOMT))
	binary.Write(&rootBuf, binary.LittleEndian, uint32(len(momtData)))
	rootBuf.Write(momtData)

	rootWMO, err := ReadRoot(&rootBuf)
	if err != nil {
		t.Fatalf("ReadRoot failed: %v", err)
	}

	if rootWMO.Header.NMaterials != 1 {
		t.Fatalf("expected 1 material, got %d", rootWMO.Header.NMaterials)
	}

	if len(rootWMO.Textures) != 1 || rootWMO.Textures[0] != "Textures\\Wood.blp" {
		t.Fatalf("unexpected textures: %+v", rootWMO.Textures)
	}

	// Build synthetic Group WMO
	var grpBuf bytes.Buffer

	// MOVT (3 vertices)
	binary.Write(&grpBuf, binary.LittleEndian, uint32(MagicMOVT))
	binary.Write(&grpBuf, binary.LittleEndian, uint32(3*12))
	for _, pt := range [][3]float32{{0, 0, 0}, {10, 0, 0}, {0, 10, 0}} {
		binary.Write(&grpBuf, binary.LittleEndian, pt[0])
		binary.Write(&grpBuf, binary.LittleEndian, pt[1])
		binary.Write(&grpBuf, binary.LittleEndian, pt[2])
	}

	// MONR (3 normals)
	binary.Write(&grpBuf, binary.LittleEndian, uint32(MagicMONR))
	binary.Write(&grpBuf, binary.LittleEndian, uint32(3*12))
	for i := 0; i < 3; i++ {
		binary.Write(&grpBuf, binary.LittleEndian, float32(0))
		binary.Write(&grpBuf, binary.LittleEndian, float32(0))
		binary.Write(&grpBuf, binary.LittleEndian, float32(1))
	}

	// MOTV (3 UVs)
	binary.Write(&grpBuf, binary.LittleEndian, uint32(MagicMOTV))
	binary.Write(&grpBuf, binary.LittleEndian, uint32(3*8))
	for _, uv := range [][2]float32{{0, 0}, {1, 0}, {0, 1}} {
		binary.Write(&grpBuf, binary.LittleEndian, uv[0])
		binary.Write(&grpBuf, binary.LittleEndian, uv[1])
	}

	// MOVI (3 indices)
	binary.Write(&grpBuf, binary.LittleEndian, uint32(MagicMOVI))
	binary.Write(&grpBuf, binary.LittleEndian, uint32(3*2))
	binary.Write(&grpBuf, binary.LittleEndian, uint16(0))
	binary.Write(&grpBuf, binary.LittleEndian, uint16(1))
	binary.Write(&grpBuf, binary.LittleEndian, uint16(2))

	grp, err := ReadGroup(&grpBuf)
	if err != nil {
		t.Fatalf("ReadGroup failed: %v", err)
	}

	if len(grp.Vertices) != 3 {
		t.Fatalf("expected 3 vertices, got %d", len(grp.Vertices))
	}

	// Attach group to root and export to GLB
	rootWMO.Groups = append(rootWMO.Groups, grp)

	var glbBuf bytes.Buffer
	if err := rootWMO.ExportGLB(&glbBuf); err != nil {
		t.Fatalf("ExportGLB failed: %v", err)
	}

	if glbBuf.Len() < 50 {
		t.Fatalf("GLB buffer too small: %d", glbBuf.Len())
	}
}
