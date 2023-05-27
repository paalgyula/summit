package babysocket

import "github.com/paalgyula/summit/pkg/wow"

type ClientProvider interface {
	Clients() map[string]wow.PayloadSender
}
