package world

import (
	"math/rand"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/paalgyula/summit/pkg/wow/wotlk"
)

func (gc *GameClient) sendAuthChallenge() {
	gc.seed = uint32(rand.Int31())

	// 0x1ec
	w := wow.NewPacket(wow.ServerAuthChallenge)
	w.Write(uint32(0x00)) // This is a seed

	gc.Send(w)
}

type ClientAuthSessionPacket struct {
	BuildNumber      uint32
	ServerID         uint32
	AccountName      string
	ClientSeed       uint32
	Digest           [20]byte
	AddonSize        uint16
	AddonsCompressed []byte
}

type BillingDetails struct {
	BillingTimeRemaining uint32
	BillingFlags         uint8
	BillingTimeRested    uint32
}

func (gc *GameClient) AuthSessionHandler(data wow.PacketData) {
	r := wow.NewPacketReader(data)
	pkt := new(ClientAuthSessionPacket)

	r.ReadL(&pkt.BuildNumber)
	r.ReadL(&pkt.ServerID)

	r.ReadString(&pkt.AccountName)

	r.ReadL(&pkt.ClientSeed)
	r.ReadL(&pkt.Digest)
	r.ReadL(&pkt.AddonSize)

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

	p := wow.NewPacket(wow.ServerAuthResponse)
	p.Write(uint8(wotlk.AUTH_OK))
	// w.Write(uint32(0)) // BillingTimeRemaining
	// w.Write(uint8(0))  // BillingFlags
	// w.Write(uint32(0)) // BillingTimeRested
	p.Write(&BillingDetails{})
	p.Write(uint8(2)) // Expansion

	gc.Send(p)

}
