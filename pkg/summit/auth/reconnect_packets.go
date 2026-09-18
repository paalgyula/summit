package auth

import (
	"crypto/rand"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/crypt"
)

// ClientReconnectChallenge is the same format as ClientLoginChallenge.
// The client sends this to initiate a reconnect using an existing session.
type ClientReconnectChallenge struct {
	GameName        string
	Version         [3]byte
	Build           uint16
	Platform        string
	OS              string
	Locale          string
	WorldRegionBias uint32
	IP              []uint8
	AccountName     string
}

func (ClientReconnectChallenge) OpCode() RealmCommand {
	return AuthReconnectChallenge
}

//nolint:errcheck
func (p *ClientReconnectChallenge) UnmarshalPacket(bb wow.PacketData) error {
	r := bb.Reader()
	r.ReadStringFixed(&p.GameName, 4)
	r.ReadL(&p.Version)
	r.ReadL(&p.Build)
	r.ReadStringFixed(&p.Platform, 4)
	r.ReadStringFixed(&p.OS, 4)
	r.ReadStringFixed(&p.Locale, 4)
	r.ReadL(&p.WorldRegionBias)
	p.IP, _ = r.ReadNBytes(4)

	var size uint8
	_ = r.ReadB(&size)

	r.ReadStringFixed(&p.AccountName, int(size))

	return nil
}

// ServerReconnectChallenge is the server's response to a reconnect challenge.
// It contains a random 16-byte proof that the client must echo back.
type ServerReconnectChallenge struct {
	StatusCode uint8
	// R1 is a random 16-byte challenge for the client.
	R1 [16]byte
	// VersionChallenge is a fixed 16-byte value.
	VersionChallenge [16]byte
}

var versionChallenge = [16]byte{
	0xBA, 0xA3, 0x1E, 0x99, 0xA0, 0x0B, 0x21, 0x57,
	0xFC, 0x37, 0x3F, 0xB3, 0x69, 0xCD, 0xD2, 0xF1,
}

// NewServerReconnectChallenge creates a new server reconnect challenge with a random R1.
func NewServerReconnectChallenge() *ServerReconnectChallenge {
	challenge := &ServerReconnectChallenge{
		StatusCode: 0, // WOW_SUCCESS
	}

	_, _ = rand.Read(challenge.R1[:])
	challenge.VersionChallenge = versionChallenge

	return challenge
}

// MarshalPacket serializes the reconnect challenge response.
//
//nolint:errcheck
func (pkt *ServerReconnectChallenge) MarshalPacket() []byte {
	w := wow.NewPacket(wow.OpCode(AuthReconnectChallenge))

	w.WriteOne(int(pkt.StatusCode))

	if pkt.StatusCode == 0 {
		w.WriteBytes(pkt.R1[:])
		w.WriteBytes(pkt.VersionChallenge[:])
	}

	return w.Bytes()
}

// ClientReconnectProof is sent by the client in response to ServerReconnectChallenge.
type ClientReconnectProof struct {
	R1           [16]byte
	R2           [20]byte // SHA1(login + R1 + serverReconnectProof + sessionKey)
	R3           [20]byte // version hash
	NumberOfKeys uint8
}

func (ClientReconnectProof) OpCode() RealmCommand {
	return AuthReconnectProof
}

//nolint:errcheck
func (pkt *ClientReconnectProof) UnmarshalPacket(bb wow.PacketData) error {
	r := bb.Reader()

	copy(pkt.R1[:], r.ReadReverseBytes(16))
	copy(pkt.R2[:], r.ReadReverseBytes(20))
	copy(pkt.R3[:], r.ReadReverseBytes(20))
	r.ReadL(&pkt.NumberOfKeys)

	return nil
}

// ServerReconnectProof is the server's response to a reconnect proof.
type ServerReconnectProof struct {
	StatusCode uint8
	LoginFlags uint16
}

// MarshalPacket serializes the reconnect proof response.
//
//nolint:errcheck
func (pkt *ServerReconnectProof) MarshalPacket() []byte {
	w := wow.NewPacket(wow.OpCode(AuthReconnectProof))

	w.WriteOne(int(pkt.StatusCode))
	w.Write(pkt.LoginFlags)

	return w.Bytes()
}

// GenerateReconnectProof calculates SHA1(login + R1 + serverReconnectProof + sessionKey)
// for verifying the client's reconnect proof.
func GenerateReconnectProof(login string, r1, serverProof, sessionKey []byte) []byte {
	return crypt.Hash(
		reverseBytes([]byte(login)),
		reverseBytes(r1),
		reverseBytes(serverProof),
		reverseBytes(sessionKey),
	)
}

// reverseBytes returns a new reversed copy of the input slice.
func reverseBytes(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}
