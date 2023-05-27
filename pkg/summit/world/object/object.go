package object

import "bytes"

type UpdateData bytes.Buffer

type Object struct {
	guid GUID

	UpdateMask *UpdateMask
}

func (o *Object) Update() {

}
