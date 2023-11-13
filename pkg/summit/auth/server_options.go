package auth

type ServerOption func(s *Server)

// WithRealmProvider sets the realm provider for the server.
func WithRealmProvider(rp RealmProvider) ServerOption {
	return func(s *Server) {
		s.rp = rp
	}
}
