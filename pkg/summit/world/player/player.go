package player

import (
	"github.com/paalgyula/summit/pkg/summit/world/guid"
	"github.com/paalgyula/summit/pkg/wow"
)

type PlayerClass uint8

// TODO: generate this from dbc?
const (
	ClassWarior      PlayerClass = 0x01
	ClassPaladin     PlayerClass = 0x02
	ClassHunter      PlayerClass = 0x03
	ClassRogue       PlayerClass = 0x04
	ClassPriest      PlayerClass = 0x05
	ClassDeathKnight PlayerClass = 0x06
	ClassShaman      PlayerClass = 0x07
	ClassMage        PlayerClass = 0x08
	ClassWarlock     PlayerClass = 0x09
	ClassDruid       PlayerClass = 0x0b
)

type PlayerRace uint8

const (
	RaceHuman    PlayerRace = 0x01
	RaceDwarf    PlayerRace = 0x03
	RaceNightElf PlayerRace = 0x04
	RaceGnome    PlayerRace = 0x07
	RaceDraenei  PlayerRace = 0x0b
)

type PlayerGender uint8

const (
	GenderMale   PlayerGender = 0x00
	GenderFemale PlayerGender = 0x01
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

	Inventory *Inventory
	GuildID   uint32

	// CharFlags for example dead, and display ghost
	CharFlags uint32

	// Recustomization flags (change name, look, etc)
	// Needs some research
	Recustomization uint32

	FirstLogin uint8 // Boolean, but uint8 :D

	Pet Pet
}

// Initializes an empty inventory
func (p *Player) InitInventory() {
	if p.Inventory != nil {
		return
	}

	p.Inventory = &Inventory{
		InventorySlots: []*InventoryItem{},
	}

	for i := 0; i < InventorySlotBagEnd; i++ {
		p.Inventory.InventorySlots = append(p.Inventory.InventorySlots, &InventoryItem{})
	}
}

func (p *Player) GUID() *guid.GUID {
	return guid.NewPlayerGUID(p.ID)
}

func (p *Player) Init() {
	p.InitInventory()
}

func (p *Player) WriteToLogin(w *wow.PacketWriter) {
	w.Write(p.GUID().RawValue())
	w.WriteString(p.Name)
	w.Write(p.Race)
	w.Write(p.Class)
	w.Write(p.Gender)

	w.Write(p.Skin)
	w.Write(p.Face)
	w.Write(p.HairStyle)
	w.Write(p.HairColor)
	w.Write(p.FacialHair)

	w.Write(p.Level)

	w.Write(p.Location.Zone)
	w.Write(p.Location.Map)

	w.Write(p.Location.X)
	w.Write(p.Location.Y)
	w.Write(p.Location.Z)

	w.Write(p.GuildID)

	// Character flags
	w.Write(p.CharFlags)
	w.Write(p.Recustomization)

	// First login
	// *data << uint8(atLoginFlags & AT_LOGIN_FIRST ? 1 : 0);
	w.Write(p.FirstLogin)

	// Player Pet section
	w.Write(p.Pet.DisplayID)
	w.Write(p.Pet.PetLevel)
	w.Write(p.Pet.PetFamilly)

	for _, slot := range p.Inventory.InventorySlots {
		w.Write(slot.DisplayInfoID)
		w.Write(slot.InventoryType)
		w.Write(slot.EnchantSlot)
	}

	// Yipeee
}
