package client

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/paalgyula/summit/pkg/wow/wotlk"
)

type CharEnum struct {
	guid   wow.GUID
	Name   string
	Race   wow.PlayerRace
	Class  wow.PlayerClass
	Gender wow.PlayerGender

	Skin       uint8
	Face       uint8
	HairStyle  uint8
	HairColor  uint8
	FacialHair uint8
	OutfitID   uint8

	Level uint8

	Location player.WorldLocation

	GuildID uint32

	// CharFlags for example dead, and display ghost
	CharFlags       uint32
	Recustomization uint32

	FirstLogin uint8

	Pet player.Pet

	Inventory *player.Inventory
}

func readCharEnum(r *wow.PacketReader) *CharEnum {
	var c CharEnum

	r.Read(&c.guid)
	r.ReadString(&c.Name)

	r.Read(&c.Race)
	r.Read(&c.Class)
	r.Read(&c.Gender)

	r.Read(&c.Skin)
	r.Read(&c.Face)
	r.Read(&c.HairStyle)
	r.Read(&c.HairColor)
	r.Read(&c.FacialHair)

	r.Read(&c.Level)

	r.Read(&c.Location.Zone)
	r.Read(&c.Location.Map)

	r.Read(&c.Location.X)
	r.Read(&c.Location.Y)
	r.Read(&c.Location.Z)

	r.Read(&c.GuildID)

	// Character flags
	r.Read(&c.CharFlags)
	r.Read(&c.Recustomization)

	// First login
	// *data << uint8(atLoginFlags & AT_LOGIN_FIRST ? 1 : 0);
	r.Read(&c.FirstLogin)

	// Player Pet section
	r.Read(&c.Pet.DisplayID)
	r.Read(&c.Pet.PetLevel)
	r.Read(&c.Pet.PetFamilly)

	c.Inventory = &player.Inventory{
		InventorySlots: make([]*player.InventoryItem, player.InventorySlotBagEnd),
	}

	for i := range c.Inventory.InventorySlots {
		var slot player.InventoryItem
		r.Read(&slot.DisplayInfoID)
		r.Read(&slot.InventoryType)
		r.Read(&slot.EnchantSlot)

		c.Inventory.InventorySlots[i] = &slot
	}

	return &c
}

func (wc *WorldClient) handleCharEnum(msg *ServerMessage) {
	r := msg.Reader()

	var count uint8
	r.Read(&count)

	chars := make([]*CharEnum, count)
	for i := 0; i < int(count); i++ {
		chars[i] = readCharEnum(r)
	}

	fmt.Println("readed %d characters", len(chars))
}

func (wc *WorldClient) handleAuthResponse(msg *ServerMessage) {
	r := msg.Reader()

	var status uint8
	r.Read(&status)

	if status == wotlk.AUTH_OK { // success
		wc.Send(wow.NewPacket(wow.ClientCharEnum))

		return
	}

	wc.log.Error().Msgf("auth failed with status: 0x%02x", status)

	wc.Disconnect()
}

func (wc *WorldClient) handleAuthChallenge(msg *ServerMessage) {
	r := msg.Reader()

	var placeholder uint32
	_ = r.Read(&placeholder) // Should be the placeholder, always 1

	// ? handle the error here
	// if placeholder != 1 {
	// }

	wc.serverSeed, _ = r.ReadNBytes(4)
	// wc.log.Error().Msgf("auth seed: 0x%x", wc.serverSeed)

	// * the encrypt keys are unused yet
	encryptKeys := make([]uint8, 32)
	if err := r.Read(encryptKeys); err != nil {
		wc.log.Error().Err(err).Msg("cannot read new encryption seed from auth challenge")
	}

	cs := make([]byte, 4)
	_, _ = rand.Read(cs)

	// * generate session proof for login
	digest := crypt.AuthSessionProof(
		wc.AccountName,
		wc.serverSeed,
		cs,
		wc.SessionKey.Bytes(),
	)

	addonInfo, _ := hex.DecodeString(defaultAddonInfo)
	cas := world.ClientAuthSessionPacket{
		ClientBuild:     12340,
		ServerID:        0x0,
		AccountName:     wc.AccountName,
		LoginServerType: 0,
		ClientSeed:      cs,
		RegionID:        0x00,
		BattleGroupID:   0x00,
		RealmID:         0,
		DOSResponse:     0,
		Digest:          digest,
		AddonInfo:       addonInfo,
	}

	pkt := wow.NewPacket(wow.ClientAuthSession)
	pkt.Write(cas.Bytes())

	wc.Send(pkt)
}
