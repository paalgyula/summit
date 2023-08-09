package bitmask

type Bitmask interface {
	uint8 | uint16 | uint32 | uint64
}

// Returns base with flag applied
func With[F Bitmask](base F, flag F) F {
	return base | flag
}

// Returns base with flag removed
func Without[F Bitmask](base F, flag F) F {
	return base &^ flag
}

// Returns true if base contains all flags in flag
func HasAll[F Bitmask](base F, flag F) bool {
	return base&flag == flag
}

// Returns true if base contains at least one flag in flag (usually the only flag)
func HasOne[F Bitmask](base F, flag F) bool {
	return base&flag != 0
}

// Returns a flag active at the position/index idx (starting at 0, from right to left)
// e.g., FlagAt(3) returns 0000 1000, and FlagAt(0) returns 0000 0001
func FlagAt[F Bitmask](idx F) F {
	return 1 << idx
}
