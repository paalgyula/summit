package auth

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/paalgyula/summit/pkg/summit/auth/packets"
)

type AuthPacket interface {
	MarshalPacket() []byte
	OpCode() packets.AuthCmd
}

type PacketWriter struct {
	m       sync.Mutex
	stream  io.Writer
	version int
}

func NewPacketWriter(out io.Writer, version int) *PacketWriter {
	return &PacketWriter{
		stream:  out,
		version: version,
	}
}

func (pw *PacketWriter) Send(pkt AuthPacket) {
	pw.m.Lock()
	defer pw.m.Unlock()

	bb := pkt.MarshalPacket()

	data := append([]byte{}, byte(pkt.OpCode()), uint8(pw.version))
	data = binary.LittleEndian.AppendUint16(data, uint16(len(bb)))
	data = append(data, bb...)

	fmt.Println("Sending: ", pkt.OpCode())
	fmt.Printf("%s", hex.Dump(data))

	pw.stream.Write(data)
}
