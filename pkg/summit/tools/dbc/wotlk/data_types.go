package wotlk

import (
	"bytes"
	"encoding/binary"
)

type StringRef struct {
	Location uint32
	Value    string
}

type LocalizedString struct {
	// Always 15 len in wotlk?
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

func CreatesLocalizedString(data []byte) LocalizedString {
	ls := LocalizedString{
		Locales:      make([]*StringRef, 16),
		Flags:        0,
		ClientLocale: 0,
	}

	br := bytes.NewReader(data)

	for i := 0; i < len(ls.Locales); i++ {
		var location uint32
		binary.Read(br, binary.LittleEndian, &location)

		if location != 0 {
			ls.Locales[i] = &StringRef{
				Location: location,
			}
		}
	}

	return ls
}
