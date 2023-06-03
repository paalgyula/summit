package world

import (
	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/paalgyula/summit/pkg/wow/wotlk"
)

func (gc *GameClient) sendAuthChallenge() {
	// gc.seed = uint32(rand.Int31())
	gc.seed = 0

	// 0x1ec
	w := wow.NewPacket(wow.ServerAuthChallenge)
	w.Write(gc.seed) // This is a seed

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

func (cas *ClientAuthSessionPacket) Bytes() []byte {
	w := wow.NewPacket(wow.ClientAuthSession)
	w.Write(cas.ClientBuild)
	w.Write(cas.ServerID)
	w.WriteString(cas.AccountName)
	w.Write(cas.ClientSeed)

	// Duno whats this (paalgyula)
	w.Write(uint32(0x00))
	w.Write(uint32(0x00))
	w.Write(uint32(0x00))
	w.Write(uint64(0x00))

	w.WriteReverseBytes(cas.Digest[:20])
	w.Write(cas.AddonSize)
	w.Write(cas.AddonsCompressed)

	return w.Bytes()
}

type BillingDetails struct {
	BillingTimeRemaining uint32
	BillingFlags         uint8
	BillingTimeRested    uint32
}

func (gc *GameClient) AuthSessionHandler(data wow.PacketData) {
	r := wow.NewPacketReader(data)
	pkt := new(ClientAuthSessionPacket)

	r.Read(&pkt.ClientBuild)
	r.Read(&pkt.ServerID)
	r.ReadString(&pkt.AccountName)
	r.Read(&pkt.ClientSeed)

	// recvPacket.read_skip<uint32>();
	// recvPacket.read_skip<uint32>();
	// recvPacket.read_skip<uint32>();
	// recvPacket.read_skip<uint64>();

	// Skip fragment Whats that?
	var tmp uint32
	var tmp2 uint64
	r.Read(&tmp)
	r.Read(&tmp)
	r.Read(&tmp)
	r.Read(&tmp2)

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
