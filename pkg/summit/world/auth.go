package world

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/paalgyula/summit/pkg/wow/wotlk"
)

func (gc *GameClient) sendAuthChallenge() {
	// 0x1ec
	w := wow.NewPacket(wow.ServerAuthChallenge)

	encryptSeed := make([]byte, 32)
	_, _ = rand.Read(encryptSeed)

	_ = w.Write(uint32(1))
	_ = w.Write(gc.serverSeed) // This is a seed
	_ = w.Write(encryptSeed)

	gc.Send(w)
}

type ClientAuthSessionPacket struct {
	ClientBuild     uint32
	ServerID        uint32
	AccountName     string
	LoginServerType uint32

	// ClientSeed seed or maybe better name as local challenge, sent by the client to verify the hash.
	ClientSeed []byte

	RegionID      uint32
	BattleGroupID uint32
	RealmID       uint32
	DOSResponse   uint64 // ? don't really know whats this

	// Digest20 bytes long SHA1 hash
	Digest []byte

	AddonInfo []byte
}

//nolint:errcheck
func (cas *ClientAuthSessionPacket) Bytes() []byte {
	pkt := wow.NewPacket(wow.ClientAuthSession)
	pkt.Write(cas.ClientBuild)
	pkt.Write(cas.ServerID)
	pkt.WriteString(cas.AccountName)
	pkt.Write(cas.LoginServerType)

	// Seed or Local challenge...
	pkt.Write(cas.ClientSeed)

	pkt.Write(cas.RegionID)
	pkt.Write(cas.BattleGroupID)
	pkt.Write(cas.RealmID)
	pkt.Write(cas.DOSResponse)

	pkt.WriteReverseBytes(cas.Digest[:20])

	pkt.Write(cas.AddonInfo)

	return pkt.Bytes()
}

//nolint:errcheck
func (cas *ClientAuthSessionPacket) ReadPacket(reader *wow.PacketReader) {
	reader.Read(&cas.ClientBuild)
	reader.Read(&cas.ServerID)
	reader.ReadString(&cas.AccountName)
	reader.Read(&cas.LoginServerType)

	cas.ClientSeed, _ = reader.ReadNBytes(4)

	reader.Read(&cas.RegionID)
	reader.Read(&cas.BattleGroupID)
	reader.Read(&cas.RealmID)
	reader.Read(&cas.DOSResponse)

	cas.Digest, _ = reader.ReadNBytes(20)

	// * Addon data read refactored here, reading all data into a byte array!
	cas.AddonInfo, _ = reader.ReadAll()
}

func (cas *ClientAuthSessionPacket) String() string {
	return fmt.Sprintf(
		"AccountName: %s ClientSeed: 0x%x Digest: %s, AddonInfo: %s",
		cas.AccountName, cas.ClientSeed, hex.EncodeToString(cas.Digest), hex.EncodeToString(cas.AddonInfo),
	)
}

type BillingDetails struct {
	BillingTimeRemaining uint32
	BillingFlags         uint8
	BillingTimeRested    uint32
}

// AuthSessionHandler handling auth session, exchanging session token from
// logon server and checks for auth proof, then initializes the wowcrypt
// two way packet header encryption.
//
//nolint:godox,errcheck
func (gc *GameClient) AuthSessionHandler(data wow.PacketData) {
	reader := wow.NewPacketReader(data)
	pkt := new(ClientAuthSessionPacket)

	pkt.ReadPacket(reader)

	// Call back to logon server to retrive the session key
	authSession := gc.ws.GetAuthSession(pkt.AccountName)
	if authSession != nil {
		gc.AccountName = strings.ToLower(authSession.AccountName)
		gc.sessionKey, _ = new(big.Int).SetString(authSession.SessionKey, 16)
	}

	// TODO: implement auth proof calculation
	// proof := crypt.AuthSessionProof(acc.Name, gc.serverSeed, pkt.ClientSeed, []byte(acc.Session))

	gc.log.Error().Msg("digest calculation not implemented yet, allowing all clients!!!")
	// gc.log.Trace().Msgf("%s ServerSeed: 0x%x SKey: %s",
	// 	pkt.String(), gc.serverSeed, authSession.SessionKey)

	gc.log = gc.log.With().Str("acc", gc.AccountName).Logger()

	var err error

	gc.crypt, err = crypt.NewWowcrypt(gc.sessionKey, 1024)
	if err != nil {
		panic(err)
	}

	//nolint:varnamelen
	p := wow.NewPacket(wow.ServerAuthResponse)

	// TODO: Implement server queuing

	queued := false

	if queued {
		// * Server is full, user queued
		p.WriteOne(wotlk.AUTH_WAIT_QUEUE)
	} else {
		// * Auth OK case
		p.WriteOne(wotlk.AUTH_OK)
	}

	p.Write(&BillingDetails{
		BillingTimeRemaining: 0,
		BillingFlags:         0,
		BillingTimeRested:    0,
	})

	// TODO: get the expansion from account (database)
	p.Write(uint8(2)) // Expansion

	// Queue info
	if queued {
		p.Write(uint32(4))
		p.WriteOne(0x00) // ? Whats this
	}

	gc.Send(p)
}
