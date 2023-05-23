package packets

import (
	"math/big"

	"github.com/paalgyula/summit/lib/util"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/server/auth/data/static"
)

// ClientLoginProof encodes proof that the client has the correct information.
type ClientLoginProof struct {
	A             big.Int
	M             big.Int
	CRCHash       big.Int
	NumberOfKeys  uint8
	SecurityFlags uint8
}

// Read will load a ClientLoginProof packet from a buffer.
// An error will be returned if at least one of the fields didn't load correctly.
func (pkt *ClientLoginProof) UnmarshalPacket(bb []byte) error {
	r := wow.NewPacketReader(bb)

	pkt.A.SetBytes(r.ReadReverseBytes(32))
	pkt.M.SetBytes(r.ReadReverseBytes(20))
	pkt.CRCHash.SetBytes(r.ReadReverseBytes(20))

	r.ReadL(&pkt.NumberOfKeys)
	return r.ReadL(&pkt.SecurityFlags)
}

// ServerLoginProof is the server's response to a client's challenge. It contains
// some SRP information used for handshaking.
type ServerLoginProof struct {
	Error static.LoginErrorCode
	Proof big.Int
}

// Bytes writes out the packet to an array of bytes.
func (pkt *ServerLoginProof) MarshalPacket() []byte {
	w := wow.NewPacketWriter()

	w.WriteByte(uint8(pkt.Error))

	if pkt.Error == 0 {
		w.Write(util.PadBigIntBytes(util.ReverseBytes(pkt.Proof.Bytes()), 32))
		// buffer.Write([]byte("\x00\x00\x00\x00")) // unk1
	}

	return w.Bytes()
}
