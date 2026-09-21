// Package lfg implements the WoW 3.3.5a Looking For Group system.
//
// The LFG system manages dungeon queues, role validation, and
// matchmaking for players seeking dungeon groups.
package lfg

import "time"

// State represents the LFG state of a player or group.
type State int

const (
	StateNone              State = 0
	StateQueued            State = 1
	StateProposal          State = 2
	StateRoleCheck         State = 3
	StateBootProposing     State = 4
	StateInDungeon         State = 5
	StateFinishedDungeon   State = 6
	StateRaised            State = 7
	StateOff               State = 8
)

// Role bit flags for the role selection UI.
const (
	RoleTank    uint32 = 1
	RoleHealer  uint32 = 2
	RoleDPS     uint32 = 4
)

// JoinResult codes sent in SMSG_LFG_JOIN_RESULT.
const (
	JoinResultOk                     uint32 = 0x00
	JoinResultDungeonUnobtainable    uint32 = 0x01
	JoinResultMaxLevel               uint32 = 0x02
	JoinResultMinLevel               uint32 = 0x03
	JoinResultRoleUnobtainable       uint32 = 0x04
	JoinResultLFGDisabled            uint32 = 0x05
	JoinResultSuspended              uint32 = 0x06
	JoinResultNoSlots                uint32 = 0x07
	JoinResultPartyFull              uint32 = 0x08
	JoinResultLFGAlreadyInQueue      uint32 = 0x09
	JoinResultLFGAlreadyInGroup      uint32 = 0x0A
	JoinResultLFGJoinFailed          uint32 = 0x0B
	JoinResultPartyMemberInLFG       uint32 = 0x0C
	JoinResultPartyMemberOffline     uint32 = 0x0D
	JoinResultPartyMemberNotInVanilla uint32 = 0x0E
	JoinResultLFGMemberIgnored       uint32 = 0x0F
)

// DungeonType identifies the kind of LFG dungeon.
type DungeonType uint8

const (
	DungeonTypeRandom  DungeonType = 0
	DungeonTypeSpecific DungeonType = 1
	DungeonTypeSuggested DungeonType = 2
	DungeonTypeMandom DungeonType = 3
)

// Dungeon defines an LFG dungeon entry (from LFGDungeons.dbc).
type Dungeon struct {
	ID         uint32
	Name       string
	MapID      uint32
	Difficulty uint32 // 0 = normal, 1 = heroic
	Type       DungeonType
	MinLevel   uint8
	MaxLevel   uint8
	Expansion  uint8
}

// PlayerData tracks one player's LFG state.
type PlayerData struct {
	GUID            uint64
	GroupGUID       uint64
	State           State
	Roles           uint32 // bitmask of RoleTank|RoleHealer|RoleDPS
	SelectedDungeon uint32
	OldState        State
}

// GroupData tracks one group's LFG state.
type GroupData struct {
	GUID            uint64
	LeaderGUID      uint64
	State           State
	Dungeon         uint32
	IsLFG           bool
	KicksLeft       int
	Players         map[uint64]bool
}

// QueueEntry is one item in the matchmaking queue.
type QueueEntry struct {
	GUID       uint64
	GroupGUID  uint64
	Roles      uint32
	DungeonID  uint32
	QueuedAt   time.Time
	Ready      bool
}

// ProposalData represents a dungeon proposal sent to a group.
type ProposalData struct {
	GroupGUID  uint64
	DungeonID  uint32
	Players    []uint64
	Accepted   map[uint64]bool
	CreatedAt  time.Time
}

// QueueStatusData is the payload for SMSG_LFG_QUEUE_STATUS.
type QueueStatusData struct {
	DungeonID      uint32
	WaitTimeAvg    int32
	WaitTime       int32
	WaitTimeTank   int32
	WaitTimeHealer int32
	WaitTimeDps    int32
	TanksNeeded    uint8
	HealersNeeded  uint8
	DpsNeeded      uint8
	QueuedTime     uint32
}

// RoleCheckData is the payload for SMSG_LFG_ROLE_CHECK_UPDATE.
type RoleCheckData struct {
	State    uint32
	Roles    uint32
	Members  []RoleCheckMember
}

// RoleCheckMember represents one member's role selection during role check.
type RoleCheckMember struct {
	GUID  uint64
	Role  uint32
}
