// Package ws serves the native WoW logon protocol (SRP6 challenge/proof and
// REALM_LIST) to browsers over a WebSocket, using the same AuthConnection
// handler as the TCP listener.
package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world/wsconn"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// AuthPath is the logon WebSocket route; behind the ingress it is
// <public url>/realm/auth.
const AuthPath = "/realm/auth"

// Server is the Echo-based WebSocket logon server.
type Server struct {
	echo          *echo.Echo
	mgmt          auth.ManagementService
	realmProvider auth.RealmProvider
	upgrader      websocket.Upgrader
	log           zerolog.Logger
	// publicURL is the externally reachable base URL realm endpoints are built
	// from (e.g. "wss://summit.dev.pilab.hu"); empty means direct addresses.
	publicURL string
}

// Option customises the logon WebSocket server.
type Option func(*Server)

// WithPublicURL sets the public base URL realm WebSocket endpoints are advertised under.
func WithPublicURL(u string) Option {
	return func(s *Server) { s.publicURL = u }
}

// NewServer creates a new logon WebSocket server.
func NewServer(mgmt auth.ManagementService, rp auth.RealmProvider, opts ...Option) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodOptions},
		AllowHeaders: []string{"*"},
	}))

	s := &Server{
		echo:          e,
		mgmt:          mgmt,
		realmProvider: rp,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		log: log.With().Str("service", "auth-ws").Logger(),
	}
	for _, o := range opts {
		o(s)
	}

	e.GET(AuthPath, s.handleWS)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "auth-ws"})
	})

	return s
}

// ServeHTTP dispatches requests to the underlying Echo instance.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.echo.ServeHTTP(w, r)
}

// Start starts the logon WebSocket server on the given address.
func (s *Server) Start(addr string) error {
	s.log.Info().Str("listen", addr).Msg("login websocket server starting")
	return s.echo.Start(addr)
}

// Close stops the server.
func (s *Server) Close() error {
	return s.echo.Close()
}

// handleWS upgrades the connection and runs the native logon handler on it.
// The realm list is the native one; only the realm address is rewritten to
// the realm's WebSocket endpoint, since that is how a browser reaches it.
func (s *Server) handleWS(c echo.Context) error {
	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to upgrade auth websocket")
		return err
	}

	s.log.Info().Str("addr", ws.RemoteAddr().String()).Msg("web client connected to logon")
	auth.NewAuthConnection(wsconn.NewConn(ws), &webRealmProvider{inner: s.realmProvider, publicURL: s.publicURL}, s.mgmt)
	return nil
}

// webRealmProvider presents the realm list with WebSocket addresses.
type webRealmProvider struct {
	inner     auth.RealmProvider
	publicURL string
}

func (p *webRealmProvider) Realms(account string) ([]*auth.Realm, error) {
	if p.inner == nil {
		return nil, nil
	}
	realms, err := p.inner.Realms(account)
	if err != nil {
		return nil, err
	}
	out := make([]*auth.Realm, len(realms))
	for i, r := range realms {
		copyOf := *r
		copyOf.Address = r.WebSocketURL(p.publicURL)
		if !r.Online {
			copyOf.Flags |= auth.RealmFlagOffline
		}
		out[i] = &copyOf
	}
	return out, nil
}
