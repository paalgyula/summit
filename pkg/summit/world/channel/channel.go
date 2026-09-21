// Package channel implements WoW 3.3.5a chat channels.
//
// Channel types are classified by flag bits. Built-in channels (Trade,
// General, City, LFG) are created from ChatChannelsEntry DBC data;
// custom channels are player-created and optionally persisted.
package channel

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/wow"
)

// Flag describes the type of a channel.
type Flag uint8

const (
	FlagGeneral  Flag = 0x01
	FlagTrade    Flag = 0x02
	FlagCity     Flag = 0x04
	FlagLFG      Flag = 0x08
	FlagNotLFG   Flag = 0x10
	FlagCustom   Flag = 0x20
)

// Member flags for PlayerInfo.Flags.
const (
	MemberFlagNone       uint8 = 0
	MemberFlagOwner      uint8 = 0x01
	MemberFlagModerator  uint8 = 0x02
	MemberFlagMuted      uint8 = 0x04
	MemberFlagBanned     uint8 = 0x08
)

var (
	ErrNotMember       = errors.New("not a channel member")
	ErrMuted           = errors.New("muted in this channel")
	ErrWrongPassword   = errors.New("incorrect password")
	ErrBanned          = errors.New("banned from this channel")
	ErrChannelFull     = errors.New("channel is full")
	ErrNotModerator    = errors.New("not a moderator")
	ErrEmptyName       = errors.New("channel name cannot be empty")
)

// PlayerInfo tracks one player's state inside a channel.
type PlayerInfo struct {
	GUID     wow.GUID
	Flags    uint8
	JoinedAt time.Time
}

// IsOwner returns true if the player is the channel owner.
func (pi *PlayerInfo) IsOwner() bool { return pi.Flags&MemberFlagOwner != 0 }

// IsModerator returns true if the player can moderate.
func (pi *PlayerInfo) IsModerator() bool { return pi.Flags&MemberFlagModerator != 0 }

// IsMuted returns true if the player is muted.
func (pi *PlayerInfo) IsMuted() bool { return pi.Flags&MemberFlagMuted != 0 }

// Channel represents a single chat channel (built-in or custom).
type Channel struct {
	Name     string
	ID       uint32 // non-zero for built-in channels
	Flags    Flag
	Team     int // 0 = neutral, 1 = alliance, 2 = horde
	Password string
	Announce bool

	mu      sync.RWMutex
	players map[wow.GUID]*PlayerInfo
	banned  map[wow.GUID]time.Time
}

// New creates a channel. Built-in channels (id != 0) disable announces
// and ownership; custom channels default to announce-on.
func New(name string, id uint32, team int, flags Flag) *Channel {
	ch := &Channel{
		Name:     name,
		ID:       id,
		Team:     team,
		Flags:    flags,
		Announce: flags&FlagCustom != 0,
		players:  make(map[wow.GUID]*PlayerInfo),
		banned:   make(map[wow.GUID]time.Time),
	}

	if flags&FlagCustom == 0 {
		ch.Announce = false // built-in channels don't announce joins/leaves
	}

	return ch
}

// Join adds a player to the channel.
func (ch *Channel) Join(guid wow.GUID, password string) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.Password != "" && ch.Password != password {
		return ErrWrongPassword
	}

	if banExpiry, ok := ch.banned[guid]; ok {
		if time.Now().Before(banExpiry) {
			return ErrBanned
		}

		delete(ch.banned, guid)
	}

	if _, ok := ch.players[guid]; ok {
		return nil // already in channel
	}

	ch.players[guid] = &PlayerInfo{
		GUID:     guid,
		JoinedAt: time.Now(),
	}

	// First player becomes owner of custom channels
	if ch.Flags&FlagCustom != 0 && len(ch.players) == 1 {
		ch.players[guid].Flags = MemberFlagOwner | MemberFlagModerator
	}

	return nil
}

// Leave removes a player from the channel.
func (ch *Channel) Leave(guid wow.GUID) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	delete(ch.players, guid)
}

