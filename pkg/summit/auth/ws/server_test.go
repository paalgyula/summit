package ws_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/auth"
	authws "github.com/paalgyula/summit/pkg/summit/auth/ws"
	"github.com/paalgyula/summit/pkg/summit/client"
	"github.com/paalgyula/summit/pkg/summit/world/wsconn"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAccountRepo struct {
	accounts map[string]*store.Account
}

func (m *mockAccountRepo) FindAccount(name string) *store.Account {
	return m.accounts[strings.ToUpper(name)]
}

func (m *mockAccountRepo) CreateAccount(account *store.Account) error {
	m.accounts[strings.ToUpper(account.Name)] = account
	return nil
}

// The browser speaks the native logon protocol; the Go realm client is that
// same protocol, so it doubles as the test client over the WebSocket stream.
func TestNativeLogonOverWebSocket(t *testing.T) {
	repo := &mockAccountRepo{accounts: map[string]*store.Account{}}
	mgmt := auth.NewManagementService(repo)
	require.NoError(t, mgmt.Register("TEST", "secret", "test@example.com"))
	rp := &auth.StaticRealmProvider{
		RealmList: []*auth.Realm{
			{Name: "The Highest Summit", Slug: "highest-summit", Address: "127.0.0.1:5002", Online: true, NumCharacters: 1},
		},
	}

	authServer := authws.NewServer(mgmt, rp, authws.WithPublicURL("wss://summit.dev.pilab.hu"))
	ts := httptest.NewServer(http.HandlerFunc(authServer.ServeHTTP))
	defer ts.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http")+authws.AuthPath, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Challenge, proof, REALM_LIST with the realm's WebSocket address
	realms, err := client.NewRealmClient(wsconn.NewConn(ws), 12340).Authenticate("TEST", "secret")
	require.NoError(t, err)
	require.Len(t, realms, 1)
	assert.Equal(t, "The Highest Summit", realms[0].Name)
	assert.Equal(t, "wss://summit.dev.pilab.hu/realms/highest-summit", realms[0].Address)
	assert.EqualValues(t, 1, realms[0].NumCharacters)

	// The proof left a session key behind for the world server to verify against
	sess := mgmt.GetSession("TEST")
	require.NotNil(t, sess)
	assert.NotEmpty(t, sess.SessionKey)
}

// A wrong password fails the SRP6 proof: the server answers status 4 and closes.
func TestNativeLogonRejectsWrongPassword(t *testing.T) {
	repo := &mockAccountRepo{accounts: map[string]*store.Account{}}
	mgmt := auth.NewManagementService(repo)
	require.NoError(t, mgmt.Register("TEST", "secret", "test@example.com"))
	ts := httptest.NewServer(http.HandlerFunc(authws.NewServer(mgmt, &auth.StaticRealmProvider{}).ServeHTTP))
	defer ts.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http")+authws.AuthPath, nil)
	require.NoError(t, err)
	conn := wsconn.NewConn(ws)
	defer conn.Close()

	// AUTH_LOGON_CHALLENGE: cmd, protocol version, size, body
	body := auth.NewClientLoginChallenge("TEST", 12340, [3]byte{3, 3, 5}).MarshalPacket()
	hdr := []byte{byte(auth.AuthLoginChallenge), 8, byte(len(body)), byte(len(body) >> 8)}
	_, err = conn.Write(append(hdr, body...))
	require.NoError(t, err)

	cmd := make([]byte, 1)
	_, err = conn.Read(cmd)
	require.NoError(t, err)
	require.Equal(t, byte(auth.AuthLoginChallenge), cmd[0])
	var challenge auth.ServerLoginChallenge
	challenge.ReadPacket(conn)
	require.Equal(t, auth.ChallengeStatusSuccess, challenge.Status)

	srp := crypt.NewSRP6(int64(challenge.G), 3, &challenge.N)
	srp.B = &challenge.B
	A := srp.GenerateClientPubkey()
	_, M := srp.CalculateClientSessionKey(&challenge.Salt, &challenge.B, "TEST", "WRONG")
	proof := auth.ClientLoginProof{A: *A, M: *M, CRCHash: challenge.SaltCRC}
	_, err = conn.Write(append([]byte{byte(auth.AuthLoginProof)}, proof.MarshalPacket()...))
	require.NoError(t, err)

	resp := make([]byte, 2) // cmd + status
	_, err = conn.Read(resp)
	require.NoError(t, err)
	assert.Equal(t, byte(auth.AuthLoginProof), resp[0])
	assert.Equal(t, byte(4), resp[1], "proof must be rejected")
	assert.Nil(t, mgmt.GetSession("TEST"))
}

func TestRealmWebSocketURLBehindIngress(t *testing.T) {
	r := &auth.Realm{Name: "The Highest Summit", Address: "10.0.0.5:5002"}
	assert.Equal(t, "the-highest-summit", r.URLSlug())
	assert.Equal(t, "wss://summit.dev.pilab.hu/realms/the-highest-summit", r.WebSocketURL("wss://summit.dev.pilab.hu/"))
	assert.Equal(t, "ws://10.0.0.5:5002/realms/the-highest-summit", r.WebSocketURL(""))
}
