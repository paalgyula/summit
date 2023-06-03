// Code generated by "stringer -type=RealmCommand -output=commands_string.go"; DO NOT EDIT.

package auth

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[AuthLoginChallenge-0]
	_ = x[AuthLoginProof-1]
	_ = x[AuthReconnectChallenge-2]
	_ = x[AuthReconnectProof-3]
	_ = x[RealmList-16]
}

const (
	_RealmCommand_name_0 = "AuthLoginChallengeAuthLoginProofAuthReconnectChallengeAuthReconnectProof"
	_RealmCommand_name_1 = "RealmList"
)

var (
	_RealmCommand_index_0 = [...]uint8{0, 18, 32, 54, 72}
)

func (i RealmCommand) String() string {
	switch {
	case i <= 3:
		return _RealmCommand_name_0[_RealmCommand_index_0[i]:_RealmCommand_index_0[i+1]]
	case i == 16:
		return _RealmCommand_name_1
	default:
		return "RealmCommand(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}