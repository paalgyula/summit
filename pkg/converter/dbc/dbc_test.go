package dbc

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// buildDBC assembles a WDBC file from uint32 rows and a string block.
func buildDBC(rows [][]uint32, strs []byte) []byte {
	var b bytes.Buffer
	b.WriteString("WDBC")
	fields := 0
	if len(rows) > 0 {
		fields = len(rows[0])
	}
	_ = binary.Write(&b, binary.LittleEndian, uint32(len(rows)))
	_ = binary.Write(&b, binary.LittleEndian, uint32(fields))
	_ = binary.Write(&b, binary.LittleEndian, uint32(fields*4))
	_ = binary.Write(&b, binary.LittleEndian, uint32(len(strs)))
	for _, r := range rows {
		for _, v := range r {
			_ = binary.Write(&b, binary.LittleEndian, v)
		}
	}
	b.Write(strs)
	return b.Bytes()
}

func TestReadColumns(t *testing.T) {
	strs := []byte("\x00hello\x00Character\\Human\\Male\\HumanMaleSkin00_00.blp\x00")
	rows := [][]uint32{
		{1, 0xFFFFFFFF, math.Float32bits(1.5), 1},
		{2, 7, 0, 7},
	}
	f, err := Read(bytes.NewReader(buildDBC(rows, strs)))
	if err != nil {
		t.Fatal(err)
	}
	if f.Records != 2 || f.Fields != 4 {
		t.Fatalf("header: %d records, %d fields", f.Records, f.Fields)
	}
	if f.Int32(0, 1) != -1 || f.Float32(0, 2) != 1.5 || f.String(0, 3) != "hello" {
		t.Errorf("row 0: %d %v %q", f.Int32(0, 1), f.Float32(0, 2), f.String(0, 3))
	}
	if got := f.String(1, 3); got != "Character\\Human\\Male\\HumanMaleSkin00_00.blp" {
		t.Errorf("row 1 string: %q", got)
	}
	if f.Uint32(5, 0) != 0 || f.String(0, 9) != "" {
		t.Error("out of range access must be zero / empty")
	}
}

func TestReadRejectsGarbage(t *testing.T) {
	if _, err := Read(bytes.NewReader([]byte("not a dbc file at all"))); err == nil {
		t.Fatal("expected an error")
	}
}

func TestBuildCharacterData(t *testing.T) {
	strs := []byte{0}
	intern := func(s string) uint32 {
		ofs := uint32(len(strs))
		strs = append(strs, s...)
		strs = append(strs, 0)
		return ofs
	}
	hu, human := intern("Hu"), intern("Human")
	face := intern("Character\\Human\\Male\\HumanMaleFaceLower00_01.blp")
	sword, chest := intern("Sword_1H_Short_A_01.mdx"), intern("Cloth_A_01_Chest_TU")

	races := make([]uint32, 69)
	races[0], races[6], races[14] = 1, hu, human
	sections := [][]uint32{{5184, 1, 0, SectionFace, face, 0, 0, 1, 0, 1}}
	hair := [][]uint32{{21, 1, 0, 0, 0, 1}, {22, 1, 0, 1, 2, 0}}
	facial := [][]uint32{{1, 0, 1, 1, 2, 3, 0, 0}}
	helmets := [][]uint32{{246, 0xFFFFFFBF, 0, 128, 4, 0xFFFFFB6F, 0, 0}}
	display := make([][]uint32, 3)
	display[0] = make([]uint32, 25)
	display[0][0], display[0][1] = 1542, sword
	display[1] = make([]uint32, 25)
	display[1][0], display[1][8], display[1][18] = 9873, 1, chest
	display[2] = make([]uint32, 25)
	display[2][0] = 7 // icon only: dropped
	items := [][]uint32{{25, 2, 7, 0, 1, 1542, 21, 3}}

	outfit := make([]uint32, 74)
	outfit[0] = 1                                // id
	outfit[1] = 1 | (1 << 8) | (0 << 16)         // race 1 (Human), class 1 (Warrior), gender 0 (Male)
	outfit[26] = 1542                            // displayId for first item
	outfit[50] = 13                              // inventoryType 13 (1H weapon)

	read := func(rows [][]uint32) *File {
		f, err := Read(bytes.NewReader(buildDBC(rows, strs)))
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	data := BuildCharacterData(Tables{
		ChrRaces:            read([][]uint32{races}),
		CharSections:        read(sections),
		CharHairGeosets:     read(hair),
		FacialHairStyles:    read(facial),
		HelmetGeosetVisData: read(helmets),
		ItemDisplayInfo:     read(display),
		Item:                read(items),
		CharStartOutfit:     read([][]uint32{outfit}),
	})

	if data.Races[1] != (Race{Name: "Human", Prefix: "Hu"}) {
		t.Errorf("race: %+v", data.Races[1])
	}
	if len(data.Sections) != 1 || data.Sections[0].Textures[0] != "Character/Human/Male/HumanMaleFaceLower00_01.webp" || data.Sections[0].Color != 1 {
		t.Errorf("section: %+v", data.Sections)
	}
	if len(data.HairGeosets) != 2 || !data.HairGeosets[0].ShowScalp || data.HairGeosets[1].Geoset != 2 {
		t.Errorf("hair geosets: %+v", data.HairGeosets)
	}
	if data.FacialHair[0].Geosets != [3]int{1, 3, 2} {
		t.Errorf("facial hair groups 1,2,3: %v", data.FacialHair[0].Geosets)
	}
	if data.HelmetGeosets[246][0] != 0xFFFFFFBF {
		t.Errorf("helmet masks: %v", data.HelmetGeosets[246])
	}
	if d := data.ItemDisplay[1542]; d.Models[0] != "Sword_1H_Short_A_01" || d.Sheathe != 3 {
		t.Errorf("sword display: %+v", d)
	}
	if d := data.ItemDisplay[9873]; d.Geosets != [3]int{0, 1, 0} || len(d.Components) != 4 || d.Components[3] != "Cloth_A_01_Chest_TU" {
		t.Errorf("chest display: %+v", d)
	}
	if _, ok := data.ItemDisplay[7]; ok {
		t.Error("icon-only display must be dropped")
	}
	if items, ok := data.Outfits["1_1_0"]; !ok || len(items) != 1 {
		t.Errorf("outfit 1_1_0: %+v", data.Outfits["1_1_0"])
	} else if items[0] != (StartOutfitItem{Slot: 15, DisplayID: 1542, InventoryType: 13}) {
		t.Errorf("outfit item: %+v", items[0])
	}
}
