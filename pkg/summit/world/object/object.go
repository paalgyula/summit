package object

import "github.com/paalgyula/summit/pkg/wow"

// 1973 in MoP? Seems 1326 in wotlk
const dataLength int = int(wow.NumMsgTypes)

type Object struct {
	guid wow.GUID

	UpdateData []wow.Packet
	UpdateMask *UpdateMask
}

func (o *Object) GetGuid() uint64 {
	return uint64(o.guid)
}

func (o *Object) Update() {

}
