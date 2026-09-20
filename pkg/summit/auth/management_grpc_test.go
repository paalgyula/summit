package auth

import (
	"net"
	"testing"

	authv1 "github.com/paalgyula/summit/pkg/pb/proto/auth/v1"
	"github.com/paalgyula/summit/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type memAccounts struct{ accounts map[string]*store.Account }

func (m *memAccounts) FindAccount(name string) *store.Account { return m.accounts[name] }
func (m *memAccounts) CreateAccount(a *store.Account) error {
	m.accounts[a.Name] = a
	return nil
}

// startManagement serves the management API on a random port with the given token.
func startManagement(t *testing.T, ms ManagementService, token string) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.UnaryInterceptor(managementAuthInterceptor(token)))
	authv1.RegisterAuthManagementServer(srv, &managementRPCServer{srv: ms})
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(srv.Stop)
	return l.Addr().String()
}

func TestManagementOverGRPC(t *testing.T) {
	ms := NewManagementService(&memAccounts{accounts: map[string]*store.Account{}})
	ms.AddSession(&Session{AccountName: "TEST", SessionKey: "cafe"})
	addr := startManagement(t, ms, "s3cret")

	// A world server with the right token validates sessions and registers its realm
	client, err := NewManagementClient(addr, "s3cret")
	require.NoError(t, err)
	defer client.Close()

	sess := client.GetSession("TEST")
	require.NotNil(t, sess)
	assert.Equal(t, "cafe", sess.SessionKey)
	assert.Nil(t, client.GetSession("NOBODY"))

	ttl, err := client.UpdateRealm(&Realm{Name: "The Highest Summit", Slug: "highest-summit", Address: "world:5002", OnlinePlayers: 3, MaxPlayers: 100})
	require.NoError(t, err)
	assert.Equal(t, DefaultRealmTTL, ttl)

	realms, _ := ms.RealmRegistry().Realms("TEST")
	require.Len(t, realms, 1)
	assert.True(t, realms[0].Online)
	assert.EqualValues(t, 3, realms[0].OnlinePlayers)
	assert.InDelta(t, 0.03, realms[0].Population, 0.001)

	// Without the token nothing is served: no session leaks, no realm registration
	rogue, err := NewManagementClient(addr, "")
	require.NoError(t, err)
	defer rogue.Close()
	assert.Nil(t, rogue.GetSession("TEST"))
	_, err = rogue.UpdateRealm(&Realm{Name: "Evil"})
	assert.Error(t, err)
}
