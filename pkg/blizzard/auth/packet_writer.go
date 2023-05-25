package auth

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
)

type AuthPacket interface {
	MarshalPacket() []byte
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

func (pw *PacketWriter) Send(opcode int, pkt AuthPacket) {
	pw.m.Lock()
	defer pw.m.Unlock()

	bb := pkt.MarshalPacket()

	data := append([]byte{}, uint8(opcode), uint8(pw.version))
	data = binary.LittleEndian.AppendUint16(data, uint16(len(bb)))
	data = append(data, bb...)

	fmt.Printf(">> %s", hex.Dump(data))

	pw.stream.Write(data)
}
