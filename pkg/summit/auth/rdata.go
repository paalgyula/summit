package auth

import "github.com/paalgyula/summit/pkg/wow"

type RData struct {
	Command uint8
	Data    wow.PacketData
}

func (r *RData) Unmarshal(lp wow.RealmPacket) {
	lp.UnmarshalPacket(r.Data)
}
