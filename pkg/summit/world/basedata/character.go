package basedata

import (
	"github.com/paalgyula/summit/pkg/wow"
)

type InventorySlot struct {
	ItemID        int32
	DisplayItemID int32
	InventoryType wow.InventoryType
}

type PlayerCreateInfo struct {
	Race   wow.PlayerRace
	Class  wow.PlayerClass
	Gender wow.PlayerGender

	Map        uint32
	Zone       uint32
	X, Y, Z, O float32

	Inventory []*InventorySlot
}

type RaceClassGenderKey struct {
	Race   wow.PlayerRace
	Class  wow.PlayerClass
	Gender wow.PlayerGender
}

// LookupCharacterCreateInfo looks up for creation info base data.
// Returns nil if the data not found.
func (bd *Store) LookupCharacterCreateInfo(race wow.PlayerRace,
	class wow.PlayerClass, gender wow.PlayerGender,
) *PlayerCreateInfo {
	info := bd.playerCreateInfo[RaceClassGenderKey{
		Race:   race,
		Class:  class,
		Gender: gender,
	}]

	return info
}
