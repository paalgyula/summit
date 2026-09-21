package loot

// LootMode is a bitmask controlling which loot table entries apply.
// Mirrors AzerothCore's LootModes enum (SharedDefines.h:43-48).
type LootMode uint16

const (
	LootModeDefault   LootMode = 0x01
	LootModeHardMode1 LootMode = 0x02
	LootModeHardMode2 LootMode = 0x04
	LootModeHardMode3 LootMode = 0x08
	LootModeHardMode4 LootMode = 0x10
	LootModeNoFlags   LootMode = 0x800
)

// Has returns true if m has all bits of flag set.
func (m LootMode) Has(flag LootMode) bool { return m&flag != 0 }

// LootMethod defines group distribution rules.
type LootMethod uint8

const (
	LootMethodFreeForAll      LootMethod = 0
	LootMethodRoundRobin      LootMethod = 1
	LootMethodMasterLoot      LootMethod = 2
	LootMethodGroupLoot       LootMethod = 3
	LootMethodNeedBeforeGreed LootMethod = 4
)

// LootEntry is a single row from any *_loot_template table.
// Fields match AzerothCore's LootStoreItem struct (LootMgr.h:127-149).
type LootEntry struct {
	Entry    uint32    `bson:"entry" json:"entry"`
	Item     uint32    `bson:"item" json:"item"`
	Ref      int32     `bson:"ref" json:"ref"`           // reference to reference_loot_template (0 = direct item)
	Chance   float32   `bson:"chance" json:"chance"`     // drop chance % (100 = guaranteed, 0 = equal-chanced in group)
	NeedQuest bool      `bson:"needQuest" json:"needQuest"` // only drops if player has active quest
	LootMode LootMode  `bson:"lootMode" json:"lootMode"` // bitmask of LootMode flags
	GroupID  uint8     `bson:"groupId" json:"groupId"`   // 0 = ungrouped, 1-N = grouped (one drop per group)
	MinCount uint8     `bson:"minCount" json:"minCount"` // minimum stack size
	MaxCount uint8     `bson:"maxCount" json:"maxCount"` // maximum stack size (also ref multiplicator)
}

// LootType defines the source of loot.
// Mirrors AzerothCore's LootType enum (LootMgr.h:59-72).
type LootType uint8

const (
	LootNone           LootType = 0
	LootCorpse         LootType = 1
	LootPickpocketing  LootType = 2
	LootFishing        LootType = 3
	LootDisenchanting  LootType = 4
	LootSkinning       LootType = 6
	LootProspecting    LootType = 7
	LootMilling        LootType = 8
	LootFishingHole    LootType = 20 // sent as LOOT_FISHING to client
	LootInsignia       LootType = 21 // sent as LOOT_CORPSE to client
	LootFishingJunk    LootType = 22 // sent as LOOT_FISHING to client
)

// LootSlotType defines per-slot permission in the loot response packet.
// Mirrors AzerothCore's LootSlotType enum (LootMgr.h:80-87).
type LootSlotType uint8

const (
	SlotAllowLoot    LootSlotType = 0
	SlotRollOngoing  LootSlotType = 1
	SlotMaster       LootSlotType = 2
	SlotLocked       LootSlotType = 3
	SlotOwner        LootSlotType = 4
)

// PermissionTypes determines what a player sees in the loot window.
// Mirrors AzerothCore's PermissionTypes enum (LootMgr.h:49-58).
type PermissionType uint8

const (
	AllPermission         PermissionType = 0
	GroupPermission       PermissionType = 1
	MasterPermission      PermissionType = 2
	RestrictedPermission  PermissionType = 3
	RoundRobinPermission  PermissionType = 4
	OwnerPermission       PermissionType = 5
	NonePermission        PermissionType = 6
)

// LootError defines error codes sent to the client.
// Mirrors AzerothCore's LootError enum (LootMgr.h:90-105).
type LootError uint8

const (
	ErrorDidntKill         LootError = 0
	ErrorTooFar            LootError = 4
	ErrorBadFacing         LootError = 5
	ErrorLocked            LootError = 6
	ErrorNotStanding       LootError = 8
	ErrorStunned           LootError = 9
	ErrorPlayerNotFound    LootError = 10
	ErrorPlayTimeExceeded  LootError = 11
	ErrorMasterInvFull     LootError = 12
	ErrorMasterUniqueItem  LootError = 13
	ErrorMasterOther       LootError = 14
	ErrorAlreadyPickpocketed LootError = 15
	ErrorNotWhileShapeshifted LootError = 16
)

// RollType defines possible roll outcomes.
type RollType uint8

const (
	RollPass       RollType = 0
	RollNeed       RollType = 1
	RollGreed      RollType = 2
	RollDisenchant RollType = 3
)

// Limits matching AzerothCore (LootMgr.h:35-36).
const (
	MaxLootItems   = 18 // Client can show 18 items in 3.3.5a
	MaxQuestItems  = 32 // Reserve for quest items
)
