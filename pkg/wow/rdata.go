package wow

type LoginPacket interface {
	UnmarshalPacket(PacketData) error
}

type PacketData []byte

func (pd PacketData) Reader() *Reader {
	return NewPacketReader(pd)
}

type RData struct {
	Command uint8
	Data    PacketData
}

func (r *RData) Unmarshal(lp LoginPacket) {
	lp.UnmarshalPacket(r.Data)
}
