package world

import (
	"encoding/binary"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

func (gc *GameClient) ListCharacters() {
	var players []*player.Player

	if err := gc.acc.Characters(&players); err != nil {
		log.Error().Err(err).Msg("cannot get players from database")
	}

	for _, p := range players {
		p.Init()
	}

	pkt := wow.NewPacket(wow.ServerCharEnum)

	// Character list size, this should be replaced
	_ = pkt.WriteOne(len(players))

	for _, p := range players {
		p.WriteToLogin(pkt)
	}

	gc.Send(pkt)
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
	OutfitID   uint8
}

type CharacterCreateResult uint8

//nolint:godox
func (gc *GameClient) CreateCharacter(data wow.PacketData) {
	r := wow.NewPacketReader(data)

	var characerName string

	_ = r.ReadString(&characerName)

	var req CharacterCreateRequest

	_ = r.Read(&req, binary.BigEndian)

	// fmt.Printf("%s %+v\n", characerName, req)

	// TODO: #2 when the DBC reader is ready this should be re-written
	loc := player.WorldLocation{
		X:           10311.3,
		Y:           832.463,
		Z:           1326.41,
		Map:         1,
		Zone:        141,
		Orientation: 0,
	}

	//nolint:exhaustruct
	p := player.Player{
		Name:            characerName,
		Race:            wow.PlayerRace(req.Race),
		Class:           wow.PlayerClass(req.Class),
		Gender:          wow.PlayerGender(req.Gender),
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

	p.InitInventory()

	var players player.Players

	_ = gc.acc.Characters(&players)
	players.Add(&p)

	gc.acc.UpdateCharacters(players)

	res := []byte{0x00} // OK :)

	gc.SendPayload(int(wow.ServerCharCreate), res)
}
