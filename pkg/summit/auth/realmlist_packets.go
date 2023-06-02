package auth

import (
	"github.com/paalgyula/summit/pkg/wow"
)

// ClientRealmlistPacket packet contains no fields.
type ClientRealmlistPacket struct {
	Unknown uint32
}

func (pkt *ClientRealmlistPacket) OpCode() RealmCommand {
	return RealmList
}

func (pkt *ClientRealmlistPacket) MarshalPacket() []byte {
	w := wow.NewPacket(wow.OpCode(RealmList))
	w.WriteUint32(0x00) // padding

	return w.Bytes()
}

func (pkt *ClientRealmlistPacket) UnmarshalPacket(bb wow.PacketData) error {
	return bb.Reader().Read(pkt)
	// return binary.Read(bb.Reader(), binary.LittleEndian, &pkt)
}

// ServerRealmlistPacket is made up of a list of realms.
type ServerRealmlistPacket struct {
	Realms []Realm
}

func (pkt *ServerRealmlistPacket) ReadPacket(r *wow.Reader) {
	var size uint16
	r.Read(&size)

	realmPacket, err := r.ReadNBytes(int(size))
	if err != nil {
		panic(err)
	}

	r = wow.NewPacketReader(realmPacket)
	var unused uint32
	var realmCount uint16
	r.Read(&unused)
	r.Read(&realmCount)

	pkt.Realms = make([]Realm, int(realmCount))

	for i := 0; i < int(realmCount); i++ {
		var realm Realm

		r.Read(&realm.Icon)
		r.Read(&realm.Lock)
		r.Read(&realm.Flags)
		r.ReadString(&realm.Name)
		r.ReadString(&realm.Address)
		r.Read(&realm.Population)
		r.Read(&realm.NumCharacters)
		r.Read(&realm.Timezone)
		r.Read(&realm.Unknown)

		pkt.Realms[i] = realm
	}
}

// MarshalPacket converts the ServerRealmlist packet to an array of bytes.
func (pkt *ServerRealmlistPacket) MarshalPacket() []byte {
	w := wow.NewPacket(wow.OpCode(RealmList))

	realmPkt := wow.NewPacket(0)
	realmPkt.Write(uint32(0)) // unk

	realmPkt.Write(uint16(len(pkt.Realms))) // Size placeholder

	for _, realm := range pkt.Realms {
		realmPkt.Write(realm.Icon)
		realmPkt.Write(realm.Lock)
		realmPkt.Write(realm.Flags)
		realmPkt.WriteString(realm.Name)
		realmPkt.WriteString(realm.Address)
		realmPkt.Write(realm.Population)
		realmPkt.Write(realm.NumCharacters)
		realmPkt.Write(realm.Timezone)
		realmPkt.Write(uint8(0x2c)) // TODO: who and why need this?
	}

	realmPkt.Write(uint16(0x0010)) // Terminator

	// Make the real buffer, which has the length at the start.
	w.Write(uint16(len(realmPkt.Bytes()))) // Size of the full packet
	w.WriteBytes(realmPkt.Bytes())

	return w.Bytes()
}
