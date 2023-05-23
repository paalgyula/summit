package packets

import (
	"bytes"
	"encoding/binary"

	"github.com/paalgyula/summit/pkg/wow"
)

// ClientRealmlist packet contains no fields.
type ClientRealmlist struct{}

func (pkt *ClientRealmlist) UnmarshalPacket(bb []byte) error {
	var unk uint32
	return binary.Read(bytes.NewBuffer(bb), binary.LittleEndian, &unk)
}

type RealmFlag uint8

const (
	RealmFlagNone         RealmFlag = iota
	RealmFlagInvalid      RealmFlag = 0x01
	RealmFlagOffline      RealmFlag = 0x02
	RealmFlagSpecifybuild RealmFlag = 0x04 // client will show realm version in RealmList screen in form "RealmName (major.minor.revision.build)"
	RealmFlagUnk1         RealmFlag = 0x08
	RealmFlagUnk2         RealmFlag = 0x10
	RealmFlagNewPlayers   RealmFlag = 0x20
	RealmFlagRecommended  RealmFlag = 0x40
	RealmFlagFull         RealmFlag = 0x80
)

// Realm is information required to send as part of the realmlist.
type Realm struct {
	// realm type (this is second column in Cfg_Configs.dbc)
	Icon uint8
	// flags, if 0x01, then realm locked
	Lock uint8
	// see enum RealmFlags
	Flags RealmFlag
	// Name name of the server
	Name string
	// Address is a network address of the world server
	Address string
	// Population
	Population float32
	// NumCharacters number of characters in server
	NumCharacters uint8
	// Timezone
	Timezone uint8
}

// ServerRealmlist is made up of a list of realms.
type ServerRealmlist struct {
	Realms []Realm
}

// Bytes converts the ServerRealmlist packet to an array of bytes.
func (pkt *ServerRealmlist) MarshalPacket() []byte {
	w := wow.NewPacketWriter()

	w.WriteL(uint8(0x10))
	w.WriteL(uint16(0)) // Size placeholder
	w.WriteL(uint32(0)) // unk

	w.WriteL(uint16(len(pkt.Realms)))

	for _, realm := range pkt.Realms {
		w.WriteL(realm.Icon)
		w.WriteL(realm.Lock)
		w.WriteL(realm.Flags)
		w.WriteString(realm.Name + "\x00")
		w.WriteString(realm.Address + "\x00")
		w.WriteL(realm.Population)
		w.WriteL(realm.NumCharacters)
		w.WriteL(realm.Timezone)
		w.WriteL(uint8(0x2c))
	}

	w.WriteL(uint16(0x0010)) // Terminator

	// Make the real buffer, which has the length at the start.
	bb := w.Bytes()
	bb[1] = uint8(len(w.Bytes()) - 3)

	return bb
}
