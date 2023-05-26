package packets

import (
	"math/big"

	"github.com/paalgyula/summit/pkg/wow"
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
func (pkt *ClientLoginProof) UnmarshalPacket(bb wow.PacketData) error {
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
	StatusCode uint8
	Proof      big.Int
}

// Bytes writes out the packet to an array of bytes.
func (pkt *ServerLoginProof) MarshalPacket() []byte {
	w := wow.NewPacketWriter()

	w.WriteByte(uint8(pkt.StatusCode))

	if pkt.StatusCode == 0 {
		w.WriteBytes(PadBigIntBytes(wow.ReverseBytes(pkt.Proof.Bytes()), 32))
		// buffer.Write([]byte("\x00\x00\x00\x00")) // unk1
	}

	return w.Bytes()
}

// PadBigIntBytes takes as input an array of bytes and a size and ensures that the
// byte array is at least nBytes in length. \x00 bytes will be added to the end
// until the desired length is reached.
func PadBigIntBytes(data []byte, nBytes int) []byte {
	if len(data) > nBytes {
		return data[:nBytes]
	}

	currSize := len(data)
	for i := 0; i < nBytes-currSize; i++ {
		data = append(data, '\x00')
	}

	return data
}
