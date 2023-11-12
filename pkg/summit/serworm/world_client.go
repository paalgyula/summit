package serworm

import (
	"fmt"
	"net"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type WorldClient struct {
	crypt *crypt.WowCrypt

	client *world.GameClient

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
		wc.log.Error().Msgf("reading %d of auth challenge", size)
		bb, _ := wc.input.ReadNBytes(size)
		r := wow.NewPacketReader(bb)

		var tmp uint32

		_ = r.Read(&tmp) // Should be 1?
		wc.log.Error().Msgf("content of the first auth segment: %d", tmp)

		_ = r.Read(&tmp)
		wc.log.Error().Msgf("seed of the auth: 0x%x", tmp)

		// wc.conn.Write()
		// !
		// ! Create the proof calculation here to send back the CMSG_AUTH_SESSION
		// ! and initialize the packet encrypter
		// !
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
