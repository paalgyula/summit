package player_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
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

	p.InitInventory(nil)

	w := wow.NewPacket(0)
	p.ToCharacterEnum(w)

	fmt.Printf("%s", hex.Dump(w.Bytes()))
}

func TestAssertBytes(t *testing.T) {
	bn := big.NewInt(0)
	bn.SetString(strings.Join([]string{
		"1ef690f864f4402b50df84ba080045000150ca2a4000330646e433b24061c0a800",
		"de1f9bc4845a1f701442ae2cd1801801fb4a3000000101080ae09e8be6c9d5dbbc",
		"975c312a01beca6700000000064c656374726967677900040b000705020503018d",
		"00000001000000331d2146a21d50441fcda5440000006000000002000000000100",
		"000000000000000000000000000000000000000000000000000000000000000000",
		"00000000000000000000000000008b310000140000000000000000000000000003",
		"270000070000000000000000000000000000000000000000000000000000000000",
		"000000000000000000000000000000000000000000000000000000000000000000",
		"000000000000000000000000000062480000110000000000000000000000000000",
		"000000000000000000000000000000000000000000000000000000000000000000",
		"0000000000000000000000000000000000000000",
	}, ""), 16)

	fmt.Printf("%s", hex.Dump(bn.Bytes()))

	r := wow.NewPacketReader(bn.Bytes())
	r.ReadNBytes(70) // Header
	var playerCount uint8
	r.Read(&playerCount)

	// data, _ := r.ReadAll()
	// s := base64.StdEncoding.EncodeToString(data)
	// fmt.Printf("%s\n", s)

	var rawGuid uint64

	r.Read(&rawGuid, binary.BigEndian)
	g := wow.GUID(rawGuid)
	assert.EqualValues(t, 6, g.Counter())

	t.Log("Player high GUID test should be fixed")
	// assert.EqualValues(t, wow.PlayerGuid, g.High())

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
	r.Read(&p.Recustomization)

	r.Read(&p.FirstLogin) // First login

	r.Read(&p.Pet)

	p.InitInventory(nil)

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
