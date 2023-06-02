package serworm

import (
	"net"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Bridge struct {
	server string
	user   string
	pass   string

	client *world.GameClient

	socket    *RealmClient
	worldConn net.Conn

	crypt *crypt.WowCrypt
	srp   *crypt.SRP6

	log zerolog.Logger
}

// HandleExternalPacket handles an external packet received from the client by writing
// it to the packet dumper and sending the packet to the bridge.
//
// client: the game client sending the packet.
// oc: the op opcode of the packet.
// data: the data block of the packet.
func (b *Bridge) HandleExternalPacket(client *world.GameClient, oc wow.OpCode, data []byte) {
	wow.GetPacketDumper().Write(oc, data)

	b.SendPacket(oc, data)
}

func (b *Bridge) SetGameClient(gc *world.GameClient) {
	b.client = gc
}

func (b *Bridge) SendPacket(oc wow.OpCode, data []byte) {
	w := wow.NewPacket(oc)

	w.WriteB(uint16(len(data) + 2))
	w.Write(uint16(oc))
	header := w.Bytes()

	if b.crypt != nil {
		header = b.crypt.Encrypt(w.Bytes())
	} else {
		b.log.Error().Msg("no encryption?!")
	}

	data = append(header, data...)

	// TODO: send to server
}

func (b *Bridge) setup() {
	host, _, err := net.SplitHostPort(b.server)
	if err != nil {
		panic(err)
	}

	b.log = b.log.With().Str("host", host).Logger()
	loginConn, err := net.Dial("tcp4", b.server)
	if err != nil {
		panic(err)
	}

	client := NewRealmClient(loginConn, 0x08)
	client.Authenticate(b.user, b.pass)
}

func NewBridge(logonServer, user, pass string) *Bridge {
	b := &Bridge{
		server: logonServer,
		user:   user,
		pass:   pass,

		log: log.With().Str("service", "proxy").Logger(),
	}

	b.setup()

	return b
}
