package wow

type HasGuid interface {
	Guid() uint64
}

type Updater interface {
	Update()
}

type HasObjectType interface {
	ObjectType() string
}

// HasLocation is an interface for objects that can be located.
type HasLocation interface {
	Location() (float32, float32, float32, uint)
}

type WorldUnit interface {
	HasGuid
	HasLocation
	HasObjectType
	Updater
}
