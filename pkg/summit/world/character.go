package world

import (
	"encoding/binary"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

func (gc *WorldSession) SendCharacterEnum() {
	var players player.Players

	if err := gc.ws.GetCharacters(gc.AccountName, &players); err != nil {
		log.Error().Err(err).Msg("cannot get players from database")
	}

	for _, p := range players {
		p.Init()
	}

	pkt := wow.NewPacket(wow.ServerCharEnum)

	// Character list size, this should be replaced
	_ = pkt.WriteOne(len(players))

	for _, p := range players {
		p.ToCharacterEnum(pkt)
	}

	gc.socket.Send(pkt)
}

type CharacterCreateRequest struct {
	Race       wow.PlayerRace
	Class      wow.PlayerClass
	Gender     wow.PlayerGender
	Skin       uint8
	Face       uint8
	HairStyle  uint8
	HairColor  uint8
	FacialHair uint8
	OutfitID   uint8
}

type CharacterCreateResult uint8

//nolint:godox
func (gc *WorldSession) CreateCharacter(data wow.PacketData) {
	r := wow.NewPacketReader(data)

	var characerName string

	_ = r.ReadString(&characerName)

	var req CharacterCreateRequest

	_ = r.Read(&req, binary.BigEndian)

	pci := basedata.GetInstance().
		LookupCharacterCreateInfo(req.Race, req.Class, req.Gender)

	loc := player.WorldLocation{
		X:    pci.X,
		Y:    pci.Y,
		Z:    pci.Z,
		O:    pci.O,
		Map:  pci.Map,
		Zone: pci.Zone,
	}

	//nolint:exhaustruct
	p := player.Player{
		Name:            characerName,
		Race:            req.Race,
		Class:           req.Class,
		Gender:          req.Gender,
		Skin:            req.Skin,
		Face:            req.Face,
		HairStyle:       req.HairStyle,
		HairColor:       req.HairColor,
		FacialHair:      req.FacialHair,
		OutfitID:        req.OutfitID,
		Location:        loc,
		BindLocation:    loc,
		Level:           1,
		GuildID:         0x200000,
		CharFlags:       0x00,
		Recustomization: 0,
		FirstLogin:      1,
		Pet:             player.Pet{},
	}

	p.InitInventory(pci.Inventory)

	// TODO: error check
	_ = gc.ws.CreateCharacter(gc.AccountName, &p)

	status := []byte{0x00} // OK :)

	pkt := wow.NewPacketWithData(wow.ServerCharCreate, status)

	// Send response, then the new character list
	gc.socket.Send(pkt)
	gc.SendCharacterEnum()
}
