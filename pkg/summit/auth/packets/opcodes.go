//go:generate stringer -type=AuthCmd
package packets

type AuthCmd uint32

const (
	AuthLoginChallenge AuthCmd = iota + 0x00
	AuthLoginProof
	AuthReconnectChallenge
	AuthReconnectProof
	RealmList AuthCmd = 0x10
)
