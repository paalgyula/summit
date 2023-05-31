package wotlk

type MapEntry struct {
	ID           uint32          `dbc:"offset=0"`
	Directory    *StringRef      `dbc:"offset=1"`
	InstanceType uint32          `dbc:"offset=2"`
	MapName      LocalizedString `dbc:"offset=5"`
}
