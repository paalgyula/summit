//go:generate stringer -type=CommandCode
package babysocket

type CommandCode uint8

const (
	CommandPacket      CommandCode = 0x00
	CommandInstruction CommandCode = 0x01
	CommandResponse    CommandCode = 0x02
)

type DataPacket struct {
	Command CommandCode
	Source  string
	Target  string
	Size    int

	Opcode int
	Data   []byte
}
