package world

import (
	"fmt"
	"net"

	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
)

type ServerOption func(s *Server) error

// WithEndpoint sets the gameserver listen address. For example: 127.0.0.1:8129
func WithEndpoint(listenAddress string) ServerOption {
	return func(s *Server) error {
		listener, err := net.Listen("tcp", listenAddress)
		if err != nil {
			return fmt.Errorf("world.StartServer: %w", err)
		}

		s.gameListener = listener

		return nil
	}
}

func WithStaticBaseData() ServerOption {
	return func(s *Server) error {
		data, err := basedata.LoadFromFile("summit.dat")
		if err != nil {
			return fmt.Errorf("world.StartServer: %w", err)
		}

		s.baseData = data

		return nil
	}
}

// WithBabySocket can set the babysocket server if needed.
// The babysocket is socket based custom packet handler.
func WithBabySocket() ServerOption {
	return func(s *Server) error {
		bs, err := babysocket.NewServer("babysocket", s)
		if err != nil {
			return fmt.Errorf("world.StartServer: %w", err)
		}

		s.bs = bs

		return nil
	}
}

// WithAuthManagement set management service. There are two types:
//   - auth.ManagementClient: gRPC based (remote)
//   - auth.ManagementService: local implementation
func WithAuthManagement(ms auth.ManagementService) ServerOption {
	return func(s *Server) error {
		s.authManagement = ms

		return nil
	}
}
