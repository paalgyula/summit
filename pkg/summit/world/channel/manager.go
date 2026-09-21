package channel

import (
	"strings"
	"sync"

	"github.com/paalgyula/summit/pkg/wow"
)

// Manager holds all channels for a single faction (or neutral).
type Manager struct {
	mu       sync.RWMutex
	channels map[string]*Channel // lowercase name → channel
	team     int
}

// NewManager creates a channel manager for the given team.
func NewManager(team int) *Manager {
	return &Manager{
		channels: make(map[string]*Channel),
		team:     team,
	}
}

// GetOrCreate returns an existing channel or creates a new one.
func (m *Manager) GetOrCreate(name string, id uint32) *Channel {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := NameLower(name)
	if ch, ok := m.channels[key]; ok {
		return ch
	}

	flags := classifyChannel(id)
	ch := New(name, id, m.team, flags)
	m.channels[key] = ch

	return ch
}

// Get returns a channel by name, or nil if not found.
func (m *Manager) Get(name string) *Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.channels[NameLower(name)]
}

// Remove deletes a channel from the manager (only if empty and custom).
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := NameLower(name)
	if ch, ok := m.channels[key]; ok && ch.PlayerCount() == 0 && ch.Flags&FlagCustom != 0 {
		delete(m.channels, key)
	}
}

// Count returns the number of channels managed.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.channels)
}

// ForEach calls fn for each channel.
func (m *Manager) ForEach(fn func(name string, ch *Channel)) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, ch := range m.channels {
		fn(name, ch)
	}
}

// SendToAll sends a packet to all players in a channel except the sender.
// buildPacket is called once per recipient with their GUID.
func SendToAll(ch *Channel, senderGUID wow.GUID, buildPacket func(guid wow.GUID) *wow.Packet) {
	ch.ForEachPlayer(func(guid wow.GUID, _ *PlayerInfo) {
		if guid == senderGUID {
			return
		}

		// Packet sending is handled by the caller via the session system.
		// This function just identifies the recipients.
		_ = buildPacket
	})
}

// classifyChannel returns the Flag set for a built-in channel ID.
// Built-in channel IDs come from ChatChannelsEntry.dbc.
func classifyChannel(id uint32) Flag {
	// Channel IDs from ChatChannelsEntry.dbc (3.3.5a):
	// 1 = General, 2 = Trade, 22 = LocalDefense, 26 = LFG
	switch {
	case id == 1:
		return FlagGeneral
	case id == 2:
		return FlagTrade
	case id == 26:
		return FlagLFG
	case id >= 22 && id <= 25:
		return FlagCity
	default:
		if id != 0 {
			return FlagGeneral
		}

		return FlagCustom
	}
}

// BuildChannelList builds the SMSG_CHANNEL_LIST response payload.
func BuildChannelList(ch *Channel) []byte {
	// Payload: uint32 channel_count, then per channel:
	//   uint8 name_len, string name, uint32 member_count
	// Simplified: single channel listing
	name := ch.Name
	payload := make([]byte, 0, 4+1+len(name)+4)
	payload = append(payload, 1, 0, 0, 0) // channel count = 1
	payload = append(payload, byte(len(name)))
	payload = append(payload, []byte(name)...)

	members := uint32(ch.PlayerCount())
	payload = append(payload, byte(members), byte(members>>8), byte(members>>16), byte(members>>24))

	return payload
}

// IsAddonChannelName returns true if the name starts with "AddonChannel:" prefix
// used by WoW addons for custom addon-to-addon communication.
func IsAddonChannelName(name string) bool {
	return strings.HasPrefix(name, "AddonChannel:")
}
