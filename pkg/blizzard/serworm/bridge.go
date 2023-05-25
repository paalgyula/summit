package serworm

import (
	"encoding/hex"
	"fmt"
	"net"

	"github.com/paalgyula/summit/pkg/blizzard/auth"
	authPackets "github.com/paalgyula/summit/pkg/blizzard/auth/packets"
	"github.com/paalgyula/summit/pkg/blizzard/world"
	worldPackets "github.com/paalgyula/summit/pkg/blizzard/world/packets"
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

	writer    *auth.PacketWriter
	worldConn net.Conn

	crypt *crypt.WowCrypt

	log zerolog.Logger
}

func (b *Bridge) HandleExternalPacket(client *world.GameClient, oc worldPackets.OpCode, data []byte) {
	wow.GetPacketDumper().Write(oc.Int(), data)

	b.SendPacket(oc, data)
}

func (b *Bridge) SetGameClient(gc *world.GameClient) {
	b.client = gc
}

func (b *Bridge) SendPacket(oc worldPackets.OpCode, data []byte) {
	w := wow.NewPacketWriter()
	w.WriteB(uint16(len(data) + 2))
	w.WriteL(uint16(oc.Int()))
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

	b.writer = auth.NewPacketWriter(loginConn, 0x08)

	clp := authPackets.NewClientLoginChallenge(b.user)

	// Send auth challenge
	b.writer.Send(int(authPackets.AuthLoginChallenge), clp)
	header := make([]byte, 4)
	_, err = loginConn.Read(header)
	if err != nil {
		log.Fatal().Err(err).Msgf("%s", hex.Dump(header))
	}

	fmt.Printf("%s", hex.Dump(header))
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
