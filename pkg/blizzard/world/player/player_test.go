package player_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/stretchr/testify/assert"
)

const packetData = `# code: 0x0036 len: 00014
QmVsYQAGCwAJBAIAAwA=`

type CharacterCreateRequest struct {
	Race       uint8
	Class      uint8
	Gender     uint8
	Skin       uint8
	Face       uint8
	HairStyle  uint8
	HairColor  uint8
	FacialHair uint8
	OutfitId   uint8
}

func TestCreatePlayer(t *testing.T) {
	code, data, err := wow.ParseDumpedPacket(packetData)
	assert.NoError(t, err)
	assert.Equal(t, 0x36, code)

	fmt.Printf("%s", hex.Dump(data))

	r := wow.NewPacketReader(data)
	accName := r.ReadString()

	assert.Equal(t, "Bela", accName)

	var request CharacterCreateRequest
	binary.Read(r, binary.BigEndian, &request)

	fmt.Printf("%s %+v\n", accName, request)

	_, _ = code, data
}
