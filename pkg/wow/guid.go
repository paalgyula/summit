//nolint:gomnd
package wow

import (
	"encoding/binary"
	"sync"

	"github.com/rs/zerolog/log"
)

type HighGUID uint32

const (
	ItemGUID          HighGUID = 0x4000 // blizz 4000
	ContainerGUID     HighGUID = 0x4000 // blizz 4000
	PlayerGUID        HighGUID = 0x0000 // blizz 0000
	GameObjectGUID    HighGUID = 0xF110 // blizz F110
	TransportGUID     HighGUID = 0xF120 // blizz F120 (for GAMEOBJECT_TYPE_TRANSPORT)
	UnitGUID          HighGUID = 0xF130 // blizz F130
	PetGUID           HighGUID = 0xF140 // blizz F140
	VehicleGUID       HighGUID = 0xF150 // blizz F550
	DynamicObjectGUID HighGUID = 0xF100 // blizz F100
	CorpseGUID        HighGUID = 0xF101 // blizz F100
	MoTransportGUID   HighGUID = 0x1FC0 // blizz 1FC0 (for GAMEOBJECT_TYPE_MO_TRANSPORT)
	InstanceGUID      HighGUID = 0x1F40 // blizz 1F40
	GroupGUID         HighGUID = 0x1F50
)

type GUIDPool struct {
	m sync.Mutex

	counter int

	freeIDs []int
	pool    map[HighGUID]int
}

func NewGUIDPool() *GUIDPool {
	gp := &GUIDPool{
		m:       sync.Mutex{},
		counter: 0,
		freeIDs: make([]int, 0),
		pool:    make(map[HighGUID]int),
	}

	return gp
}

func (gp *GUIDPool) Get() GUID {
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

func (gp *GUIDPool) Release(guid GUID) {
	gp.m.Lock()
	defer gp.m.Unlock()

	gp.freeIDs = append(gp.freeIDs, int(guid))
}

type GUID uint64

func (g *GUID) HasEntry() bool {
	switch g.High() {
	case ItemGUID, PlayerGUID, DynamicObjectGUID, CorpseGUID, MoTransportGUID, InstanceGUID, GroupGUID:
		return false
	case GameObjectGUID, TransportGUID, UnitGUID, PetGUID, VehicleGUID:
		return true
	}

	return false
}

func (g GUID) High() HighGUID {
	return HighGUID((g >> 48) & 0x0000FFFF)
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
	log.Printf("%64b\nValue: %32d Hex: 0x%x\n", g, g, g)
}

func NewGUID(hg HighGUID, counter uint32) GUID {
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
	return NewGUID(PlayerGUID, counter)
}

func NewItemGUID(counter uint32) GUID {
	return NewGUID(ItemGUID, counter)
}

func (g GUID) TypeID() TypeID {
	//nolint:exhaustive
	switch g.High() {
	case ItemGUID:
		return TypeIDItem
	// case HIGHGUID_CONTAINER:    return TYPEID_CONTAINER; HIGHGUID_CONTAINER==HIGHGUID_ITEM currently
	case UnitGUID, PetGUID:
		return TypeIDUnit
	case PlayerGUID:
		return TypeIDPlayer
	case GameObjectGUID, MoTransportGUID:
		return TypeIDGameObject
	case DynamicObjectGUID:
		return TypeIDDynamicoObject
	case CorpseGUID:
		return TypeIDCorpse
	case TransportGUID:
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
