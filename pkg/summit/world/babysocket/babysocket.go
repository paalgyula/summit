package babysocket

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	m sync.Mutex

	clients map[string]*socketClient

	server net.Listener
	log    zerolog.Logger

	cp ClientProvider
}

func NewServer(ctx context.Context, socketPath string, cp ClientProvider) (*Server, error) {
	logger := log.With().Ctx(ctx).Str("service", "babysocket").Logger()

	_ = os.Remove(socketPath)

	conn, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("can not create babysocket: %w", err)
	}

	s := Server{
		server:  conn,
		clients: make(map[string]*socketClient, 0),
		log:     logger,
		cp:      cp,
		m:       sync.Mutex{},
	}

	s.Listen()

	return &s, nil
}

func (s *Server) removeClient(id string) {
	s.m.Lock()
	defer s.m.Unlock()

	delete(s.clients, id)
	s.log.Trace().Msgf("client disconnected: %s", id)
}

func (s *Server) addClient(sc *socketClient) {
	s.m.Lock()
	defer s.m.Unlock()

	s.clients[sc.id] = sc
	s.log.Trace().Msgf("client added: %s", sc.id)

	go sc.Listen()
}

func (s *Server) SendToAll(opcode int, data []byte) {
	for _, c := range s.cp.Clients() {
		c.SendPayload(opcode, data)
	}
}

func (s *Server) SendPacketToBabies(source string, opcode int, data []byte) {
	dp := &DataPacket{
		Opcode:  opcode,
		Command: CommandPacket,
		Source:  source,
		Size:    len(data),
		Data:    data,
		Target:  "", // Don't need to specify, sending to all babies ;)
	}

	bb := &bytes.Buffer{}

	if err := gob.NewEncoder(bb).Encode(dp); err != nil {
		panic("encoder error")
	}

	for _, sc := range s.clients {
		_, _ = sc.conn.Write(bb.Bytes())
	}
}

func (s *Server) Listen() {
	go func() {
		for {
			c, _ := s.server.Accept()
			sc := socketClient{
				id:   xid.New().String(),
				conn: c,
				s:    s,
			}

			s.addClient(&sc)
		}
	}()
}
