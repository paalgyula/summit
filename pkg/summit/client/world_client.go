package client

import (
	"fmt"
	"io"
	"math/big"
	"net"
	"os"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//nolint:lll
const defaultAddonInfo = `9e020000789c75d2c16ac3300cc671ef2976e99becb4b450c2eacbe29e8b627f4b446c39384eb7f63dfabe65b70d94f34f48f047afc69826f2fd4e255cdefdc8b82241eab9352fe97b7732ffbc404897d557cea25a43a54759c63c6f70ad115f8c182c0b279ab52196c032a80bf61421818a4639f5544f79d834879faae001fd3ab89ce3a2e0d1ee47d20b1d6db7962b6e3ac6db3ceab2720c0dc9a46a2bcb0caf1f6c2b5297fd84ba95c7922f59954fe2a082fb2daadf739c60496880d6dbe509fa13b84201ddc4316e310bca5f7b7b1c3e9ee193c88d`

type WorldClient struct {
	crypt *crypt.WowCrypt

	// Enable after auth session
	cryptEnable bool

	AccountName string
	SessionKey  *big.Int

	// serverSeed is a 4 byte long random number
	serverSeed []byte

	input *wow.PacketReader
	conn  net.Conn
	log   zerolog.Logger

	clientMessages chan *wow.Packet
	serverMessages chan ServerMessage
}

func NewWorldClient(accountName, sessionKey, worldAddress string) (*WorldClient, error) {
	conn, err := net.Dial("tcp", worldAddress)
	if err != nil {
		return nil, fmt.Errorf("world client: %w", err)
	}

	sk, _ := new(big.Int).SetString(sessionKey, 16)
	wowcrypt, _ := crypt.NewClientWoWCrypt(sk, 1024)

	//nolint:exhaustruct
	wc := &WorldClient{
		crypt:       wowcrypt,
		input:       wow.NewConnectionReader(conn),
		conn:        conn,
		serverSeed:  nil, // after the challenge this will be filled
		AccountName: accountName,
		SessionKey:  sk,

		clientMessages: make(chan *wow.Packet),
		serverMessages: make(chan ServerMessage),

		log: log.With().
			Str("acc", accountName).
			Str("server", worldAddress).
			Str("service", "world-client").
			Logger(),

		// client: gc,
	}

	go wc.readServerPackets()
	go wc.opcodeHandler()

	// Start packet sender goroutine
	go wc.packetSender()

	return wc, nil
}

//nolint:wrapcheck
func (wc *WorldClient) Disconnect() error {
	close(wc.clientMessages)
	close(wc.serverMessages)

	return wc.conn.Close()
}

type ServerMessage struct {
	Opcode wow.OpCode
	Data   []byte
}

func (msg *ServerMessage) Reader() *wow.PacketReader {
	return wow.NewPacketReader(msg.Data)
}

// Handle packets (goroutine)
func (wc *WorldClient) opcodeHandler() {
	for msg := range wc.serverMessages {
		wc.handleMessage(&msg)
	}
}

// packetSender goroutine which sends out the packets
func (wc *WorldClient) packetSender() {
	for pkt := range wc.clientMessages {
		header := wc.makeHeader(pkt.Opcode(), pkt.Len())

		wc.log.Trace().
			Str("opcode", pkt.Opcode().String()).
			Int("size", pkt.Len()).
			Msgf(">> %s", pkt.Opcode().String())

		wc.conn.Write(header)
		wc.conn.Write(pkt.Bytes())

		if pkt.Opcode() == wow.ClientAuthSession {
			wc.cryptEnable = true
		}
	}
}

func (wc *WorldClient) readServerPackets() {
	for {
		oc, data, err := wc.readPacket()
		if err != nil {
			if err == io.EOF {
				wc.log.Info().Err(err).Msg("client dropped")

				os.Exit(1)
			}

			// wc.log.Error().Err(err).Msg("cannot read from server")

			// _ = wc.client.Close()

			return
		}

		wc.serverMessages <- ServerMessage{
			Opcode: oc,
			Data:   data,
		}
	}
}

func (wc *WorldClient) readPacket() (wow.OpCode, []byte, error) {
	header, err := wc.input.ReadNBytes(4)
	if err != nil {
		return 0, nil, fmt.Errorf("cannot read header: %w", err)
	}

	if wc.cryptEnable {
		header = wc.crypt.Decrypt(header)
	}

	r := wow.NewPacketReader(header)

	var length uint16
	// Get the length first - BigEndian!!
	if err := r.ReadB(&length); err != nil {
		return 0, nil, fmt.Errorf("error while reading packet length: %w", err)
	}

	var opcode uint16
	// Then read the opcode - LittleEndian
	if err := r.ReadL(&opcode); err != nil {
		return 0, nil, fmt.Errorf("error while reading opcode: %w", err)
	}

	wc.log.Trace().
		Int("size", int(length)).
		Str("opcode", wow.OpCode(opcode).String()).
		Msgf("<< %s encrypted: %t", wow.OpCode(opcode).String(), wc.cryptEnable)

	data := make([]byte, length)

	_, err = wc.input.ReadBytes(data)
	if err != nil {
		return 0, nil, fmt.Errorf("readPacket: not enough data to read: %w", err)
	}

	return wow.OpCode(opcode), data, nil
}
