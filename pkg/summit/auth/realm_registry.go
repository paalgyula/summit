package auth

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultRealmTTL is how long a realm stays "online" in the list after its
// last status report.
const DefaultRealmTTL = 30 * time.Second

// RealmRegistry is a RealmProvider fed by world servers reporting their own
// status (UpdateRealm). Statically configured realms are listed too, shown as
// offline until the corresponding world server reports in.
type RealmRegistry struct {
	mu     sync.RWMutex
	static []*Realm
	live   map[string]*liveRealm // by slug
	ttl    time.Duration
	now    func() time.Time
}

type liveRealm struct {
	realm    *Realm
	lastSeen time.Time
}

// NewRealmRegistry creates a registry seeded with statically configured realms.
func NewRealmRegistry(static []*Realm, ttl time.Duration) *RealmRegistry {
	if ttl <= 0 {
		ttl = DefaultRealmTTL
	}
	return &RealmRegistry{
		static: static,
		live:   make(map[string]*liveRealm),
		ttl:    ttl,
		now:    time.Now,
	}
}

// TTL is the heartbeat deadline world servers must meet.
func (rr *RealmRegistry) TTL() time.Duration {
	return rr.ttl
}

// Update records a status report from a world server. Presentation flags
// (recommended, PvP, ...) come from the static configuration of that realm.
func (rr *RealmRegistry) Update(status *Realm) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	copyOf := *status
	copyOf.Online = true
	for _, s := range rr.static {
		if s.URLSlug() == copyOf.URLSlug() {
			copyOf.Flags |= s.Flags
			if copyOf.Icon == 0 {
				copyOf.Icon = s.Icon
			}
			break
		}
	}
	rr.live[status.URLSlug()] = &liveRealm{realm: &copyOf, lastSeen: rr.now()}
}

// Realms implements RealmProvider: live realms first, then static ones that
// have not reported; stale live realms are flagged offline.
func (rr *RealmRegistry) Realms(_ string) ([]*Realm, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	now := rr.now()
	seen := make(map[string]bool, len(rr.live))
	out := make([]*Realm, 0, len(rr.live)+len(rr.static))

	for slug, lr := range rr.live {
		r := *lr.realm
		r.Online = now.Sub(lr.lastSeen) <= rr.ttl
		if !r.Online {
			r.Population = 0
			r.OnlinePlayers = 0
		}
		out = append(out, &r)
		seen[slug] = true
	}
	for _, s := range rr.static {
		if seen[s.URLSlug()] {
			continue
		}
		r := *s
		r.Online = false
		out = append(out, &r)
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}
