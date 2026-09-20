// Package wsconn adapts a browser WebSocket into the net.Conn the native
// protocol handlers read from, so the web client speaks the exact WoW auth
// and world byte streams over WebSocket frames.
package wsconn

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ErrTextFrame is returned when a client sends a text frame: the protocol is binary only.
var ErrTextFrame = errors.New("wsconn: unexpected text frame")

// Conn wraps a gorilla websocket.Conn as a byte stream net.Conn. Binary
// frames are concatenated into the read stream; every Write becomes one
// frame. Read blocks until the requested number of bytes is available, which
// is what the packet readers (fixed-size headers, then exact payloads) rely on.
type Conn struct {
	ws *websocket.Conn

	readMu  sync.Mutex
	readBuf bytes.Buffer
	readErr error

	writeMu sync.Mutex

	closed  bool
	closeMu sync.Mutex
}

// NewConn creates a new WebSocket connection adapter.
func NewConn(ws *websocket.Conn) *Conn {
	return &Conn{ws: ws}
}

// Read fills b completely from the incoming binary frames.
func (c *Conn) Read(b []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	for c.readBuf.Len() < len(b) {
		if c.readErr != nil {
			return 0, c.readErr
		}

		msgType, r, err := c.ws.NextReader()
		if err != nil {
			c.readErr = err
			return 0, err
		}
		if msgType != websocket.BinaryMessage {
			c.readErr = ErrTextFrame
			return 0, ErrTextFrame
		}

		if _, err := io.Copy(&c.readBuf, r); err != nil {
			c.readErr = err
			return 0, err
		}
	}

	return c.readBuf.Read(b)
}

// Write sends b as one binary frame.
func (c *Conn) Write(b []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.ws.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close closes the underlying websocket connection.
func (c *Conn) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	_ = c.ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return c.ws.Close()
}

func (c *Conn) LocalAddr() net.Addr  { return c.ws.LocalAddr() }
func (c *Conn) RemoteAddr() net.Addr { return c.ws.RemoteAddr() }

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *Conn) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }

var _ net.Conn = (*Conn)(nil)
