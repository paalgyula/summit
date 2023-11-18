//nolint:revive
package auth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrShortRead = errors.New("short read when reading opcode data")
	ErrWriteSize = errors.New("the written and sent bytes are not equal")
	ErrNoHandler = errors.New("no handler implemented")
)

type AuthConnection struct {
	c net.Conn

	outLock sync.Mutex

	log zerolog.Logger
	id  string

	rp RealmProvider

	account *store.Account
	mgmt    ManagementService

	srp *crypt.SRP6
}

func NewAuthConnection(c net.Conn, rp RealmProvider,
	management ManagementService,
) *AuthConnection {
	rc := &AuthConnection{
		c:    c,
		log:  log.With().Str("addr", c.RemoteAddr().String()).Logger(),
		mgmt: management,
		id:   xid.New().String(),

		srp: crypt.NewSRP6(7, 3, big.NewInt(0)),
		rp:  rp,

		outLock: sync.Mutex{},
	}

	go rc.listen()

	return rc
}

//nolint:godox
func (rc *AuthConnection) HandleLogin(pkt *ClientLoginChallenge) error {
	res := new(ServerLoginChallenge)

	res.Status = ChallengeStatusSuccess

	// Validate the packet.
	gameName := strings.TrimRight(pkt.GameName, "\x00")
	if gameName != "WoW" {
		// TODO: temporary removed this line to allow every client to log in
		// } else if pkt.Version != static.SupportedGameVersion || pkt.Build != static.SupportedGameBuild {
		// 	res.Status = ChallengeStatusFailVersionInvalid
		res.Status = ChallengeStatusFailed
	} else {
		rc.account = rc.mgmt.
			FindAccount(pkt.AccountName)

		if rc.mgmt == nil {
			res.Status = ChallengeStatusFailUnknownAccount
			rc.c.Close()
		}
	}

	if res.Status == ChallengeStatusSuccess {
		B := rc.srp.GenerateServerPubKey(rc.account.Verifier)

		res.B.Set(B)
		res.Salt.Set(rc.account.Salt)
		res.SaltCRC = make([]byte, 16)

		res.G = uint8(rc.srp.GValue())
		res.N = *rc.srp.N()
	}

	// Send out the packet
	return rc.Send(AuthLoginChallenge, res.MarshalPacket())
}

func (rc *AuthConnection) HandleProof(pkt *ClientLoginProof) error {
	response := new(ServerLoginProof)

	K, M := rc.srp.CalculateServerSessionKey(
		&pkt.A,
		rc.account.Verifier,
		rc.account.Salt,
		rc.account.Name)

	if M.Cmp(&pkt.M) != 0 {
		response.StatusCode = 4
		_ = rc.Send(AuthLoginProof, response.MarshalPacket())
		rc.c.Close()

		return nil
	}

	response.StatusCode = 0
	response.Proof.
		Set(crypt.CalculateServerProof(&pkt.A, M, K))

	rc.log = rc.log.With().
		Str("account", rc.account.Name).
		Logger()

	rc.mgmt.AddSession(&Session{
		AccountName: rc.account.Name,
		SessionKey:  K.Text(16),
		CreatedAt:   time.Now(),
	})

	return rc.Send(AuthLoginProof, response.MarshalPacket())
}

//nolint:godox
func (rc *AuthConnection) HandleRealmList() error {
	rc.log.Debug().Msg("handling realmlist request")

	// TODO: #3 use some protocol to do registration with realm/manage realms and-or offline status
	//nolint:exhaustruct
	srl := ServerRealmlistPacket{}

	realms, err := rc.rp.Realms(rc.account.Name)
	if err != nil {
		return fmt.Errorf("authConnection.HandleRealmList: %w", err)
	}

	srl.Realms = realms

	return rc.Send(RealmList, srl.MarshalPacket())
}

func (rc *AuthConnection) Send(opcode RealmCommand, payload []byte) error {
	size := len(payload)

	rc.log.Debug().
		Str("packet", opcode.String()).
		Str("opcode", fmt.Sprintf("0x%04x", int(opcode))).
		Int("size", size).
		Msg("sending packet to client")

	w := wow.NewPacket(0)
	_ = w.Write(uint8(opcode))
	_, _ = w.WriteBytes(payload)

	return rc.Write(w.Bytes())
}

func (rc *AuthConnection) Write(bb []byte) error {
	rc.outLock.Lock()
	defer rc.outLock.Unlock()

	w, err := rc.c.Write(bb)
	if err != nil {
		return fmt.Errorf("authConnection.Write: %w", err)
	}

	if w != len(bb) {
		return ErrWriteSize
	}

	return nil
}

func (rc *AuthConnection) listen() {
	defer rc.c.Close()
	rc.log.Info().Msgf("accepting messages from a new login connection")

	for {
		// Read packets infinitely :)
		pkt, err := rc.read(rc.c)
		if err != nil || pkt == nil {
			if errors.Is(err, io.EOF) {
				rc.log.Info().Msg("client disconnected from realm")

				return
			}

			rc.log.Error().Err(err).Msg("error while reading from client")

			return
		}

		switch RealmCommand(pkt.Command) {
		case AuthLoginChallenge:
			var clc ClientLoginChallenge

			pkt.Unmarshal(&clc)

			rc.log.Trace().Msgf(">> WoW -> Auth ClientLoginChallenge")

			_ = rc.HandleLogin(&clc)
		case AuthLoginProof:
			var clp ClientLoginProof

			pkt.Unmarshal(&clp)

			rc.log.Trace().Msgf(">> WoW -> Auth ClientLoginProof")

			_ = rc.HandleProof(&clp)
		case RealmList:
			var rlp ClientRealmlistPacket

			pkt.Unmarshal(&rlp)

			log.Trace().Msgf(">> WoW -> Auth ClientRealmlistPacket")

			_ = rc.HandleRealmList()
		case AuthReconnectChallenge:
			fallthrough
		case AuthReconnectProof:
			rc.log.Fatal().Msgf("unhandled command: %T(0x%02x)", pkt.Command, pkt.Command)
		}
	}
}

// read reads the packet from the auth socket.
func (rc *AuthConnection) read(r io.Reader) (*RData, error) {
	opCodeData := make([]byte, 1)

	n, err := r.Read(opCodeData)
	if err != nil {
		return nil, fmt.Errorf("erorr while reading command: %w", err)
	}

	if n != 1 {
		return nil, ErrShortRead
	}

	// In the auth server, the length is based on the packet type.
	opCode := RealmCommand(opCodeData[0])

	var length int

	switch opCode {
	case AuthLoginChallenge:
		lenData, err := ReadBytes(r, 3)
		if err != nil {
			return nil, fmt.Errorf("error while reading header length: %w", err)
		}

		length = int(binary.LittleEndian.Uint16(lenData[1:]))
	case AuthLoginProof:
		length = 74
	case RealmList:
		length = 4
	case AuthReconnectChallenge, AuthReconnectProof:
		rc.log.Error().
			Hex("packet", opCodeData).
			Msg("packet is not handled yet")

		return nil, fmt.Errorf("%w: %v", ErrNoHandler, opCode)
	}

	bb, err := ReadBytes(r, length)
	if err != nil {
		return nil, err
	}

	ret := RData{
		Command: uint8(opCode),
		Data:    bb,
	}

	return &ret, nil
}
