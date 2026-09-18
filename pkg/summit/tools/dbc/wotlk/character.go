package wotlk

// ChrClassesEntry represents the ChrClasses.dbc file structure.
//
// Format: nxixssssssssssssssssxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxixii
type ChrClassesEntry struct {
	ID               uint32          `dbc:"offset=0"`
	PowerType        uint32          `dbc:"offset=2"`
	Name             LocalizedString `dbc:"offset=5"`
	SpellFamily      uint32          `dbc:"offset=56"`
	CinematicSequence uint32         `dbc:"offset=58"`
	Expansion        uint32          `dbc:"offset=59"`
}

// ChrRacesEntry represents the ChrRaces.dbc file structure.
//
// Format: niixiixixxxxiissssssssssssssssxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxi
type ChrRacesEntry struct {
	RaceID            uint32          `dbc:"offset=0"`
	Flags             uint32          `dbc:"offset=1"`
	FactionID         uint32          `dbc:"offset=2"`
	MaleDisplayID     uint32          `dbc:"offset=4"`
	FemaleDisplayID   uint32          `dbc:"offset=5"`
	BaseLanguage      uint32          `dbc:"offset=8"`
	CinematicSequence uint32          `dbc:"offset=12"`
	Alliance          uint32          `dbc:"offset=13"`
	Name              LocalizedString `dbc:"offset=14"`
	Expansion         uint32          `dbc:"offset=68"`
}

// CharStartOutfitEntry represents the CharStartOutfit.dbc file structure.
//
// Format: dbbbXiiiiiiiiiiiiiiiiiiiiiiiixxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
//
// This struct uses slices for the item arrays since the DBC contains fixed-size
// arrays that the reader converts to slices via dbc tags.
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

// CharTitlesEntry represents the CharTitles.dbc file structure.
//
// Format: nxssssssssssssssssxssssssssssssssssxi
type CharTitlesEntry struct {
	ID         uint32          `dbc:"offset=0"`
	NameMale   LocalizedString `dbc:"offset=2"`
	NameFemale LocalizedString `dbc:"offset=19"`
	BitIndex   uint32          `dbc:"offset=36"`
}
