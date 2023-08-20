//go:generate stringer -type=RealmCommand -output=commands_string.go
package auth

type RealmCommand uint32

const (
	AuthLoginChallenge RealmCommand = iota + 0x00
	AuthLoginProof
	AuthReconnectChallenge // #4 implement reconnect challenge and reconnect proof
	AuthReconnectProof
	RealmList RealmCommand = 0x10
)
