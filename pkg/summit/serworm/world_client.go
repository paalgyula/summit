package serworm

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const defaultAddonInfo = `9e020000789c75d2c16ac3300cc671ef2976e99becb4b450c2eacbe29e8b627f4b446c39384eb7f63dfabe65b70d94f34f48f047afc69826f2fd4e255cdefdc8b82241eab9352fe97b7732ffbc404897d557cea25a43a54759c63c6f70ad115f8c182c0b279ab52196c032a80bf61421818a4639f5544f79d834879faae001fd3ab89ce3a2e0d1ee47d20b1d6db7962b6e3ac6db3ceab2720c0dc9a46a2bcb0caf1f6c2b5297fd84ba95c7922f59954fe2a082fb2daadf739c60496880d6dbe509fa13b84201ddc4316e310bca5f7b7b1c3e9ee193c88d`

type WorldClient struct {
	crypt *crypt.WowCrypt

	client *world.GameClient

	// serverSeed is a 4 byte long random number
	serverSeed []byte

	input *wow.PacketReader
	conn  net.Conn
	log   zerolog.Logger
}

func (wc *WorldClient) readHader() (wow.OpCode, int, error) {
	header, err := wc.input.ReadNBytes(4)
	if err != nil {
		return 0, -1, fmt.Errorf("cannot read header: %w", err)
	}

	if wc.crypt != nil {
		header = wc.crypt.Decrypt(header)
	}

	r := wow.NewPacketReader(header)

	var length uint16
	// Get the length first
	if err := r.ReadB(&length); err != nil {
		return 0, -1, fmt.Errorf("error while reading packet length: %w", err)
	}

	var opcode uint16
	// Then read the opcode
	if err := r.ReadL(&opcode); err != nil {
		return 0, -1, fmt.Errorf("error while reading opcode: %w", err)
	}

	wc.log.Trace().Msgf("world: decoded opcode: %02x, %v len: %d encrypted: %t",
		opcode, wow.OpCode(opcode), length, wc.crypt != nil)

	return wow.OpCode(opcode), int(length) - 2, nil
}

func (wc *WorldClient) listen() {
	for {
		oc, size, err := wc.readHader()
		if err != nil {
			wc.log.Error().Err(err).Msg("cannot read from server")

			_ = wc.client.Close()

			return
		}

		// read the packets
		wc.handlePacket(oc, size)
	}
}

func (wc *WorldClient) handlePacket(oc wow.OpCode, size int) {
	switch oc {
	case wow.ServerAuthChallenge:
		wc.log.Trace().Msgf("Handling server auth challenge %d of auth challenge", size)
		bb, _ := wc.input.ReadNBytes(size)
		r := wow.NewPacketReader(bb)

		var placeholder uint32
		_ = r.Read(&placeholder) // Should be the placeholder, always 1

		// ? handle the error here
		// if placeholder != 1 {
		// }

		wc.serverSeed, _ = r.ReadNBytes(4)
		wc.log.Error().Msgf("seed of the auth: 0x%x", wc.serverSeed)

		// * the encrypt keys are unused yet
		encryptKeys := make([]uint8, 32)
		if err := r.Read(encryptKeys); err != nil {
			wc.log.Error().Err(err).Msg("cannot read new encryption seed from auth challenge")
		}

		// !
		// ! Save the login session key from original server to calculate the proof here!
		// !

		cs := make([]byte, 4)
		_, _ = rand.Read(cs)

		// * generate session proof for login
		digest := crypt.AuthSessionProof(
			wc.client.AccountName(),
			wc.serverSeed,
			cs,
			wc.client.SessionKey(),
		)

		addonInfo, _ := hex.DecodeString(defaultAddonInfo)
		cas := world.ClientAuthSessionPacket{
			ClientBuild:     12340,
			ServerID:        0x0,
			AccountName:     wc.client.AccountName(),
			LoginServerType: 0,
			ClientSeed:      cs,
			RegionID:        0x00,
			BattleGroupID:   0x00,
			RealmID:         0,
			DOSResponse:     0,
			Digest:          digest,
			AddonInfo:       addonInfo,
		}

		wc.conn.Write(cas.Bytes())

	default:
		wc.log.Error().Msgf("received packet: %s with len: %d but not handled", oc.String(), size)
	}
}

func NewWorldClient(gc *world.GameClient, worldAddress string) (*WorldClient, error) {
	conn, err := net.Dial("tcp4", worldAddress)
	if err != nil {
		return nil, fmt.Errorf("world client: %w", err)
	}

	wc := &WorldClient{
		crypt:  nil,
		client: gc,
		input:  wow.NewConnectionReader(conn),
		conn:   conn,
		log:    log.With().Str("service", "world-client").Logger(),
	}

	go wc.listen()

	return wc, nil
}
