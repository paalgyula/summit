package auth

import (
	"regexp"
	"strings"
)

type RealmProvider interface {
	// Returns a list of realms for the given account.
	Realms(accountID string) ([]*Realm, error)
}

type StaticRealmProvider struct {
	RealmList []*Realm
}

func (srp *StaticRealmProvider) Realms(_ string) ([]*Realm, error) {
	return srp.RealmList, nil
}

type RealmFlags uint8

const (
	RealmFlagNone         RealmFlags = 0x00
	RealmFlagInvalid      RealmFlags = 0x01
	RealmFlagOffline      RealmFlags = 0x02
	RealmFlagSpecifyBuild RealmFlags = 0x04
	RealmFlagUnk1         RealmFlags = 0x08
	RealmFlagUnk2         RealmFlags = 0x10
	RealmFlagNewPlayers   RealmFlags = 0x20
	RealmFlagRecommended  RealmFlags = 0x40
	RealmFlagFull         RealmFlags = 0x80
)

// Realm is information required to send as part of the realmlist.
type Realm struct {
	// realm type (this is second column in Cfg_Configs.dbc)
	Icon uint8
	// flags, if 0x01, then realm locked
	Lock uint8
	// see enum RealmFlags
	Flags RealmFlags
	// Name name of the server
	Name string
	// Slug is the URL-safe realm identifier used by the web client's
	// WebSocket endpoint (/realms/<slug>). Derived from Name when empty.
	Slug string
	// Address is a network address of the world server
	Address string
	// Population indicator (online / capacity)
	Population float32
	// OnlinePlayers currently in the world
	OnlinePlayers uint32
	// MaxPlayers is the capacity the population indicator is relative to
	MaxPlayers uint32
	// Online is false for realms whose world server is not reporting
	Online bool
	// NumCharacters number of characters in server
	NumCharacters uint8
	// Timezone
	Timezone uint8

	// Unknown - needs research whats this
	Unknown uint8
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a realm name into its URL identifier ("The Highest Summit" -> "the-highest-summit").
func Slugify(name string) string {
	return strings.Trim(nonSlugChars.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// URLSlug returns the realm's explicit slug, or one derived from its name.
func (r *Realm) URLSlug() string {
	if r.Slug != "" {
		return r.Slug
	}
	return Slugify(r.Name)
}

// WebSocketURL is the realm's world WebSocket endpoint for web clients.
// With a public base URL (e.g. "wss://summit.dev.pilab.hu") every realm is
// reached through the ingress as <base>/realms/<slug>; otherwise the realm's
// own listen address is used directly.
func (r *Realm) WebSocketURL(publicBaseURL string) string {
	if publicBaseURL != "" {
		return strings.TrimRight(publicBaseURL, "/") + "/realms/" + r.URLSlug()
	}
	if strings.HasPrefix(r.Address, "ws://") || strings.HasPrefix(r.Address, "wss://") {
		return r.Address
	}
	return "ws://" + r.Address + "/realms/" + r.URLSlug()
}
