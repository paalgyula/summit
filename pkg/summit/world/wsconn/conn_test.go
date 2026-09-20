package wsconn_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/paalgyula/summit/pkg/summit/world/wsconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func dialPair(t *testing.T) (*wsconn.Conn, *websocket.Conn) {
	t.Helper()
	serverCh := make(chan *wsconn.Conn, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		serverCh <- wsconn.NewConn(ws)
	}))
	t.Cleanup(s.Close)

	clientWS, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { clientWS.Close() })

	select {
	case srv := <-serverCh:
		return srv, clientWS
	case <-time.After(2 * time.Second):
		t.Fatal("no server connection")
		return nil, nil
	}
}

func TestConnIsAByteStream(t *testing.T) {
	srv, client := dialPair(t)

	// Frames are concatenated; a read spanning two frames blocks until both arrived
	require.NoError(t, client.WriteMessage(websocket.BinaryMessage, []byte{1, 2, 3}))
	require.NoError(t, client.WriteMessage(websocket.BinaryMessage, []byte{4, 5}))
	buf := make([]byte, 5)
	n, err := srv.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, buf)

	// A partial read leaves the remainder buffered
	require.NoError(t, client.WriteMessage(websocket.BinaryMessage, []byte{9, 8, 7}))
	one := make([]byte, 1)
	_, err = srv.Read(one)
	require.NoError(t, err)
	assert.Equal(t, byte(9), one[0])
	two := make([]byte, 2)
	_, err = srv.Read(two)
	require.NoError(t, err)
	assert.Equal(t, []byte{8, 7}, two)

	// Writes go out as binary frames
	_, err = srv.Write([]byte{0xAA, 0xBB})
	require.NoError(t, err)
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, frame, err := client.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.BinaryMessage, msgType)
	assert.Equal(t, []byte{0xAA, 0xBB}, frame)
}

func TestConnRejectsTextFrames(t *testing.T) {
	srv, client := dialPair(t)
	require.NoError(t, client.WriteMessage(websocket.TextMessage, []byte(`{"type":"login"}`)))
	_, err := srv.Read(make([]byte, 1))
	assert.ErrorIs(t, err, wsconn.ErrTextFrame)
}
