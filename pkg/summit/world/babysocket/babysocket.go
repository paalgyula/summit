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

type socketClient struct {
	conn net.Conn
	id   string
}

type Server struct {
	m       sync.Mutex
	clients []*socketClient
	server  net.Listener
	log     zerolog.Logger
}

func NewServer(ctx context.Context, socketPath string) (*Server, error) {
	logger := log.With().Str("service", "babysocket").Logger()

	_ = os.Remove(socketPath)

	conn, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("can not create babysocket: %w", err)
	}

	s := Server{
		server:  conn,
		clients: make([]*socketClient, 0),
		log:     logger,
	}

	s.Listen()

	return &s, nil
}

func (s *Server) addClient(sc *socketClient) {
	s.m.Lock()
	defer s.m.Unlock()

	s.clients = append(s.clients, sc)
	s.log.Trace().Msgf("client added: %s", sc.id)
}

func (s *Server) SendPacket(source string, opcode int, data []byte) {
	dp := &DataPacket{
		Command: CommandPacket,
		Source:  source,
		Size:    len(data),
		Data:    data,
	}

	bb := &bytes.Buffer{}
	err := gob.NewEncoder(bb).Encode(dp)
	if err != nil {
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
			}
			s.addClient(&sc)
		}
	}()
}
