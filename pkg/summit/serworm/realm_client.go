package serworm

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type RealmPacket interface {
	MarshalPacket() []byte
	OpCode() auth.RealmCommand
}

type RealmClient struct {
	readMutex  sync.Mutex
	writeMutex sync.Mutex

	conn    net.Conn
	version int

	srp        *crypt.SRP6
	sessionKey *big.Int

	log zerolog.Logger

	user, pass string

	// Channel for receiving realm
	realmChannel chan []*auth.Realm
}

func NewRealmClient(conn net.Conn, version int) *RealmClient {
	rs := &RealmClient{
		conn:         conn,
		version:      version,
		log:          log.With().Str("module", "realmclient").Logger(),
		realmChannel: make(chan []*auth.Realm),
	}

	go rs.HandlePackets()

	return rs
}

// Authenticate authenticates a user with a password and returns a slice of realms or an error.
func (pw *RealmClient) Authenticate(user, pass string) ([]*auth.Realm, error) {
	pw.user = strings.ToUpper(user)
	pw.pass = pass

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pw.SendChallenge(user)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case realms := <-pw.realmChannel:
		return realms, nil
	}
}

func (pw *RealmClient) Send(pkt RealmPacket) {
	bb := pkt.MarshalPacket()

	buf := &bytes.Buffer{}

	buf.WriteByte(byte(pkt.OpCode()))

	if pkt.OpCode() == auth.AuthLoginChallenge {
		// Write version + length when sending logon challenge
		buf.WriteByte(byte(pw.version))
		binary.Write(buf, binary.LittleEndian, uint16(len(bb)))
	}

	buf.Write(bb)

	pw.writeMutex.Lock()
	defer pw.writeMutex.Unlock()

	fmt.Println(">> Sending:", pkt.OpCode())
	fmt.Printf("%s", hex.Dump(buf.Bytes()))

	buf.WriteTo(pw.conn)
}

func (pw *RealmClient) SendChallenge(user string) {
	pw.log.Info().Msgf("logging in as user: %s", user)
	user = strings.ToUpper(user)
	clp := auth.NewClientLoginChallenge(user)

	pw.Send(clp)
}

func (pw *RealmClient) ReadCommand() auth.RealmCommand {
	var cmd uint8

	_ = binary.Read(pw.conn, binary.LittleEndian, &cmd)

	return auth.RealmCommand(cmd)
}

func (pw *RealmClient) HandleLoginChallenge() {
	var slc auth.ServerLoginChallenge

	readed := slc.ReadPacket(pw.conn)
	if readed == 0 {
		panic("failed to read challenge")
	}

	log.Debug().Msgf("received challenge with status: 0x%02x readed: %d", slc.Status, readed)

	// Initialize SRP6
	pw.srp = crypt.NewSRP6(int64(slc.G), 3, &slc.N)
	pw.srp.B = &slc.B
	A := pw.srp.GenerateClientPubkey()

	pass := strings.ToUpper(pw.pass)
	K, M := pw.srp.CalculateClientSessionKey(&slc.Salt, &slc.B, pw.user, pass)

	fmt.Println("s: ", slc.Salt.Text(16))
	fmt.Println("K: ", K.Text(16))
	fmt.Println("B: ", slc.B.Text(16))
	fmt.Println("M: ", M.Text(16))

	proof := auth.ClientLoginProof{
		A:             *A,
		M:             *M,
		CRCHash:       slc.SaltCRC,
		NumberOfKeys:  0,
		SecurityFlags: 0,
	}

	pw.sessionKey = K
	pw.Send(proof)
}

func (pw *RealmClient) HandleProof() {
	var proof auth.ServerLoginProof
	_ = proof.ReadPacket(pw.conn)

	log.Debug().Msgf("proof response received: %+v", proof)

	if proof.StatusCode != 0 {
		log.Fatal().Msgf("cannot proof user: %v", auth.ChallengeStatus(proof.StatusCode))
	}

	pkt := new(auth.ClientRealmlistPacket)
	pw.Send(pkt)
}

func (pw *RealmClient) HandleRealmlist() {
	pkt := new(auth.ServerRealmlistPacket)
	reader := wow.NewConnectionReader(pw.conn)
	pkt.ReadPacket(reader)

	log.Debug().Msgf("realmlist response received: %+v", pkt)

	pw.realmChannel <- pkt.Realms
}

func (pw *RealmClient) HandlePackets() {
	for {
		cmd := pw.ReadCommand()

		switch cmd {
		case auth.AuthLoginChallenge:
			pw.HandleLoginChallenge()
		case auth.AuthLoginProof:
			pw.HandleProof()
		case auth.RealmList:
			pw.HandleRealmlist()
		default:
			pw.log.Fatal().Msgf("packet not handled: %v(0x%02x)", cmd, int(cmd))
		}
	}
}
