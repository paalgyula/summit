//nolint:godox
package packets

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/wow"
)

// TODO: #1 this file should be generated

//nolint:gochecknoinits
func init() {
	OpcodeTable = make(Opcodes, int(wow.NumMsgTypes))

	for i := 0; i < int(wow.NumMsgTypes); i++ {
		OpcodeTable[i] = &Handler{
			Name:    fmt.Sprintf("%v", wow.OpCode(i)),
			State:   STATUS_NEVER,
			Handler: "none",
		}
	}
}

// Correspondence between opcodes and their names.
var OpcodeTable = Opcodes{}
