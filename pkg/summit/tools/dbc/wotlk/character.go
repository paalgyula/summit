package wotlk

type CharStartOutfitEntry struct {
	ID            uint32   `dbc:"offset=0"`
	RaceID        uint8    `dbc:"offset=1"`
	ClassID       uint8    `dbc:"offset=1,byte=1"`
	Gender        uint8    `dbc:"offset=1,byte=2"`
	OutfitID      uint8    `dbc:"offset=1,byte=3"`
	ItemID        []uint32 `dbc:"offset=2,len=12"`
	DisplayItemID []uint32 `dbc:"offset=27,len=12"`
	InventoryType []uint32 `dbc:"offset=52,len=12"`
}

type ChrRacesRec struct {
	RaceID              uint32
	Flags               uint32
	FactionID           uint32
	MaleDisplayId       uint32
	FemaleDisplayId     uint32
	ClientPrefix        uint32
	MountScale          float32
	BaseLanguage        uint32
	CreatureType        uint32
	LoginEffectSpellID  uint32
	CombatStunSpellID   uint32
	ResSicknessSpellID  uint32
	SplashSoundID       uint32
	StartingTaxiNodes   uint32
	ClientFileString    uint32
	CinematicSequenceID uint32
	NameLang            uint32
}
