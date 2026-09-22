//nolint:revive
package auth

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

// AuthState represents the state of an auth connection.
type AuthState uint8

const (
	AuthStateChallenge      AuthState = iota // Waiting for login/reconnect challenge
	AuthStateLogonProof                      // Waiting for login proof
	AuthStateReconnectProof                  // Waiting for reconnect proof
	AuthStateAuthed                          // Authenticated, can handle realm list
	AuthStateClosed                          // Connection closed
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

	// Auth state machine
	state AuthState

	// Reconnect state
	reconnectProof [16]byte // Random proof sent to client during reconnect challenge
	sessionKey     string   // Session key from previous login (for reconnect verification)
}

func NewAuthConnection(c net.Conn, rp RealmProvider,
	management ManagementService,
) *AuthConnection {
	rc := &AuthConnection{
		c:    c,
		log:  log.With().Str("addr", c.RemoteAddr().String()).Logger(),
		mgmt: management,
		id:   xid.New().String(),

		srp:   crypt.NewWoWSRP6(),
		rp:    rp,
		state: AuthStateChallenge,

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
	} else if rc.mgmt == nil {
		res.Status = ChallengeStatusFailUnknownAccount
		rc.c.Close()
	} else if rc.account = rc.mgmt.FindAccount(pkt.AccountName); rc.account == nil {
		// Unknown account: answer with the proper status. Without this the
		// challenge below dereferences a nil account and the whole auth
		// server dies on a single mistyped username.
		res.Status = ChallengeStatusFailUnknownAccount
		rc.c.Close()
	}

	if res.Status == ChallengeStatusSuccess {
		B := rc.srp.GenerateServerPubKey(rc.account.Verifier)

		res.B.Set(B)
		res.Salt.Set(rc.account.Salt)
		res.SaltCRC = make([]byte, 16)

		res.G = uint8(rc.srp.GValue())
		res.N = *rc.srp.N()

		rc.state = AuthStateLogonProof
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
	response.AccountFlags = 0 // Normal account
	response.SurveyID = 0
	response.LoginFlags = 0 // 0x1 = has account message

	rc.log = rc.log.With().
		Str("account", rc.account.Name).
		Logger()

	rc.mgmt.AddSession(&Session{
		AccountName: rc.account.Name,
		SessionKey:  K.Text(16),
		CreatedAt:   time.Now(),
	})

	rc.state = AuthStateAuthed

	return rc.Send(AuthLoginProof, response.MarshalPacket())
}

//nolint:godox
func (rc *AuthConnection) HandleReconnectChallenge(pkt *ClientReconnectChallenge) error {
	rc.log.Info().Str("account", pkt.AccountName).Msg("reconnect challenge received")

	// Look up the account
	rc.account = rc.mgmt.FindAccount(pkt.AccountName)
	if rc.account == nil {
		res := &ServerReconnectChallenge{
			StatusCode: 6, // ChallengeStatusFailUnknownAccount
		}

		return rc.Send(AuthReconnectChallenge, res.MarshalPacket())
	}

	// Retrieve existing session key for this account
	sess := rc.mgmt.GetSession(pkt.AccountName)
	if sess == nil {
		res := &ServerReconnectChallenge{
			StatusCode: 6, // ChallengeStatusFailUnknownAccount
		}

		return rc.Send(AuthReconnectChallenge, res.MarshalPacket())
	}

	// Store session key for reconnect proof verification
	rc.sessionKey = sess.SessionKey

	// Generate random 16-byte reconnect proof
	challenge := NewServerReconnectChallenge()
	rc.reconnectProof = challenge.R1

	rc.state = AuthStateReconnectProof

	return rc.Send(AuthReconnectChallenge, challenge.MarshalPacket())
}

func (rc *AuthConnection) HandleReconnectProof(pkt *ClientReconnectProof) error {
	rc.log.Info().Msg("reconnect proof received")

	// Verify: SHA1(login + R1 + serverReconnectProof + sessionKey) == R2
	sessionKeyBytes := hexToBytes(rc.sessionKey)

	expectedProof := GenerateReconnectProof(
		rc.account.Name,
		pkt.R1[:],
		rc.reconnectProof[:],
		sessionKeyBytes,
	)

	// Compare the proofs
	if !bytes.Equal(pkt.R2[:], expectedProof) {
		rc.log.Error().
			Str("account", rc.account.Name).
			Msg("reconnect proof verification failed")

		response := &ServerReconnectProof{
			StatusCode: 4, // auth failed
		}

		return rc.Send(AuthReconnectProof, response.MarshalPacket())
	}

	rc.log.Info().Str("account", rc.account.Name).Msg("reconnect proof verified successfully")

	rc.state = AuthStateAuthed

	response := &ServerReconnectProof{
		StatusCode: 0, // success
		LoginFlags: 0,
	}

	return rc.Send(AuthReconnectProof, response.MarshalPacket())
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
		case AuthLoginChallenge, AuthReconnectChallenge:
			// Both challenge types are valid in the Challenge state
			if rc.state != AuthStateChallenge {
				rc.log.Warn().
					Str("state", fmt.Sprintf("%d", rc.state)).
					Msg("unexpected challenge in current state")

				continue
			}

			if RealmCommand(pkt.Command) == AuthLoginChallenge {
				var clc ClientLoginChallenge

				pkt.Unmarshal(&clc)

				rc.log.Trace().Msgf(">> WoW -> Auth ClientLoginChallenge")

				_ = rc.HandleLogin(&clc)
			} else {
				var rcc ClientReconnectChallenge

				pkt.Unmarshal(&rcc)

				rc.log.Trace().Msgf(">> WoW -> Auth ClientReconnectChallenge")

				_ = rc.HandleReconnectChallenge(&rcc)
			}
		case AuthLoginProof:
			if rc.state != AuthStateLogonProof {
				rc.log.Warn().
					Str("state", fmt.Sprintf("%d", rc.state)).
					Msg("unexpected login proof in current state")

				continue
			}

			var clp ClientLoginProof

			pkt.Unmarshal(&clp)

			rc.log.Trace().Msgf(">> WoW -> Auth ClientLoginProof")

			_ = rc.HandleProof(&clp)
		case AuthReconnectProof:
			if rc.state != AuthStateReconnectProof {
				rc.log.Warn().
					Str("state", fmt.Sprintf("%d", rc.state)).
					Msg("unexpected reconnect proof in current state")

				continue
			}

			var rcp ClientReconnectProof

			pkt.Unmarshal(&rcp)

			rc.log.Trace().Msgf(">> WoW -> Auth ClientReconnectProof")

			_ = rc.HandleReconnectProof(&rcp)
		case RealmList:
			if rc.state != AuthStateAuthed {
				rc.log.Warn().
					Str("state", fmt.Sprintf("%d", rc.state)).
					Msg("unexpected realm list request in current state")

				continue
			}

			var rlp ClientRealmlistPacket

			pkt.Unmarshal(&rlp)

			log.Trace().Msgf(">> WoW -> Auth ClientRealmlistPacket")

			_ = rc.HandleRealmList()
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
	case AuthReconnectChallenge:
		// Reconnect challenge has the same format as login challenge
		lenData, err := ReadBytes(r, 3)
		if err != nil {
			return nil, fmt.Errorf("error while reading header length: %w", err)
		}

		length = int(binary.LittleEndian.Uint16(lenData[1:]))
	case AuthLoginProof:
		length = 74
	case AuthReconnectProof:
		// R1(16) + R2(20) + R3(20) + numberOfKeys(1) = 57
		length = 57
	case RealmList:
		length = 4
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

// hexToBytes converts a hex string to a byte slice.
func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}

	return b
}
