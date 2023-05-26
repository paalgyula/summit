package player

import (
	"github.com/paalgyula/summit/pkg/summit/world/guid"
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

type WorldLocation struct {
	X, Y, Z float32
	Map     uint32
	Zone    uint32
}

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

	Location     WorldLocation
	BindLocation WorldLocation

	Level uint8

	Inventory Inventory

	GuildID uint32
}

func (p *Player) GUID() *guid.GUID {
	return guid.NewPlayerGUID(p.ID)
}

func (p *Player) WriteToLogin(w *wow.PacketWriter) {
	// 	*data << guid;
	w.Write(p.GUID().RawValue())
	// *data << fields[1].Get<std::string>();                   // name
	w.WriteString(p.Name)
	// *data << uint8(plrRace);                                 // race
	w.Write(p.Race)
	// *data << uint8(plrClass);                                // class
	w.Write(p.Class)
	// *data << uint8(gender);                                  // gender
	w.Write(p.Gender)

	// *data << uint8(skin);
	w.Write(p.Skin)
	// *data << uint8(face);
	w.Write(p.Face)
	// *data << uint8(hairStyle);
	w.Write(p.HairStyle)
	// *data << uint8(hairColor);
	w.Write(p.HairColor)
	// *data << uint8(facialStyle);
	w.Write(p.FacialHair)

	// *data << uint8(fields[10].Get<uint8>());                   // level
	w.Write(p.Level)

	// *data << uint32(zone);                                   // zone
	w.Write(uint32(0))
	// *data << uint32(fields[12].Get<uint16>());                 // map
	w.Write(uint32(530))

	// *data << fields[13].Get<float>();                          // x
	w.Write(p.Location.X)
	// *data << fields[14].Get<float>();                          // y
	w.Write(p.Location.Y)
	// *data << fields[15].Get<float>();                          // z
	w.Write(p.Location.Z)

	// *data << uint32(fields[16].Get<uint32>());                 // guild id
	w.Write(p.GuildID)

	var charFlags uint32 = 0
	charFlags |= 0x00002000

	// Character flags
	// *data << uint32(charFlags);                              // character flags
	w.Write(charFlags)

	// First login
	// *data << uint8(atLoginFlags & AT_LOGIN_FIRST ? 1 : 0);
	w.Write(uint8(0))

	// PET section
	// *data << uint32(petDisplayId);
	w.Write(uint32(0))
	// *data << uint32(petLevel);
	w.Write(uint32(0))
	// *data << uint32(petFamily);
	w.Write(uint32(0))

	for _, item := range p.Inventory.InventorySlots {
		// *data << uint32(proto->DisplayInfoID);
		w.Write(item.DisplayInfoID)
		// *data << uint8(proto->InventoryType);
		w.Write(item.InventoryType)
		// *data << uint32(enchant ? enchant->aura_id : 0);

		// Find out how enchant slots works
		w.Write(uint32(0))
	}

	// Yipeee
}