// Say broadcasts a message to all members except the sender (or all if
// sender is zero). Returns the serialized packet payload for each recipient.
func (ch *Channel) Say(guid wow.GUID, msg string) error {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if _, ok := ch.players[guid]; !ok {
		return ErrNotMember
	}

	if ch.players[guid].IsMuted() {
		return ErrMuted
	}

	// Message broadcast is handled by the caller using ForEachPlayer.
	return nil
}

// Kick removes a target from the channel. Caller must be moderator+.
func (ch *Channel) Kick(caller, target wow.GUID) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	callerInfo, ok := ch.players[caller]
	if !ok || !callerInfo.IsModerator() {
		return ErrNotModerator
	}

	if _, ok := ch.players[target]; !ok {
		return ErrNotMember
	}

	delete(ch.players, target)

	return nil
}

// Ban removes a target and bans them for the given duration.
func (ch *Channel) Ban(caller, target wow.GUID, duration time.Duration) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	callerInfo, ok := ch.players[caller]
	if !ok || !callerInfo.IsModerator() {
		return ErrNotModerator
	}

	delete(ch.players, target)
	ch.banned[target] = time.Now().Add(duration)

	return nil
}

// UnBan removes a ban for the given GUID.
func (ch *Channel) UnBan(caller, target wow.GUID) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	callerInfo, ok := ch.players[caller]
	if !ok || !callerInfo.IsModerator() {
		return ErrNotModerator
	}

	delete(ch.banned, target)

	return nil
}

// SetPassword sets the channel password. Caller must be owner.
func (ch *Channel) SetPassword(caller wow.GUID, pass string) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	callerInfo, ok := ch.players[caller]
	if !ok || !callerInfo.IsOwner() {
		return ErrNotModerator
	}

	ch.Password = pass

	return nil
}

// SetModerator grants or revokes moderator status. Caller must be owner.
func (ch *Channel) SetModerator(caller, target wow.GUID, set bool) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	callerInfo, ok := ch.players[caller]
	if !ok || !callerInfo.IsOwner() {
		return ErrNotModerator
	}

	targetInfo, ok := ch.players[target]
	if !ok {
		return ErrNotMember
	}

	if set {
		targetInfo.Flags |= MemberFlagModerator
	} else {
		targetInfo.Flags &^= MemberFlagModerator
	}

	return nil
}

// SetMute mutes or unmutes a player. Caller must be moderator.
func (ch *Channel) SetMute(caller, target wow.GUID, set bool) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	callerInfo, ok := ch.players[caller]
	if !ok || !callerInfo.IsModerator() {
		return ErrNotModerator
	}

	targetInfo, ok := ch.players[target]
	if !ok {
		return ErrNotMember
	}

	if set {
		targetInfo.Flags |= MemberFlagMuted
	} else {
		targetInfo.Flags &^= MemberFlagMuted
	}

	return nil
}

// IsOn returns true if the player is in the channel.
func (ch *Channel) IsOn(guid wow.GUID) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	_, ok := ch.players[guid]

	return ok
}

// PlayerCount returns the number of players in the channel.
func (ch *Channel) PlayerCount() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return len(ch.players)
}

// GetPlayerFlags returns the member flags for a player, or MemberFlagNone.
func (ch *Channel) GetPlayerFlags(guid wow.GUID) uint8 {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if pi, ok := ch.players[guid]; ok {
		return pi.Flags
	}

	return MemberFlagNone
}

// ForEachPlayer calls fn for each player in the channel while holding the read lock.
func (ch *Channel) ForEachPlayer(fn func(guid wow.GUID, info *PlayerInfo)) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	for guid, info := range ch.players {
		fn(guid, info)
	}
}

// SetOwner transfers ownership to the given GUID.
func (ch *Channel) SetOwner(newOwner wow.GUID) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Clear old owner
	for _, pi := range ch.players {
		pi.Flags &^= MemberFlagOwner
	}

	if pi, ok := ch.players[newOwner]; ok {
		pi.Flags |= MemberFlagOwner | MemberFlagModerator
	}
}

// NameLower returns the lowercase channel name (for map key use).
func NameLower(name string) string {
	return strings.ToLower(name)
}
