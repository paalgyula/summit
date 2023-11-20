package auth

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	authv1 "github.com/paalgyula/summit/pkg/pb/proto/auth/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type Server struct {
	logonListener net.Listener
	rpcListener   net.Listener

	// The realm provider
	realmProvider RealmProvider

	// store store.AccountRepository
	management ManagementService

	log zerolog.Logger
}

type Session struct {
	// AccountName name of the account.
	AccountName string

	// SessionKey is a 40bytes long salt encoded as hex string
	SessionKey string

	// Session creation time
	CreatedAt time.Time
}

func (as *Server) Run() {
	for {
		conn, err := as.logonListener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "closed network connection") {
				// Do not log, we are closed the client
				return
			}

			log.Error().Err(err).Msgf("listener error")

			return
		}

		NewAuthConnection(
			conn, as.realmProvider, as.management,
		)
	}
}

func (as *Server) Close() error {
	//nolint:wrapcheck
	return as.logonListener.Close()
}

func NewServerListener(l net.Listener, management ManagementService, opts ...ServerOption) (*Server, error) {
	as := &Server{
		logonListener: l,
		rpcListener:   nil, // initializing later
		management:    management,
		realmProvider: &StaticRealmProvider{
			RealmList: make([]*Realm, 0),
		},

		log: log.With().Str("service", "auth").Logger(),
	}

	for _, opt := range opts {
		opt(as)
	}

	go as.Run()
	go as.StartManagementServer()

	as.log.Info().Msgf("auth server is listening on: %s", l.Addr().String())

	return as, nil
}

func NewServer(listenAddress string, management ManagementService, opts ...ServerOption) (*Server, error) {
	logonListener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return nil, fmt.Errorf("logonListener: %w", err)
	}

	return NewServerListener(logonListener, management, opts...)
}

// StartManagementServer starts the RPC server if the listener is defined for.
// When the rpcListener is empty, the management server will not be enabled.
// The listener can be set in the NewServer() constructor with the
// WithManagement() ServerOption.
func (s *Server) StartManagementServer() {
	if s.rpcListener == nil {
		s.log.Info().Msg("management server is not enabled")

		return
	}

	srv := grpc.NewServer()
	authv1.RegisterAuthManagementServer(srv, &managementRPCServer{
		srv: s.management,
	})

	s.log.Info().Msgf("management server listening at %v", s.rpcListener.Addr())
	if err := srv.Serve(s.rpcListener); err != nil {
		log.Fatal().Err(err).Msgf("failed to serve: %v", err)
	}
}

// ReadBytes will read a specified number of bytes from a given buffer. If not all
// of the data is read (or there was an error), an error will be returned.
func ReadBytes(buffer io.Reader, length int) ([]byte, error) {
	data := make([]byte, length)

	if length > 0 {
		n, err := buffer.Read(data)
		if err != nil {
			return nil, fmt.Errorf("error while reading bytes: %w", err)
		}

		if n != length {
			log.Trace().Msgf("WTF: %s\n", hex.Dump(data[:n]))

			return nil, fmt.Errorf("%w: wanted %v bytes, got %v", ErrShortRead, length, n)
		}
	}

	return data, nil
}
