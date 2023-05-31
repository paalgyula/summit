package wotlk

type ChrClassesEntry struct {
	ID          uint32          `dbc:"offset=0"`
	PowerType   uint32          `dbc:"offset=2"`
	Name        LocalizedString `dbc:"offset=4"`
	SpellFamily uint32          `dbc:"offset=56"`
	Expansion   uint32          `dbc:"offset=59"`
}
