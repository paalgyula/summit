package babysocket

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"
)

type Client struct {
	client        net.Conn
	packetHandler OnPacketFunc

	writeLock sync.Mutex
}

type OnDataFunc func()
type OnPacketFunc func(string, int, []byte)

func NewClient(addr ...string) (*Client, error) {
	socketPath := "babysocket"
	if len(addr) > 0 {
		socketPath = addr[0]
	}

	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to babysocket: %w", err)
	}

	return &Client{
		client: c,
	}, nil
}

func (c *Client) readData() error {
	var dp DataPacket
	err := gob.NewDecoder(c.client).Decode(&dp)
	if err != nil {
		panic(err)
	}

	switch dp.Command {
	case CommandPacket:
		if c.packetHandler != nil {
			c.packetHandler(dp.Source, dp.Opcode, dp.Data)
		}
	default:
		fmt.Printf("data command not handled: %+v\n", dp)
	}

	fmt.Printf("data: %+v\n", dp)
	return nil
}

func (c *Client) SendToAll(opcode int, data []byte) {
	c.send(DataPacket{
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

	return gob.NewEncoder(c.client).Encode(dp)
}

func (c *Client) Start() {
	go func() {
		for {
			c.readData()
		}
	}()
}

func (c *Client) Close() error {
	return c.client.Close()
}
