package packets

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// ClientRealmlist packet contains no fields.
type ClientRealmlist struct {
	Unknown uint32
}

func (pkt *ClientRealmlist) UnmarshalPacket(bb wow.PacketData) error {
	return bb.Reader().Read(pkt)
	// return binary.Read(bb.Reader(), binary.LittleEndian, &pkt)
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

	// Unknown - needs research whats this
	Unknown uint8
}

// ServerRealmlist is made up of a list of realms.
type ServerRealmlist struct {
	Realms []Realm
}

type realmsHeader struct {
	Always10 uint8
	Size     uint16
	Unk2     uint32
}

func (pkt *ServerRealmlist) UnmarshalPacket(data wow.PacketData) {
	r := data.Reader()

	var rh realmsHeader
	r.ReadL(&rh)

	var realm Realm

	r.ReadL(realm.Icon)
	r.ReadL(realm.Lock)
	r.ReadL(realm.Flags)
	r.ReadString(&realm.Name)
	r.ReadString(&realm.Address)
	r.ReadL(realm.Population)
	r.ReadL(realm.NumCharacters)
	r.ReadL(realm.Timezone)
	r.ReadL(realm.Unknown)
}

// MarshalPacket converts the ServerRealmlist packet to an array of bytes.
func (pkt *ServerRealmlist) MarshalPacket() []byte {
	w := wow.NewPacket(int(RealmList))

	w.Write(uint8(0x10))
	w.Write(uint16(0)) // unk
	w.Write(uint32(0)) // unk

	w.Write(uint16(len(pkt.Realms))) // Size placeholder

	for _, realm := range pkt.Realms {
		w.Write(realm.Icon)
		w.Write(realm.Lock)
		w.Write(realm.Flags)
		w.WriteString(realm.Name)
		w.WriteString(realm.Address)
		w.Write(realm.Population)
		w.Write(realm.NumCharacters)
		w.Write(realm.Timezone)
		w.Write(uint8(0x2c)) // TODO: who and why need this?
	}

	w.Write(uint16(0x0010)) // Terminator

	// Make the real buffer, which has the length at the start.
	bb := w.Bytes()
	bb[1] = uint8(len(w.Bytes()) - 3)

	return bb
}
