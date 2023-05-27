package object

import "bytes"

type UpdateData bytes.Buffer

type Object struct {
	guid GUID

	// Flags for update
	updateFlags uint16
}

func (o *Object) IsCorpse() bool {
	return o.guid.High() == Corpse
}

func (o *Object) IsPlayer() bool {
	return o.guid.High() == Player
}

func (*Object) CreateUpdateForPlayer() {

}
