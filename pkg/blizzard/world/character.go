package world

import (
	"encoding/binary"
	"fmt"

	"github.com/paalgyula/summit/pkg/blizzard/world/packets"
	"github.com/paalgyula/summit/pkg/blizzard/world/player"
	"github.com/paalgyula/summit/pkg/wow"
)

func (gc *GameClient) ListCharacters() {

	var players []*player.Player

	players = append(players, &player.Player{
		ID:     151,
		Class:  11,
		Race:   6,
		Gender: 0,

		Face:       4,
		HairStyle:  2,
		HairColor:  0,
		FacialHair: 3,
		OutfitID:   0,

		Name: "Bela",
	})

	pkt := wow.NewPacketWriter()

	// Character list size, this should be replaced
	pkt.WriteB(1)

	for _, p := range players {
		p.WriteToLogin(pkt)
	}

	gc.SendPacket(packets.ServerCharEnum, pkt.Bytes())
}

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

type CharacterCreateResult uint8

func (gc *GameClient) CreateCharacter(data []byte) {
	r := wow.NewPacketReader(data)
	var accName string
	r.ReadString(&accName)

	var request CharacterCreateRequest
	binary.Read(r, binary.BigEndian, &request)

	fmt.Printf("%s %+v\n", accName, request)

	res := []byte{0x00} // OK :)

	gc.SendPacket(packets.ServerCharCreate, res)
}
