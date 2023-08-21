package babysocket

import (
	"encoding/gob"
	"fmt"
	"net"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog/log"
)

type socketClient struct {
	conn net.Conn
	s    *Server
	id   string
}

func (sc *socketClient) Listen() {
	defer sc.disconnected()

	var data DataPacket

	dec := gob.NewDecoder(sc.conn)

	for {
		err := dec.Decode(&data)
		if err != nil {
			log.Err(err).Msg("babysocket listener error")

			return
		}

		fmt.Printf("data from baby client: %+v\n", data)

		switch data.Command {
		case CommandPacket:
			if data.Target == "*" {
				fmt.Printf("broadcasting opcode packet: %T\n", wow.OpCode(data.Opcode))
				sc.s.SendToAll(data.Opcode, data.Data)
			}
		}
	}
}

func (sc *socketClient) disconnected() {
	sc.conn.Close()
	sc.s.removeClient(sc.id)
}
