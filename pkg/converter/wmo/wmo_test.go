package wmo

import (
	"bytes"
	"encoding/binary"
	"math"
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

func TestParseBatchesLayout(t *testing.T) {
	// SMOBatch: int16 bounds[6], u32 startIndex, u16 count, u16 minIndex, u16 maxIndex, u8 flags, u8 material
	data := make([]byte, 24)
	binary.LittleEndian.PutUint32(data[12:], 702)
	binary.LittleEndian.PutUint16(data[16:], 6327)
	binary.LittleEndian.PutUint16(data[18:], 197)
	binary.LittleEndian.PutUint16(data[20:], 1406)
	data[23] = 2

	b := parseBatches(data)
	if len(b) != 1 || b[0].StartIndex != 702 || b[0].IndexCount != 6327 || b[0].VertexStart != 197 || b[0].VertexEnd != 1406 || b[0].MaterialID != 2 {
		t.Fatalf("unexpected batch: %+v", b)
	}
}

func TestDoodadSetsAndNames(t *testing.T) {
	var buf bytes.Buffer
	write := func(magic uint32, data []byte) {
		binary.Write(&buf, binary.LittleEndian, magic)
		binary.Write(&buf, binary.LittleEndian, uint32(len(data)))
		buf.Write(data)
	}

	modn := []byte("a.mdx\x00World\\Torch.mdx\x00")
	write(MagicMODN, modn)

	mods := make([]byte, 64)
	copy(mods[0:], "Set_$DefaultGlobal")
	copy(mods[32:], "set_Inside")
	binary.LittleEndian.PutUint32(mods[32+20:], 0)
	binary.LittleEndian.PutUint32(mods[32+24:], 1)
	write(MagicMODS, mods)

	modd := make([]byte, 40)
	binary.LittleEndian.PutUint32(modd[0:], 6|0x01<<24) // name offset 6, flags 1
	binary.LittleEndian.PutUint32(modd[16:], math.Float32bits(0))
	binary.LittleEndian.PutUint32(modd[28:], math.Float32bits(1)) // quaternion w
	binary.LittleEndian.PutUint32(modd[32:], math.Float32bits(0.5))
	write(MagicMODD, modd)

	root, err := ReadRoot(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(root.DoodadSets) != 2 || root.DoodadSets[1].Name != "set_Inside" || root.DoodadSets[1].Count != 1 {
		t.Fatalf("unexpected sets: %+v", root.DoodadSets)
	}
	if len(root.Doodads) != 1 || root.Doodads[0].Name != `World\Torch.mdx` || root.Doodads[0].Flags != 1 || root.Doodads[0].Scale != 0.5 {
		t.Fatalf("unexpected doodad: %+v", root.Doodads)
	}
	if GroupFileName(`World\wmo\Foo.wmo`, 3) != `World\wmo\Foo_003.wmo` {
		t.Fatalf("GroupFileName: %q", GroupFileName(`World\wmo\Foo.wmo`, 3))
	}
}
