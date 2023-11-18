package auth

import (
	"net"
)

type ServerOption func(s *Server)

// WithRealmProvider sets the realm provider for the server.
func WithRealmProvider(rp RealmProvider) ServerOption {
	return func(s *Server) {
		s.realmProvider = rp
	}
}

// WithManagement this option will initialize a JSON-RPC 2.0
// server listening on the specified listener to be able to manage
// the account server remotely. The management should be enabled
// to be able to use summitcli and separated world servers.
func WithManagement(l net.Listener) ServerOption {
	return func(s *Server) {
		s.rpcListener = l
	}
}

// WithAccountStore you can specify the account source.
// func WithAccountStore(store store.AccountRepository) ServerOption {
// 	return func(s *Server) {
// 		s.accounts = NewManagementService(store)
// 	}
// }
