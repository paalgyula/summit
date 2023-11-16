package basedata

//nolint:gochecknoglobals
var store *Store

func GetInstance() *Store {
	return store
}
