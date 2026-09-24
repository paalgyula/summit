package client

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//nolint:lll
const defaultAddonInfo = `9e020000789c75d2c16ac3300cc671ef2976e99becb4b450c2eacbe29e8b627f4b446c39384eb7f63dfabe65b70d94f34f48f047afc69826f2fd4e255cdefdc8b82241eab9352fe97b7732ffbc404897d557cea25a43a54759c63c6f70ad115f8c182c0b279ab52196c032a80bf61421818a4639f5544f79d834879faae001fd3ab89ce3a2e0d1ee47d20b1d6db7962b6e3ac6db3ceab2720c0dc9a46a2bcb0caf1f6c2b5297fd84ba95c7922f59954fe2a082fb2daadf739c60496880d6dbe509fa13b84201ddc4316e310bca5f7b7b1c3e9ee193c88d`

type WorldClient struct {
	crypt *crypt.WowCrypt

	// cryptEnable is enabled after the CMSG_AUTH_SESSION is sent; from that
	// point on the header of every packet is encrypted with the session key.
	cryptEnable atomic.Bool

	AccountName string
	SessionKey  *big.Int

	// serverSeed is a 4 byte long random number
	serverSeed []byte

	conn net.Conn
	log  zerolog.Logger

	clientMessages chan *wow.Packet
	serverMessages chan ServerMessage

	// ForwardHandler is called for all packets from upstream when set.
	// This enables proxy mode where packets are forwarded instead of handled locally.
	forwardHandler func(opcode wow.OpCode, data []byte)

	closeOnce sync.Once
	closed    chan struct{}

	// startedAt is the reference for the client-side clock used in time sync
	// responses.
	startedAt time.Time

	stateMu    sync.RWMutex
	characters []*CharEnum
	self       *Player

	objectsMu     sync.RWMutex
	objects       map[wow.GUID]*Entity
	selfGUID      wow.GUID
	creatureNames map[uint32]string
	creatureRanks map[uint32]uint32

	// spellFailures remembers the last SpellCastResult per spell so the bot
	// stops retrying spells the server keeps rejecting.
	spellFailures map[uint32]SpellFailure

	// Death / resurrection state.
	dead      atomic.Bool
	deathLoc  player.WorldLocation
	reclaimCh chan struct{}
	deathCh   chan struct{}

	// Loot and combat feedback.
	lootMu       sync.Mutex
	lastLoot     *LootInfo
	lootCh       chan LootInfo
	combatErrors chan string

	// charEnumCh receives the latest character list (buffered, latest wins).
	charEnumCh chan []*CharEnum
	// loginVerifyCh is signalled once SMSG_LOGIN_VERIFY_WORLD arrived.
	loginVerifyCh chan struct{}
	// loginFailedCh carries the status code of SMSG_CHARACTER_LOGIN_FAILED.
	loginFailedCh chan uint8
	// readyCh is signalled once the player entered the world and the initial
	// spellbook and action bar have been received.
	readyCh chan struct{}
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
		conn:        conn,
		serverSeed:  nil, // after the challenge this will be filled
		AccountName: accountName,
		SessionKey:  sk,

		clientMessages: make(chan *wow.Packet),
		serverMessages: make(chan ServerMessage),

		closed:    make(chan struct{}),
		startedAt: time.Now(),

		objects:       make(map[wow.GUID]*Entity),
		creatureNames: make(map[uint32]string),
		creatureRanks: make(map[uint32]uint32),
		spellFailures: make(map[uint32]SpellFailure),

		reclaimCh:    make(chan struct{}, 1),
		deathCh:      make(chan struct{}, 1),
		lootCh:       make(chan LootInfo, 8),
		combatErrors: make(chan string, 16),

		charEnumCh:    make(chan []*CharEnum, 1),
		loginVerifyCh: make(chan struct{}, 1),
		loginFailedCh: make(chan uint8, 1),
		readyCh:       make(chan struct{}, 1),

		log: log.With().
			Str("acc", accountName).
			Str("server", worldAddress).
			Str("service", "world-client").
			Logger(),
	}

	go wc.readServerPackets()
	go wc.opcodeHandler()

	// Start packet sender goroutine
	go wc.packetSender()

	return wc, nil
}

// Closed returns a channel that is closed when the connection is dropped.
func (wc *WorldClient) Closed() <-chan struct{} {
	return wc.closed
}

// isClosed reports whether the connection has already been closed.
func (wc *WorldClient) isClosed() bool {
	select {
	case <-wc.closed:
		return true
	default:
		return false
	}
}

// Disconnect closes the connection to the world server. It is safe to call
// multiple times and from multiple goroutines.
//
//nolint:wrapcheck
func (wc *WorldClient) Disconnect() error {
	var err error

	wc.closeOnce.Do(func() {
		close(wc.closed)
		err = wc.conn.Close()
	})

	return err
}

// SetForwardHandler sets a callback function that will be called for all packets
// from the upstream server. When set, packets are forwarded instead of handled locally.
func (wc *WorldClient) SetForwardHandler(handler func(opcode wow.OpCode, data []byte)) {
	wc.forwardHandler = handler
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
	for {
		select {
		case <-wc.closed:
			return
		case msg := <-wc.serverMessages:
			wc.handleMessage(&msg)
		}
	}
}

// packetSender goroutine which sends out the packets
func (wc *WorldClient) packetSender() {
	for {
		select {
		case <-wc.closed:
			return
		case pkt := <-wc.clientMessages:
			header := wc.makeHeader(pkt.Opcode(), pkt.Len())

			wc.log.Trace().
				Str("opcode", pkt.Opcode().String()).
				Int("size", pkt.Len()).
				Msgf(">> %s", pkt.Opcode().String())

			_, _ = wc.conn.Write(header)
			_, _ = wc.conn.Write(pkt.Bytes())

			if pkt.Opcode() == wow.ClientAuthSession {
				wc.cryptEnable.Store(true)
			}
		}
	}
}

func (wc *WorldClient) readServerPackets() {
	for {
		oc, data, err := wc.readPacket()
		if err != nil {
			if errors.Is(err, io.EOF) {
				wc.log.Info().Msg("world server closed the connection")
			} else if !wc.isClosed() {
				wc.log.Debug().Err(err).Msg("cannot read from server")
			}

			_ = wc.Disconnect()

			return
		}

		select {
		case wc.serverMessages <- ServerMessage{Opcode: oc, Data: data}:
		case <-wc.closed:
			return
		}
	}
}

func (wc *WorldClient) readPacket() (wow.OpCode, []byte, error) {
	header := make([]byte, 4)

	if _, err := io.ReadFull(wc.conn, header); err != nil {
		return 0, nil, fmt.Errorf("cannot read header: %w", err)
	}

	if wc.cryptEnable.Load() {
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
		Msgf("<< %s encrypted: %t", wow.OpCode(opcode).String(), wc.cryptEnable.Load())

	// The size field counts the 2-byte opcode, so the payload is size-2.
	payloadLen := int(length) - 2
	if payloadLen <= 0 {
		return wow.OpCode(opcode), nil, nil
	}

	data := make([]byte, payloadLen)
	if _, err := io.ReadFull(wc.conn, data); err != nil {
		return 0, nil, fmt.Errorf("readPacket: not enough data to read: %w", err)
	}

	return wow.OpCode(opcode), data, nil
}
