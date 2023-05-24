package world

import (
	"math/rand"

	"github.com/paalgyula/summit/pkg/blizzard/world/packets"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/paalgyula/summit/server/world/data/static"
)

func (gc *GameClient) sendAuthChallenge() {
	gc.seed = uint32(rand.Int31())

	// 0x1ec
	w := wow.NewPacketWriter()
	w.WriteL(uint32(0)) // This is a seed

	gc.SendPacket(packets.ServerAuthChallenge, w.Bytes())
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

func (gc *GameClient) AuthSessionHandler(data []byte) {
	r := wow.NewPacketReader(data)

	// fmt.Printf("%s", hex.Dump(data))

	pkt := new(ClientAuthSessionPacket)

	r.ReadL(&pkt.BuildNumber)
	r.ReadL(&pkt.ServerID)

	pkt.AccountName = r.ReadString()

	r.ReadL(&pkt.ClientSeed)
	r.ReadL(&pkt.Digest)
	r.ReadL(&pkt.AddonSize)

	acc := gc.ws.db.FindAccount(pkt.AccountName)

	// TODO: check the digest
	var err error
	gc.crypt, err = crypt.NewWowcrypt(acc.SessionKey())
	if err != nil {
		panic(err)
	}

	gc.log.Debug().Str("key", acc.SessionKey().Text(16)).Send()

	w := wow.NewPacketWriter()
	w.WriteL(uint8(static.AuthOK))
	w.WriteL(uint32(0)) // BillingTimeRemaining
	w.WriteL(uint8(0))  // BillingFlags
	w.WriteL(uint32(0)) // BillingTimeRested
	w.WriteL(uint8(1))  // Expansion

	gc.SendPacket(packets.ServerAuthResponse, w.Bytes())

}
