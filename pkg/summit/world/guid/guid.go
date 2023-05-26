package guid

type HighGuid uint32

const (
	Item          HighGuid = 0x4000 // blizz 4000
	Container     HighGuid = 0x4000 // blizz 4000
	Player        HighGuid = 0x0000 // blizz 0000
	GameObject    HighGuid = 0xF110 // blizz F110
	Transport     HighGuid = 0xF120 // blizz F120 (for GAMEOBJECT_TYPE_TRANSPORT)
	Unit          HighGuid = 0xF130 // blizz F130
	Pet           HighGuid = 0xF140 // blizz F140
	Vehicle       HighGuid = 0xF150 // blizz F550
	DynamicObject HighGuid = 0xF100 // blizz F100
	Corpse        HighGuid = 0xF101 // blizz F100
	Mo_Transport  HighGuid = 0x1FC0 // blizz 1FC0 (for GAMEOBJECT_TYPE_MO_TRANSPORT)
	Instance      HighGuid = 0x1F40 // blizz 1F40
	Group         HighGuid = 0x1F50
)

type GUID struct {
	rawValue uint64
}

func (g *GUID) HasEntry() bool {
	switch g.High() {
	case Item, Player, DynamicObject, Corpse, Mo_Transport, Instance, Group:
		return false
	case GameObject, Transport, Unit, Pet, Vehicle:
		return true
	}

	return false
}

func (g *GUID) RawValue() uint64 {
	return g.rawValue
}

func (g *GUID) High() HighGuid {
	return HighGuid((g.rawValue >> 48) & 0x0000FFFF)
}

func (g *GUID) Entry() uint32 {
	if g.HasEntry() {
		return uint32((g.rawValue >> 24) & uint64(0x0000000000FFFFFF))
	}

	return 0
}

func (g *GUID) Counter() uint32 {
	return uint32(g.rawValue)
}

func NewGUID(hg HighGuid, counter uint32) *GUID {
	return &GUID{
		// rawValue: uint64((uint64(hg) << 48) & uint64(counter)),
		rawValue: uint64(counter) | (uint64(hg) << 48),
	}
}

func NewPlayerGUID(counter uint32) *GUID {
	return NewGUID(Player, counter)
}
