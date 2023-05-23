package packets

type LoginPacket interface {
	UnmarshalPacket([]byte) error
}

type RData struct {
	Command AuthCmd
	Data    []byte
}

func (r *RData) Unmarshal(lp LoginPacket) {
	lp.UnmarshalPacket(r.Data)
}
