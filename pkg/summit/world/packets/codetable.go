//nolint:godox
package packets

import (
	"fmt"

	"github.com/paalgyula/summit/pkg/wow"
)

// TODO: #1 this file should be generated

// NewOpcodeTable returns a table with every opcode present and no handler
// installed. Handlers are closures bound to one session, so every
// WorldSession owns a table of its own.
func NewOpcodeTable() Opcodes {
	table := make(Opcodes, int(wow.NumMsgTypes))

	for i := 0; i < int(wow.NumMsgTypes); i++ {
		table[i] = &Handler{
			Name:    fmt.Sprintf("%v", wow.OpCode(i)),
			State:   STATUS_NEVER,
			Handler: "none",
		}
	}

	return table
}
