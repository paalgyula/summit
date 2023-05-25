package player

import (
	"math"

	"github.com/paalgyula/summit/pkg/blizzard/world/guid"
	"github.com/paalgyula/summit/pkg/wow"
)

type PlayerClass uint8

// TODO: generate this from dbc?
const (
	ClassWarior PlayerClass = 0x00
	ClassTauren PlayerClass = 0x06
)

type PlayerRace uint8

const (
	RaceHuman   PlayerRace = 0x00
	RaceWarlock PlayerRace = 0x11
)

type PlayerGender uint8

const (
	GenderMale   = 0x00
	GenderFemale = 0x01
)

type Player struct {
	ID     uint32
	Name   string
	Race   PlayerRace
	Class  PlayerClass
	Gender PlayerGender

	Skin       uint8
	Face       uint8
	HairStyle  uint8
	HairColor  uint8
	FacialHair uint8
	OutfitID   uint8
}

func (p *Player) GUID() *guid.GUID {
	return guid.NewPlayerGUID(p.ID)
}

func (p *Player) WriteToLogin(w *wow.PacketWriter) {
	// 	*data << guid;
	w.WriteB(p.GUID().RawValue())
	// *data << fields[1].Get<std::string>();                          // name
	w.WriteString(p.Name)
	// *data << uint8(plrRace);                                 // race
	w.WriteL(uint8(p.Race))
	// *data << uint8(plrClass);                                // class
	w.WriteL(uint8(p.Class))
	// *data << uint8(gender);                                  // gender
	w.WriteL(uint8(p.Gender))

	// *data << uint8(skin);
	w.WriteL(uint8(p.Skin))
	// *data << uint8(face);
	w.WriteL(uint8(p.Face))
	// *data << uint8(hairStyle);
	w.WriteL(uint8(p.HairStyle))
	// *data << uint8(hairColor);
	w.WriteL(uint8(p.HairColor))
	// *data << uint8(facialStyle);
	w.WriteL(uint8(p.FacialHair))

	// *data << uint8(fields[10].Get<uint8>());                   // level
	w.WriteL(uint8(1))

	// *data << uint32(zone);                                   // zone
	w.WriteL(uint32(0))
	// *data << uint32(fields[12].Get<uint16>());                 // map
	w.WriteL(uint32(530))

	// *data << fields[13].Get<float>();                          // x
	bits := math.Float32bits(1.1)

	w.WriteL(bits)
	// *data << fields[14].Get<float>();                          // y
	w.WriteL(bits)
	// *data << fields[15].Get<float>();                          // z
	w.WriteL(bits)

	// *data << uint32(fields[16].Get<uint32>());                 // guild id
	w.WriteL(uint32(0))

	// Character flags
	// *data << uint32(charFlags);                              // character flags
	w.WriteL(uint32(0))

	// First login
	// *data << uint8(atLoginFlags & AT_LOGIN_FIRST ? 1 : 0);
	w.WriteL(uint8(0))

	// PET section
	// *data << uint32(petDisplayId);
	w.WriteL(uint32(0))
	// *data << uint32(petLevel);
	w.WriteL(uint32(0))
	// *data << uint32(petFamily);
	w.WriteL(uint32(0))

	for i := 0; i < InventorySlotBagEnd; i++ {
		// *data << uint32(proto->DisplayInfoID);
		w.WriteL(uint32(0))
		// *data << uint8(proto->InventoryType);
		w.WriteL(uint8(0))
		// *data << uint32(enchant ? enchant->aura_id : 0);
		w.WriteL(uint32(0))
	}

	// Yipeee
}
