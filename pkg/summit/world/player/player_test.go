package player_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/guid"
	"github.com/paalgyula/summit/pkg/summit/world/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const packetData = `# code: 0x0036 len: 00014
QmVsYQAGCwAJBAIAAwA=`

type CharacterCreateRequest struct {
	Race       uint8
	Class      uint8
	Gender     uint8
	Skin       uint8
	Face       uint8
	HairStyle  uint8
	HairColor  uint8
	FacialHair uint8
	OutfitId   uint8
}

func TestCreatePlayer(t *testing.T) {
	code, data, err := wow.ParseDumpedPacket(packetData)
	assert.NoError(t, err)
	assert.Equal(t, 0x36, code)

	fmt.Printf("%s", hex.Dump(data))

	r := wow.NewPacketReader(data)
	var accName string
	r.ReadString(&accName)

	assert.Equal(t, "Bela", accName)

	var request CharacterCreateRequest
	r.Read(&request)

	fmt.Printf("%s %+v\n", accName, request)

	_, _ = code, data
}

func TestParseCharEnum(t *testing.T) {
	p := player.Player{
		ID:         0,
		Name:       "Bela",
		Race:       6,
		Class:      11,
		Gender:     1,
		Skin:       3,
		Face:       4,
		HairStyle:  6,
		HairColor:  1,
		FacialHair: 4,
		OutfitID:   0,
		Location: player.WorldLocation{
			X:    12,
			Y:    13,
			Z:    14,
			Map:  5,
			Zone: 11,
		},
		BindLocation: player.WorldLocation{},
		Level:        80,
		GuildID:      0,
	}

	p.InitInventory()

	w := wow.NewPacketWriter(0)
	p.WriteToLogin(w)

	fmt.Printf("%s", hex.Dump(w.Bytes()))
}

func TestAssertBytes(t *testing.T) {
	bn := big.NewInt(0)
	bn.SetString("1ef690f864f4402b50df84ba080045000150ca2a4000330646e433b24061c0a800de1f9bc4845a1f701442ae2cd1801801fb4a3000000101080ae09e8be6c9d5dbbc975c312a01beca6700000000064c656374726967677900040b000705020503018d00000001000000331d2146a21d50441fcda544000000600000000200000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008b3100001400000000000000000000000000032700000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000624800001100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", 16)

	fmt.Printf("%s", hex.Dump(bn.Bytes()))

	r := wow.NewPacketReader(bn.Bytes())
	r.ReadNBytes(70) // Header
	var playerCount uint8
	r.Read(&playerCount)

	// data, _ := r.ReadAll()
	// s := base64.StdEncoding.EncodeToString(data)
	// fmt.Printf("%s\n", s)

	var rawGuid uint64

	r.Read(&rawGuid, binary.LittleEndian)
	g := guid.FromRaw(rawGuid)
	assert.Equal(t, 6, g.Counter())
	assert.Equal(t, guid.Player, g.High())

	var p player.Player

	r.ReadString(&p.Name)
	assert.Equal(t, "Lectriggy", p.Name)

	r.Read(&p.Race)
	r.Read(&p.Class)
	r.Read(&p.Gender)

	r.Read(&p.Skin)
	r.Read(&p.Face)
	r.Read(&p.HairStyle)
	r.Read(&p.HairColor)
	r.Read(&p.FacialHair)

	r.Read(&p.Level)

	r.Read(&p.Location.Zone)
	r.Read(&p.Location.Map)

	r.Read(&p.Location.X)
	r.Read(&p.Location.Y)
	r.Read(&p.Location.Z)

	r.Read(&p.GuildID)

	r.Read(&p.CharFlags)

	var tmp2 uint8
	r.Read(&tmp2) // First login

	r.Read(&p.Pet)

	p.InitInventory()

	for _, is := range p.Inventory.InventorySlots {
		r.Read(is)
	}

	r.ResetCounter()
	_, _ = r.ReadAll()
	fmt.Println(r.ReadedCount())

	c, _ := os.Create("character.yaml")
	yaml.NewEncoder(c).Encode(p)
	c.Close()

	// assert.Equal(t, 1, playerCount)
}

// 0000   1e f6 90 f8 64 f4 40 2b 50 df 84 ba 08 00 45 00   ....d.@+P.....E.
// 0010   01 50 ca 2a 40 00 33 06 46 e4 33 b2 40 61 c0 a8   .P.*@.3.F.3.@a..
// 0020   00 de 1f 9b c4 84 5a 1f 70 14 42 ae 2c d1 80 18   ......Z.p.B.,...
// 0030   01 fb 4a 30 00 00 01 01 08 0a e0 9e 8b e6 c9 d5   ..J0............
// 0040   db bc 97 5c 31 2a 01 be ca 67 00 00 00 00 06 4c   ...\1*...g.....L
// 0050   65 63 74 72 69 67 67 79 00 04 0b 00 07 05 02 05   ectriggy........
// 0060   03 01 8d 00 00 00 01 00 00 00 33 1d 21 46 a2 1d   ..........3.!F..
// 0070   50 44 1f cd a5 44 00 00 00 60 00 00 00 02 00 00   PD...D...`......
// 0080   00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 0090   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 00a0   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 00b0   00 00 00 8b 31 00 00 14 00 00 00 00 00 00 00 00   ....1...........
// 00c0   00 00 00 00 00 03 27 00 00 07 00 00 00 00 00 00   ......'.........
// 00d0   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 00e0   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 00f0   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 0100   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 0110   00 00 00 00 00 00 62 48 00 00 11 00 00 00 00 00   ......bH........
// 0120   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 0130   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 0140   00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00   ................
// 0150   00 00 00 00 00 00 00 00 00 00 00 00 00 00         ..............
