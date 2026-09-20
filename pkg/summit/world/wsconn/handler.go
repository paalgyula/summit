package wsconn

import (
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RealmPath is the world WebSocket route; behind the ingress every realm is
// reached as <public url>/realms/<slug>. The connection carries the native
// world protocol: SMSG_AUTH_CHALLENGE, CMSG_AUTH_SESSION, CMSG_CHAR_ENUM,
// CMSG_PLAYER_LOGIN and everything after, with RC4 header encryption.
const RealmPath = "/realms/:slug"

// SessionFactory starts a native world session on an accepted connection.
type SessionFactory interface {
	NewWorldSessionConn(conn net.Conn)
}

// Handler upgrades browser connections and hands them to the world server
// exactly like an accepted TCP connection.
type Handler struct {
	sessions SessionFactory
	upgrader websocket.Upgrader
	log      zerolog.Logger
}

// NewHandler creates the realm WebSocket handler.
func NewHandler(sessions SessionFactory) *Handler {
	return &Handler{
		sessions: sessions,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // the web client lives on a different origin in development
			},
		},
		log: log.With().Str("service", "realm-ws").Logger(),
	}
}

// HandleWS is the Echo route handler for the realm endpoint.
func (h *Handler) HandleWS(c echo.Context) error {
	ws, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to upgrade websocket")
		return err
	}

	h.log.Info().
		Str("realm", c.Param("slug")).
		Str("addr", ws.RemoteAddr().String()).
		Msg("web client connected to realm")

	// The session owns the connection from here and closes it on disconnect
	h.sessions.NewWorldSessionConn(NewConn(ws))
	return nil
}
