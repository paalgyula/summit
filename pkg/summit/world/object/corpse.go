package object

import "github.com/paalgyula/summit/pkg/wow"

// Corpse represents a player corpse in the world.
type Corpse struct {
	*Object
}

// NewCorpse creates a new Corpse with proper type flags.
func NewCorpse() *Corpse {
	obj := NewObject()
	obj.objectTypeID = wow.TypeIDCorpse
	obj.objectType = wow.TypeMaskCorpse

	// Corpse uses stationary position
	obj.AddUpdateFlags(wow.UpdateFlagStationaryPosition)

	return &Corpse{
		Object: obj,
	}
}
