package packets

import (
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

//nolint:revive,stylecheck
const (
	STATUS_NEVER            = "never"
	STATUS_LOGGEDIN         = "logged_in"
	STATUS_AUTHED           = "authed"
	STATUS_TRANSFER_PENDING = "pending"
)

type Opcodes []*Handler

func (o Opcodes) Get(code wow.OpCode) *Handler {
	if int(code) > len(o) {
		return nil
	}

	return o[code]
}

func (o Opcodes) Handle(code wow.OpCode, handler any) {
	oc := o.Get(code)

	if oc == nil {
		log.Fatal().Msgf("you should define a handler first for this message: 0x%x", int(code))

		return
	}

	oc.Handler = handler
}

type Packet interface {
	OpCode() int
}

type Handler struct {
	Name  string
	State string
	// Handler func(interfaces.Packet, *system.State)
	Handler any
}
