package basedata

//nolint:gochecknoglobals
var store *Store

// GetInstance returns the loaded base data, nil until LoadFromFile succeeds.
func GetInstance() *Store {
	return store
}

// SetInstance installs base data built in memory (tests, embedded tables).
func SetInstance(s *Store) {
	if s != nil {
		s.Index()
	}

	store = s
}
