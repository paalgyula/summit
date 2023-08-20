//nolint:gomnd
package wow

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type HighGuid uint32

const (
	ItemGuid          HighGuid = 0x4000 // blizz 4000
	ContainerGuid     HighGuid = 0x4000 // blizz 4000
	PlayerGuid        HighGuid = 0x0000 // blizz 0000
	GameObjectGuid    HighGuid = 0xF110 // blizz F110
	TransportGuid     HighGuid = 0xF120 // blizz F120 (for GAMEOBJECT_TYPE_TRANSPORT)
	UnitGuid          HighGuid = 0xF130 // blizz F130
	PetGuid           HighGuid = 0xF140 // blizz F140
	VehicleGuid       HighGuid = 0xF150 // blizz F550
	DynamicObjectGuid HighGuid = 0xF100 // blizz F100
	CorpseGuid        HighGuid = 0xF101 // blizz F100
	MoTransportGuid   HighGuid = 0x1FC0 // blizz 1FC0 (for GAMEOBJECT_TYPE_MO_TRANSPORT)
	InstanceGuid      HighGuid = 0x1F40 // blizz 1F40
	GroupGuid         HighGuid = 0x1F50
)

type GuidPool struct {
	m sync.Mutex

	counter int

	freeIDs []int
	pool    map[HighGuid]int
}

func NewGuidPool() *GuidPool {
	gp := &GuidPool{
		freeIDs: make([]int, 0),
		pool:    make(map[HighGuid]int),
	}

	return gp
}

func (gp *GuidPool) Get() GUID {
	gp.m.Lock()
	defer gp.m.Unlock()

	if len(gp.freeIDs) > 0 {
		id := GUID(gp.freeIDs[0])
		gp.freeIDs = gp.freeIDs[1:]

		return id
	}

	gp.counter++

	return GUID(gp.counter)
}

func (gp *GuidPool) Release(guid GUID) {
	gp.m.Lock()
	defer gp.m.Unlock()

	gp.freeIDs = append(gp.freeIDs, int(guid))
}

type GUID uint64

func (g *GUID) HasEntry() bool {
	switch g.High() {
	case ItemGuid, PlayerGuid, DynamicObjectGuid, CorpseGuid, MoTransportGuid, InstanceGuid, GroupGuid:
		return false
	case GameObjectGuid, TransportGuid, UnitGuid, PetGuid, VehicleGuid:
		return true
	}

	return false
}

func (g GUID) High() HighGuid {
	return HighGuid((g >> 48) & 0x0000FFFF)
}

func (g GUID) Entry() uint32 {
	if g.HasEntry() {
		return uint32(uint64(g) >> 24 & uint64(0x0000000000FFFFFF))
	}

	return 0
}

func (g GUID) Counter() uint32 {
	return uint32(g)
}

func (g GUID) PrintRAW() {
	fmt.Printf("%64b\nValue: %32d Hex: 0x%x\n", g, g, g)
}

func NewGUID(hg HighGuid, counter uint32) GUID {
	return GUID(
		// rawValue: uint64((uint64(hg) << 48) & uint64(counter)),
		uint64(counter) | (uint64(hg) << 48),
	)
}

// Pack returns a minimal version of the GUID as an array of bytes.
func (g GUID) Pack() []byte {
	guidBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(guidBytes, uint64(g))

	mask := uint8(0)
	packedGUID := make([]byte, 0)

	for i, b := range guidBytes {
		if b != 0 {
			mask |= (1 << uint(i))

			packedGUID = append(packedGUID, b)
		}
	}

	return append([]byte{mask}, packedGUID...)
}

func NewPlayerGUID(counter uint32) GUID {
	return NewGUID(PlayerGuid, counter)
}

func NewItemGUID(counter uint32) GUID {
	return NewGUID(ItemGuid, counter)
}

func (g GUID) TypeID() TypeID {
	switch g.High() {
	case ItemGuid:
		return TypeIDItem
	// case HIGHGUID_CONTAINER:    return TYPEID_CONTAINER; HIGHGUID_CONTAINER==HIGHGUID_ITEM currently
	case UnitGuid, PetGuid:
		return TypeIDUnit
	case PlayerGuid:
		return TypeIDPlayer
	case GameObjectGuid, MoTransportGuid:
		return TypeIDGameObject
	case DynamicObjectGuid:
		return TypeIDDynamicoObject
	case CorpseGuid:
		return TypeIDCorpse
	case TransportGuid:
	default: // unknown
		return TypeIDObject
	}

	log.Warn().Msgf("unknown guid type: 0x%04x", g.High())

	return TypeIDObject
}

type TypeID uint8

const (
	TypeIDObject TypeID = iota
	TypeIDItem
	TypeIDContainer
	TypeIDUnit
	TypeIDPlayer
	TypeIDGameObject
	TypeIDDynamicoObject
	TypeIDCorpse
)
