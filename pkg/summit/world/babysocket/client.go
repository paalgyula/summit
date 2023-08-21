package babysocket

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
)

var ErrNotImplemented = errors.New("not implemented")

type Client struct {
	client        net.Conn
	packetHandler OnPacketFunc

	writeLock sync.Mutex
}

type (
	OnDataFunc   func()
	OnPacketFunc func(string, int, []byte)
)

func NewClient(addr ...string) (*Client, error) {
	socketPath := "babysocket"
	if len(addr) > 0 {
		socketPath = addr[0]
	}

	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to babysocket: %w", err)
	}

	//nolint:exhaustruct
	return &Client{
		client: c,
	}, nil
}

func (c *Client) readData() error {
	var dp DataPacket

	if err := gob.NewDecoder(c.client).Decode(&dp); err != nil {
		panic(err)
	}

	switch dp.Command {
	case CommandPacket:
		if c.packetHandler != nil {
			c.packetHandler(dp.Source, dp.Opcode, dp.Data)
		}
	case CommandInstruction:
		return fmt.Errorf("CommandInstruction %w", ErrNotImplemented)
	case CommandResponse:
		return fmt.Errorf("CommandResponse %w", ErrNotImplemented)
	}

	// fmt.Printf("data: %+v\n", dp)

	return nil
}

func (c *Client) SendToAll(opcode int, data []byte) {
	_ = c.send(DataPacket{
		Source:  "",  // Don't need to specify
		Target:  "*", // special target
		Command: CommandPacket,
		Size:    len(data),
		Opcode:  opcode,
		Data:    data,
	})
}

func (c *Client) send(dp DataPacket) error {
	// Prevent simultaneous write
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	//nolint:wrapcheck
	return gob.NewEncoder(c.client).Encode(dp)
}

func (c *Client) Start() {
	go func() {
		for {
			_ = c.readData()
		}
	}()
}

func (c *Client) Close() error {
	//nolint:wrapcheck
	return c.client.Close()
}
