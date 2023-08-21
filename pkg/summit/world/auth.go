package world

import (
	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/paalgyula/summit/pkg/wow/wotlk"
)

//nolint:godox
func (gc *GameClient) sendAuthChallenge() {
	// TODO: #21 investigate on how the seed used and calculated
	// gc.seed = uint32(rand.Int31())
	gc.seed = 0

	// 0x1ec
	w := wow.NewPacket(wow.ServerAuthChallenge)
	_ = w.Write(gc.seed) // This is a seed

	gc.Send(w)
}

type ClientAuthSessionPacket struct {
	ClientBuild      uint32
	ServerID         uint32
	AccountName      string
	ClientSeed       uint32
	Digest           []byte // 20bytes long
	AddonSize        uint16
	AddonsCompressed []byte
}

//nolint:errcheck
func (cas *ClientAuthSessionPacket) Bytes() []byte {
	pkt := wow.NewPacket(wow.ClientAuthSession)
	pkt.Write(cas.ClientBuild)
	pkt.Write(cas.ServerID)
	pkt.WriteString(cas.AccountName)
	pkt.Write(cas.ClientSeed)

	// Duno whats this (paalgyula)
	pkt.Write(uint32(0x00))
	pkt.Write(uint32(0x00))
	pkt.Write(uint32(0x00))
	pkt.Write(uint64(0x00))

	pkt.WriteReverseBytes(cas.Digest[:20])
	pkt.Write(cas.AddonSize)
	pkt.Write(cas.AddonsCompressed)

	return pkt.Bytes()
}

type BillingDetails struct {
	BillingTimeRemaining uint32
	BillingFlags         uint8
	BillingTimeRested    uint32
}

//nolint:godox,errcheck
func (gc *GameClient) AuthSessionHandler(data wow.PacketData) {
	reader := wow.NewPacketReader(data)
	pkt := new(ClientAuthSessionPacket)

	reader.Read(&pkt.ClientBuild)
	reader.Read(&pkt.ServerID)
	reader.ReadString(&pkt.AccountName)
	reader.Read(&pkt.ClientSeed)

	// Skip fragment Whats that?
	var tmp uint32

	var tmp2 uint64

	reader.Read(&tmp)
	reader.Read(&tmp)
	reader.Read(&tmp)
	reader.Read(&tmp2)

	reader.ReadL(&pkt.Digest)
	reader.ReadL(&pkt.AddonSize)

	// TODO: rewrite back from singleton to instance based DB
	acc := db.GetInstance().FindAccount(pkt.AccountName)
	if acc != nil {
		gc.acc = acc
	}

	// TODO: check the digest

	var err error

	gc.crypt, err = crypt.NewWowcrypt(acc.SessionKey())
	if err != nil {
		panic(err)
	}

	gc.log.Debug().Str("key", acc.SessionKey().Text(16)).Send()

	//nolint:varnamelen
	p := wow.NewPacket(wow.ServerAuthResponse)
	p.Write(uint8(wotlk.AUTH_OK))
	p.Write(&BillingDetails{
		BillingTimeRemaining: 0,
		BillingFlags:         0,
		BillingTimeRested:    0,
	})
	p.Write(uint8(2)) // Expansion

	gc.Send(p)
}
