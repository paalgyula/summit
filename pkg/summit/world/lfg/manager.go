package lfg

import (
	"sync"
	"time"
)

// Manager coordinates the LFG queue and matchmaking.
type Manager struct {
	mu        sync.RWMutex
	players   map[uint64]*PlayerData
	groups    map[uint64]*GroupData
	queue     []*QueueEntry
	dungeons  map[uint32]*Dungeon
	proposal  *ProposalData
	nextGroup uint32
}

// NewManager creates an LFG manager with the given dungeon definitions.
func NewManager(dungeons []*Dungeon) *Manager {
	dm := make(map[uint32]*Dungeon, len(dungeons))
	for _, d := range dungeons {
		dm[d.ID] = d
	}

	return &Manager{
		players:  make(map[uint64]*PlayerData),
		groups:   make(map[uint64]*GroupData),
		dungeons: dm,
	}
}

// JoinQueue adds a player (or group) to the LFG queue.
func (m *Manager) JoinQueue(guid uint64, groupGUID uint64, roles uint32, dungeonID uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.players[guid] = &PlayerData{
		GUID:            guid,
		GroupGUID:       groupGUID,
		State:           StateQueued,
		Roles:           roles,
		SelectedDungeon: dungeonID,
		OldState:        StateNone,
	}

	entry := &QueueEntry{
		GUID:      guid,
		GroupGUID: groupGUID,
		Roles:     roles,
		DungeonID: dungeonID,
		QueuedAt:  time.Now(),
		Ready:     true,
	}

	m.queue = append(m.queue, entry)

	return nil
}

// LeaveQueue removes a player from the LFG queue.
func (m *Manager) LeaveQueue(guid uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pd, ok := m.players[guid]; ok {
		pd.State = StateNone
	}

	for i, entry := range m.queue {
		if entry.GUID == guid {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)

			break
		}
	}

	return nil
}

// SetRoles updates the role selection for a player.
func (m *Manager) SetRoles(guid uint64, roles uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pd, ok := m.players[guid]; ok {
		pd.Roles = roles
	}

	return nil
}

// ProposalResponse handles a player's accept/decline for a dungeon proposal.
func (m *Manager) ProposalResponse(guid uint64, accept bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.proposal == nil {
		return nil
	}

	m.proposal.Accepted[guid] = accept

	// Check if all players have responded
	allResponded := true
	allAccepted := true

	for _, pGUID := range m.proposal.Players {
		acc, ok := m.proposal.Accepted[pGUID]
		if !ok {
			allResponded = false

			break
		}

		if !acc {
			allAccepted = false
		}
	}

	if allResponded {
		if allAccepted {
			// Move all players to StateInDungeon
			for _, pGUID := range m.proposal.Players {
				if pd, ok := m.players[pGUID]; ok {
					pd.State = StateInDungeon
					pd.OldState = StateQueued
				}
			}
		} else {
			// Proposal failed, revert to queued
			for _, pGUID := range m.proposal.Players {
				if pd, ok := m.players[pGUID]; ok {
					pd.State = StateQueued
				}
			}
		}

		m.proposal = nil
	}

	return nil
}

// GetCurrentProposal returns the active proposal, or nil.
func (m *Manager) GetCurrentProposal() *ProposalData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.proposal
}

// GetPlayerData returns the LFG data for a player.
func (m *Manager) GetPlayerData(guid uint64) *PlayerData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.players[guid]
}

// GetDungeon returns dungeon info by ID.
func (m *Manager) GetDungeon(id uint32) *Dungeon {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.dungeons[id]
}

// QueueSize returns the number of entries in the queue.
func (m *Manager) QueueSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.queue)
}

// Update runs periodic queue processing. Call this from the world tick.
func (m *Manager) Update(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple matching: find groups of 5 (or solo) for the same dungeon
	// This is a placeholder — real matchmaking is more complex.
	m.tryMatch()
}

// tryMatch attempts to form a group from queued players.
func (m *Manager) tryMatch() {
	if len(m.queue) < 5 {
		return
	}

	// Group by dungeon ID
	byDungeon := make(map[uint32][]*QueueEntry)

	for _, entry := range m.queue {
		if entry.Ready {
			byDungeon[entry.DungeonID] = append(byDungeon[entry.DungeonID], entry)
		}
	}

	for dungeonID, entries := range byDungeon {
		if len(entries) >= 5 {
			// Form a group from the first 5 entries
			players := make([]uint64, 5)

			for i := 0; i < 5; i++ {
				players[i] = entries[i].GUID
			}

			m.proposal = &ProposalData{
				DungeonID: dungeonID,
				Players:   players,
				Accepted:  make(map[uint64]bool),
				CreatedAt: time.Now(),
			}

			// Remove matched entries from queue
			matched := make(map[uint64]bool)
			for _, p := range players {
				matched[p] = true
			}

			newQueue := make([]*QueueEntry, 0, len(m.queue)-5)

			for _, entry := range m.queue {
				if !matched[entry.GUID] {
					newQueue = append(newQueue, entry)
				}
			}

			m.queue = newQueue

			// Set players to proposal state
			for _, p := range players {
				if pd, ok := m.players[p]; ok {
					pd.State = StateProposal
				}
			}

			break
		}
	}
}

// FinishDungeon marks a player as having finished their dungeon run.
func (m *Manager) FinishDungeon(guid uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pd, ok := m.players[guid]; ok {
		pd.State = StateFinishedDungeon
		pd.OldState = StateInDungeon
	}
}

// ResetPlayer clears a player's LFG state.
func (m *Manager) ResetPlayer(guid uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.players, guid)
}
