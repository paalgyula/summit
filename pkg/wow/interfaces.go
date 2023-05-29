package wow

type GuidHolder interface {
	GetGuid() uint64
}

type WorldUnit interface {
	GuidHolder
	HasLocation
	Update()

	ObjectType() string
}

// HasLocation is an interface for objects that can be located
type HasLocation interface {
	Location() (float32, float32, float32, uint)
}
