package world

import (
	"time"

	"github.com/paalgyula/summit/pkg/summit/auth"
)

// DefaultRealmReportInterval is how often the world server reports its status
// to the auth server's realm list when the auth server does not say otherwise.
const DefaultRealmReportInterval = 10 * time.Second

// RealmIdentity describes this world server's entry in the realm list.
type RealmIdentity struct {
	Name       string
	Slug       string
	Address    string // direct address (host:port) of the world WebSocket listener
	MaxPlayers uint32
	Icon       uint8
	Timezone   uint8
}

// WithRealmIdentity makes the world server register itself in the auth
// server's realm list and keep its status (online players, population) fresh.
func WithRealmIdentity(id RealmIdentity) ServerOption {
	return func(s *Server) error {
		s.realm = &id
		return nil
	}
}

// realmStatus is the current realm-list entry for this world server.
func (ws *Server) realmStatus() *auth.Realm {
	return &auth.Realm{
		Icon:          ws.realm.Icon,
		Name:          ws.realm.Name,
		Slug:          ws.realm.Slug,
		Address:       ws.realm.Address,
		OnlinePlayers: uint32(ws.GetOnlinePlayerCount()),
		MaxPlayers:    ws.realm.MaxPlayers,
		Timezone:      ws.realm.Timezone,
		Lock:          ws.realmLock,
	}
}

// reportRealmStatus is the heartbeat loop: it registers the realm at start-up
// and refreshes it well inside the deadline the auth server hands back.
// Without a realm identity or a management connection there is nothing to do.
func (ws *Server) reportRealmStatus() {
	if ws.realm == nil || ws.authManagement == nil {
		return
	}

	interval := DefaultRealmReportInterval
	for {
		ttl, err := ws.authManagement.UpdateRealm(ws.realmStatus())
		if err != nil {
			ws.log.Warn().Err(err).Msg("realm status report failed; retrying")
		} else if ttl > 0 && ttl/3 < interval {
			interval = ttl / 3
		}

		select {
		case <-ws.done:
			return
		case <-time.After(interval):
		}
	}
}

// LockRealm toggles the realm-list lock flag (no new logins); reported on the next heartbeat.
func (ws *Server) LockRealm(locked bool) {
	if locked {
		ws.realmLock = 1
	} else {
		ws.realmLock = 0
	}
}
