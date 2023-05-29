package object_test

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const updaeObjectPacket = `{SERVER} Packet: (0x00A9) SMSG_UPDATE_OBJECT PacketSize = 2176 stamp = 228266840 accountid = 1
|------------------------------------------------|----------------|
|00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F |0123456789ABCDEF|
|------------------------------------------------|----------------|
|0E 00 00 00 02 01 01 04 71 00 00 00 00 00 00 00 |........q.......|
|48 13 9B 0D A4 D2 AF C5 D7 03 BB C4 0C E2 AD 42 |H..............B|
|56 4A 03 3F 00 00 00 00 00 00 20 40 00 00 E0 40 |VJ.?...... @...@|
|00 00 90 40 71 1C 97 40 00 00 20 40 00 00 E0 40 |...@q..@.. @...@|
|00 00 90 40 D0 0F 49 40 00 00 E0 40 00 00 00 00 |...@..I@...@....|
|2A 15 00 80 01 C3 01 C0 58 DE 00 F1 01 08 00 00 |*.......X.......|
|4F 00 00 40 06 00 00 00 00 00 00 00 00 00 00 00 |O..@............|
|00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 |................|
|00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 |................|
|00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 |................|
|B9 6D DB B6 6D DB B6 0D 00 00 00 00 00 00 00 00 |.m..m...........|
|00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 |................|
|00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 |................|
|30 60 7F 02 00 00 00 00 00 00 00 00 00 00 00 00 |0...............|
|00 00 00 06 00 FE 20 00 00 00 40 00 00 00 00 00 |...... ...@.....|
|80 00 00 00 1E 3F 10 00 00 01 00 00 00 19 00 00 |.....?..........|
|00 00 00 80 3F 06 03 00 00 93 08 00 00 93 08 00 |....?...........|
|00 98 09 00 00 08 00 00 00 E8 03 00 00 F3 53 C4 |..............S.|
|40 3C 00 00 00 06 00 00 00 08 00 00 00 00 08 00 |@<..............|
|00 D0 07 00 00 02 2B C7 3E 00 00 C0 3F 3B 00 00 |......+.>...?;..|
|00 3B 00 00 00 00 00 20 42 00 00 20 42 00 00 80 |.;..... B.. B...|
-------------------------------------------------------------------
`

func TestParseUpdateObject(t *testing.T) {
	re := regexp.MustCompile(`\|((([a-fA-F0-9]{2})\s)*)\|`)

	pr := bytes.NewReader([]byte(updaeObjectPacket))
	r := bufio.NewReader(pr)

	// Skip header
	for i := 0; i < 4; i++ {
		r.ReadLine()
	}

	packet := bytes.NewBufferString("")

	for {
		line, _, err := r.ReadLine()
		if err != nil {
			break
		}

		if re.Match(line) {
			match := re.FindAllSubmatch(line, -1)
			bytesSection := string(match[0][1])

			// fmt.Printf("%s\n", bytesSection)
			hex := strings.Split(bytesSection, " ")

			for i := 0; i < len(hex); i++ {
				b, err := strconv.ParseInt(hex[i], 16, 16)
				if err != nil {
					continue
				}

				fmt.Printf("%s %d ", hex[i], b)
				packet.WriteByte(byte(b))
			}

		} else {
			fmt.Printf("%s\n", line)
		}

	}

	fmt.Printf("%s", hex.Dump(packet.Bytes()))
}
