package wow

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

var dumper *PacketDumper

var o sync.Once

func initDumper() {
	filePath := "packetdump.txt"

	// Open the file in append mode. Create the file if it doesn't exist.
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	dumper = &PacketDumper{
		w: file,
	}
}

type PacketDumper struct {
	m sync.Mutex
	w io.Writer
}

func (p *PacketDumper) Write(code OpCode, data []byte) {
	p.m.Lock()
	defer p.m.Unlock()

	bw := bufio.NewWriter(p.w)

	b64data := base64.StdEncoding.EncodeToString(data)
	bw.WriteString(fmt.Sprintf("# code: 0x%04x len: %05d\n%s\n", int(code), len(data), b64data))

	bw.Flush()
}

func GetPacketDumper() *PacketDumper {
	o.Do(initDumper)

	return dumper
}

func ParseDumpedPacket(packet string) (int, []byte, error) {
	var code int
	var length int
	var b64data string

	_, err := fmt.Sscanf(packet, "# code: 0x%04x len: %05d\n%s", &code, &length, &b64data)
	if err != nil {
		return 0, nil, err
	}

	bb, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return 0, nil, err
	}

	if len(bb) != length {
		return 0, nil, errors.New("wrong packet length")
	}

	return code, bb, nil
}
