package babysocket

import (
	"bytes"
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

func NewServer(socketPath string, cp ClientProvider) (*Server, error) {
	logger := log.With().Str("service", "babysocket").Logger()

	_ = os.Remove(socketPath)

	conn, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("can not create babysocket: %w", err)
	}

	s := Server{
		server:  conn,
		clients: make(map[string]*socketClient),
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

// SendToAll sends a packet to all connected game clients via the ClientProvider.
func (s *Server) SendToAll(opcode int, data []byte) {
	clients := s.cp.Clients()
	if len(clients) == 0 {
		s.log.Trace().Msg("no game clients connected")
		return
	}

	s.log.Debug().
		Int("opcode", opcode).
		Int("clients", len(clients)).
		Msg("sending packet to all game clients")

	for _, c := range clients {
		c.SendPayload(opcode, data)
	}
}

// SendPacketToBabies sends a packet to all connected baby clients except the sender.
func (s *Server) SendPacketToBabies(source string, opcode int, data []byte) {
	dp := &DataPacket{
		Opcode:  opcode,
		Command: CommandPacket,
		Source:  source,
		Size:    len(data),
		Data:    data,
		Target:  "",
	}

	bb := &bytes.Buffer{}

	if err := gob.NewEncoder(bb).Encode(dp); err != nil {
		s.log.Error().Err(err).Msg("failed to encode packet for babies")
		return
	}

	s.m.Lock()
	defer s.m.Unlock()

	for id, sc := range s.clients {
		if id == source {
			continue // Skip the sender
		}

		if _, err := sc.conn.Write(bb.Bytes()); err != nil {
			s.log.Warn().Err(err).Str("id", id).Msg("failed to send to baby client")
		}
	}
}

// SendToBaby sends a packet to a specific baby client by ID.
func (s *Server) SendToBaby(target string, opcode int, data []byte) {
	dp := &DataPacket{
		Opcode:  opcode,
		Command: CommandPacket,
		Source:  "",
		Size:    len(data),
		Data:    data,
		Target:  target,
	}

	bb := &bytes.Buffer{}

	if err := gob.NewEncoder(bb).Encode(dp); err != nil {
		s.log.Error().Err(err).Msg("failed to encode packet for baby client")
		return
	}

	s.m.Lock()
	defer s.m.Unlock()

	sc, ok := s.clients[target]
	if !ok {
		s.log.Warn().Str("target", target).Msg("baby client not found")
		return
	}

	if _, err := sc.conn.Write(bb.Bytes()); err != nil {
		s.log.Warn().Err(err).Str("id", target).Msg("failed to send to baby client")
	}
}

// Listen starts accepting connections on the Unix socket.
func (s *Server) Listen() {
	go func() {
		for {
			c, err := s.server.Accept()
			if err != nil {
				s.log.Error().Err(err).Msg("failed to accept connection")
				continue
			}

			sc := socketClient{
				id:   xid.New().String(),
				conn: c,
				s:    s,
			}

			s.addClient(&sc)
		}
	}()
}

// Close shuts down the server and closes all connections.
func (s *Server) Close() error {
	s.m.Lock()
	defer s.m.Unlock()

	for id, sc := range s.clients {
		sc.conn.Close()
		delete(s.clients, id)
	}

	return s.server.Close()
}
