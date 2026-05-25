package wotlk

import "encoding/binary"

const numLocales = 16

type StringRef struct {
	Location uint32
	Value    string
}

type LocalizedString struct {
	Locales      []*StringRef
	Flags        uint32
	ClientLocale uint32
}

func (ls LocalizedString) Value() string {
	for _, sr := range ls.Locales {
		if sr != nil {
			return sr.Value
		}
	}

	return ""
}

//nolint:exhaustruct
func CreatesLocalizedString(data []byte) LocalizedString {
	ls := LocalizedString{
		Locales: make([]*StringRef, numLocales),
	}

	for i := range ls.Locales {
		location := binary.LittleEndian.Uint32(data[i*4:])
		if location != 0 {
			//nolint:exhaustruct
			ls.Locales[i] = &StringRef{Location: location}
		}
	}

	return ls
}
