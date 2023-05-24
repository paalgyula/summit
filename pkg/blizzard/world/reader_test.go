package world

import (
	"fmt"
	"testing"

	"github.com/paalgyula/summit/pkg/blizzard/world/packets"
	"github.com/paalgyula/summit/pkg/wow"
)

const data = "\xae\xa2\x8e^\r\xbc"

var data2 = []byte{0x00, 0x0c, 0xdc, 0x01, 0x0, 0x0}

func TestHeaderReading(t *testing.T) {
	// p := wow.NewPacketReader([]byte(data))

	p := wow.NewPacketReader([]byte(data2))

	var len uint16
	var opcode uint32
	p.ReadB(&len)
	p.ReadL(&opcode)

	fmt.Printf("%d, %04x %s", len, opcode, packets.OpCode(opcode))
}
