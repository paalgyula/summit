package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealmRegistryStatusAndExpiry(t *testing.T) {
	now := time.Unix(1000, 0)
	rr := NewRealmRegistry([]*Realm{{Name: "Static Realm", Slug: "static"}}, 30*time.Second)
	rr.now = func() time.Time { return now }

	// Before any report the static realm is listed but offline
	realms, err := rr.Realms("acc")
	require.NoError(t, err)
	require.Len(t, realms, 1)
	assert.False(t, realms[0].Online)

	// A world server reports in for the static realm and a new one
	rr.Update(&Realm{Name: "Static Realm", Slug: "static", OnlinePlayers: 7, Population: 0.7})
	rr.Update(&Realm{Name: "Fresh Realm", Slug: "fresh", OnlinePlayers: 1})
	realms, _ = rr.Realms("acc")
	require.Len(t, realms, 2)
	byName := map[string]*Realm{}
	for _, r := range realms {
		byName[r.Name] = r
	}
	assert.True(t, byName["Static Realm"].Online)
	assert.EqualValues(t, 7, byName["Static Realm"].OnlinePlayers)
	assert.True(t, byName["Fresh Realm"].Online)

	// Past the TTL without a heartbeat the realm goes offline
	now = now.Add(31 * time.Second)
	realms, _ = rr.Realms("acc")
	for _, r := range realms {
		assert.False(t, r.Online, r.Name)
		assert.EqualValues(t, 0, r.OnlinePlayers, r.Name)
	}
}
