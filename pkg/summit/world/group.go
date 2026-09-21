package world

import (
	"sync"

	"github.com/paalgyula/summit/pkg/wow"
)

// GroupType identifies the kind of group.
type GroupType uint8

const (
	GroupTypeParty GroupType = 0
	GroupTypeRaid  GroupType = 1
	GroupTypeBG    GroupType = 2
	GroupTypeArena GroupType = 3
)

// Group represents a party, raid, battleground, or arena team.
type Group struct {
	mu       sync.RWMutex
	ID       uint32
	Type     GroupType
	LeaderID wow.GUID
	Members  []wow.GUID
}

// NewGroup creates a group with the given leader.
func NewGroup(id uint32, gtype GroupType, leaderGUID wow.GUID) *Group {
	return &Group{
		ID:       id,
		Type:     gtype,
		LeaderID: leaderGUID,
		Members:  []wow.GUID{leaderGUID},
	}
}

// AddMember adds a player GUID to the group.
func (g *Group) AddMember(guid wow.GUID) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, m := range g.Members {
		if m == guid {
			return
		}
	}

	g.Members = append(g.Members, guid)
}

// RemoveMember removes a player GUID from the group.
func (g *Group) RemoveMember(guid wow.GUID) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, m := range g.Members {
		if m == guid {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)

			return
		}
	}
}

// IsMember returns true if the GUID is in the group.
func (g *Group) IsMember(guid wow.GUID) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, m := range g.Members {
		if m == guid {
			return true
		}
	}

	return false
}

// IsLeader returns true if the GUID is the group leader.
func (g *Group) IsLeader(guid wow.GUID) bool {
	return g.LeaderID == guid
}

// MemberCount returns the number of members.
func (g *Group) MemberCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.Members)
}

// IsRaid returns true if the group type is raid.
func (g *Group) IsRaid() bool {
	return g.Type == GroupTypeRaid
}

// IsBG returns true if the group type is battleground.
func (g *Group) IsBG() bool {
	return g.Type == GroupTypeBG
}
