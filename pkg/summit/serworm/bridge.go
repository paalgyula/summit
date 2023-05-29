package serworm

import (
	"encoding/hex"
	"fmt"
	"net"

	"github.com/paalgyula/summit/pkg/summit/auth"
	authPackets "github.com/paalgyula/summit/pkg/summit/auth/packets"
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

	writer    *auth.PacketWriter
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

	b.writer = auth.NewPacketWriter(loginConn, 0x08)

	clp := authPackets.NewClientLoginChallenge(b.user)

	// Send auth challenge
	b.writer.Send(clp)

	fmt.Println("Waiting for response")
	header := make([]byte, 3)
	_, err = loginConn.Read(header)
	if err != nil {
		log.Fatal().Msgf("err: %s - Head: %s", err.Error(), hex.Dump(header))
	}
	fmt.Println("head packet readed")

	// data := make([]byte, 152)
	// _, err = loginConn.Read(data)
	// if err != nil {
	// 	log.Fatal().Err(err).Msgf("Data: %s", hex.Dump(data))
	// }

	// Get srp stuff
	sar := new(authPackets.ServerLoginChallenge)
	fmt.Printf("ServerLoginChallenge - Size: %d\n", sar.ReadPacket(loginConn))

	// Initialize SRP6
	b.srp = crypt.NewSRP6(int64(sar.G), 3, &sar.N)
	b.srp.B = &sar.B
	A := b.srp.GenerateClientPubkey()

	K, M := b.srp.CalculateClientSessionKey(&sar.Salt, &sar.B, b.user, b.pass)

	fmt.Printf("SessionKey: 0x%x\n", K.Text(16))
	fmt.Printf("M1: 0x%x\n", M.Text(16))

	proof := authPackets.ClientLoginProof{
		A:             *A,
		M:             *M,
		CRCHash:       sar.SaltCRC,
		NumberOfKeys:  0,
		SecurityFlags: 0,
	}

	b.writer.Send(proof)

	header = make([]byte, 2)
	_, err = loginConn.Read(header)
	if err != nil {
		log.Fatal().Err(err).Msgf("err: %s - head: %s", err.Error(), hex.Dump(header))
	}
	fmt.Printf("<< %s", hex.Dump(header))

	data := make([]byte, 2)

	_, err = loginConn.Read(data)
	fmt.Printf("%s", hex.Dump(data))
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
