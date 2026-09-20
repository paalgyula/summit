package wsconn_test

import (
	"encoding/binary"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/paalgyula/summit/pkg/summit/world/wsconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWorld stands in for the world server: it answers the accepted
// connection with a raw SMSG_AUTH_CHALLENGE and echoes the first client packet.
type fakeWorld struct {
	got chan []byte
}

func (f *fakeWorld) NewWorldSessionConn(conn net.Conn) {
	go func() {
		defer conn.Close()
		// server header: size (BE, includes opcode) + opcode (LE); payload: 1 + seed(4) + 32
		payload := make([]byte, 37)
		binary.LittleEndian.PutUint32(payload, 1)
		hdr := []byte{0, byte(len(payload) + 2), 0xEC, 0x01}
		_, _ = conn.Write(append(hdr, payload...))

		// client header is 6 bytes: size (BE, includes 4-byte opcode) + opcode (LE u32)
		h := make([]byte, 6)
		if _, err := conn.Read(h); err != nil {
			return
		}
		size := int(binary.BigEndian.Uint16(h[:2])) - 4
		body := make([]byte, size)
		_, _ = conn.Read(body)
		f.got <- append(h, body...)
	}()
}

func TestRealmWebSocketIsANativeStream(t *testing.T) {
	world := &fakeWorld{got: make(chan []byte, 1)}
	e := echo.New()
	e.GET(wsconn.RealmPath, wsconn.NewHandler(world).HandleWS)
	srv := httptest.NewServer(e)
	defer srv.Close()

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/realms/highest-summit", nil)
	require.NoError(t, err)
	defer client.Close()

	// The very first thing on the wire is SMSG_AUTH_CHALLENGE, as on TCP
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, frame, err := client.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.BinaryMessage, msgType)
	assert.Equal(t, uint16(0x1EC), binary.LittleEndian.Uint16(frame[2:4]))

	// A CMSG_AUTH_SESSION-shaped packet split over two frames is reassembled
	pkt := []byte{0, 8, 0xED, 0x01, 0, 0, 0xAA, 0xBB, 0xCC, 0xDD}
	require.NoError(t, client.WriteMessage(websocket.BinaryMessage, pkt[:5]))
	require.NoError(t, client.WriteMessage(websocket.BinaryMessage, pkt[5:]))
	select {
	case got := <-world.got:
		assert.Equal(t, pkt, got)
	case <-time.After(2 * time.Second):
		t.Fatal("world did not receive the packet")
	}
}
