package wotlk

import "github.com/paalgyula/summit/pkg/wow"

type CharStartOutfitEntry struct {
	ID            uint32  `dbc:"offset=0"`
	RaceID        uint8   `dbc:"offset=1"`
	ClassID       uint8   `dbc:"offset=1,byte=1"`
	Gender        uint8   `dbc:"offset=1,byte=2"`
	OutfitID      uint8   `dbc:"offset=1,byte=3"`
	ItemID        []int32 `dbc:"offset=2,len=24"`
	DisplayItemID []int32 `dbc:"offset=26,len=24"`
	InventoryType []int32 `dbc:"offset=50,len=24"`
}

type InventorySlot struct {
	ItemID        uint32
	DisplayItemID uint32
	InventoryType wow.InventoryType
}

func (e *CharStartOutfitEntry) GetSlot(id int) *InventorySlot {
	if id < 0 || id >= len(e.ItemID) {
		return nil
	}

	return &InventorySlot{
		ItemID:        uint32(e.ItemID[id]),
		DisplayItemID: uint32(e.DisplayItemID[id]),
		InventoryType: wow.InventoryType(e.InventoryType[id]),
	}
}

type ChrRacesEntry struct {
	RaceID            uint32          `dbc:"offset=1"`
	Flags             uint32          `dbc:"offset=2"`
	FactionID         uint32          `dbc:"offset=3"`
	MaleDisplayID     uint32          `dbc:"offset=5"`
	FemaleDisplayID   uint32          `dbc:"offset=6"`
	BaseLanguage      uint32          `dbc:"offset=8"`
	Name              LocalizedString `dbc:"offset=15"`
	RequiredExpansion uint32          `dbc:"offset=69"`
}
