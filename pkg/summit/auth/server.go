package auth

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"

	"github.com/paalgyula/summit/pkg/db"
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

type Server struct {
	l net.Listener
	// The realm provider
	rp RealmProvider
}

func (as *Server) Run() {
	for {
		c, err := as.l.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "closed network connection") {
				// Do not log, we are closed the client
				return
			}

			log.Error().Err(err).Msgf("listener error")

			return
		}

		NewAuthConnection(c, as.rp)
	}
}

func (as *Server) Close() error {
	//nolint:wrapcheck
	return as.l.Close()
}

func NewServer(listenAddress string, rp RealmProvider) (*Server, error) {
	l, err := net.Listen("tcp4", listenAddress)
	if err != nil {
		return nil, fmt.Errorf("auth.StartServer: %w", err)
	}

	log.Info().Msgf("auth server is listening on: %s", listenAddress)

	as := &Server{
		l:  l,
		rp: rp,
	}

	go as.Run()

	return as, nil
}

type Connection struct {
	c net.Conn

	outLock sync.Mutex

	log zerolog.Logger
	id  string

	rp RealmProvider

	account *db.Account

	srp *crypt.SRP6
}

func NewAuthConnection(c net.Conn, rp RealmProvider) *Connection {
	rc := &Connection{
		c:       c,
		log:     log.With().Str("addr", c.RemoteAddr().String()).Logger(),
		account: nil,
		id:      xid.New().String(),

		srp: crypt.NewSRP6(7, 3, big.NewInt(0)),
		rp:  rp,

		outLock: sync.Mutex{},
	}

	go rc.listen()

	return rc
}

//nolint:godox
func (rc *Connection) HandleLogin(pkt *ClientLoginChallenge) error {
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
		rc.account = db.GetInstance().FindAccount(pkt.AccountName)

		if rc.account == nil {
			res.Status = ChallengeStatusFailUnknownAccount
			rc.c.Close()
		}
	}

	if res.Status == ChallengeStatusSuccess {
		B := rc.srp.GenerateServerPubKey(rc.account.Verifier())

		res.B.Set(B)
		res.Salt.Set(rc.account.Salt())
		res.SaltCRC = make([]byte, 16)

		res.G = uint8(rc.srp.GValue())
		res.N = *rc.srp.N()
	}

	// Send out the packet
	return rc.Send(AuthLoginChallenge, res.MarshalPacket())
}

func (rc *Connection) HandleProof(pkt *ClientLoginProof) error {
	//nolint:exhaustruct
	response := ServerLoginProof{}

	K, M := rc.srp.CalculateServerSessionKey(
		&pkt.A,
		rc.account.Verifier(),
		rc.account.Salt(),
		rc.account.Name)

	if M.Cmp(&pkt.M) != 0 {
		response.StatusCode = 4
		_ = rc.Send(AuthLoginProof, response.MarshalPacket())
		rc.c.Close()

		return nil
	}

	response.StatusCode = 0
	response.Proof.Set(crypt.CalculateServerProof(&pkt.A, M, K))

	rc.log = rc.log.With().
		Str("account", rc.account.Name).
		Logger()

	rc.account.SetKey(K)

	// Save session key
	db.GetInstance().SaveAll()

	return rc.Send(AuthLoginProof, response.MarshalPacket())
}

//nolint:godox
func (rc *Connection) HandleRealmList() error {
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

func (rc *Connection) Send(opcode RealmCommand, payload []byte) error {
	size := len(payload)

	rc.log.Debug().
		Str("opcode", fmt.Sprintf("0x%04x", int(opcode))).
		Int("size", size).
		Hex("data", payload).
		Msg("sending packet to client")

	w := wow.NewPacket(0)
	_ = w.Write(uint8(opcode))
	_, _ = w.WriteBytes(payload)

	return rc.Write(w.Bytes())
}

func (rc *Connection) Write(bb []byte) error {
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

func (rc *Connection) listen() {
	defer rc.c.Close()
	rc.log.Info().Msgf("accepting messages from a new login connection")

	for {
		// Read packets infinitely :)
		pkt, err := rc.read(rc.c)
		if err != nil || pkt == nil {
			log.Error().Err(err).Msg("error while reading from client")

			return
		}

		switch RealmCommand(pkt.Command) {
		case AuthLoginChallenge:
			var clc ClientLoginChallenge

			pkt.Unmarshal(&clc)

			log.Trace().Msgf(">> WoW -> Auth ClientLoginChallenge\n%s", hex.Dump(clc.MarshalPacket()))

			_ = rc.HandleLogin(&clc)
		case AuthLoginProof:
			var clp ClientLoginProof

			pkt.Unmarshal(&clp)

			log.Trace().Msgf(">> WoW -> Auth ClientLoginProof\n%s", hex.Dump(clp.MarshalPacket()))

			_ = rc.HandleProof(&clp)
		case RealmList:
			var rlp ClientRealmlistPacket

			pkt.Unmarshal(&rlp)

			log.Trace().Msgf(">> WoW -> Auth ClientRealmlistPacket\n%s", hex.Dump(rlp.MarshalPacket()))

			_ = rc.HandleRealmList()
		case AuthReconnectChallenge:
			fallthrough
		case AuthReconnectProof:
			rc.log.Fatal().Msgf("unhandled command: %T(0x%02x)", pkt.Command, pkt.Command)
		}
	}
}

// read reads the packet from the auth socket.
func (rc *Connection) read(r io.Reader) (*RData, error) {
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

// ReadBytes will read a specified number of bytes from a given buffer. If not all
// of the data is read (or there was an error), an error will be returned.
func ReadBytes(buffer io.Reader, length int) ([]byte, error) {
	data := make([]byte, length)

	if length > 0 {
		n, err := buffer.Read(data)
		if err != nil {
			return nil, fmt.Errorf("error while reading bytes: %w", err)
		}

		if n != length {
			log.Trace().Msgf("WTF: %s\n", hex.Dump(data[:n]))

			return nil, fmt.Errorf("%w: wanted %v bytes, got %v", ErrShortRead, length, n)
		}
	}

	return data, nil
}
