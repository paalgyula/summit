package basedata

// GOState mirrors the C++ GOState enum (GameObjectData.h).
type GOState uint8

const (
	// GOStateActive is shown as used/open and not reset (e.g. open door).
	GOStateActive GOState = 0
	// GOStateReady is shown as ready (e.g. closed door).
	GOStateReady GOState = 1
	// GOStateActiveAlternative is used in an alternate way (e.g. opened by cannon).
	GOStateActiveAlternative GOState = 2
)

// GameObjectTemplate is the domain model for a game object template.
// The Data array holds type-specific fields (Data0-Data23 from SQL).
// Field meanings match AzerothCore's GameObjectData.h union layout.
type GameObjectTemplate struct {
	Entry      uint32    `bson:"entry" json:"entry"`
	Type       uint8     `bson:"type" json:"type"`
	DisplayID  uint32    `bson:"displayId" json:"displayId"`
	Name       string    `bson:"name" json:"name"`
	IconName   string    `bson:"iconName,omitempty" json:"iconName,omitempty"`
	Size       float32   `bson:"size" json:"size"`
	Data       [24]int32 `bson:"data" json:"data"`
	AIName     string    `bson:"aiName,omitempty" json:"aiName,omitempty"`
	ScriptName string    `bson:"scriptName,omitempty" json:"scriptName,omitempty"`

	// Fields merged from `gameobject_template_addon` (AzerothCore's
	// GameObjectTemplateAddon). They default to zero when no addon row exists.
	Faction        uint32 `bson:"faction,omitempty" json:"faction,omitempty"`
	Flags          uint32 `bson:"flags,omitempty" json:"flags,omitempty"`
	MinGold        uint32 `bson:"mingold,omitempty" json:"mingold,omitempty"`
	MaxGold        uint32 `bson:"maxgold,omitempty" json:"maxgold,omitempty"`
	ArtKit         uint32 `bson:"artkit,omitempty" json:"artkit,omitempty"`
	CastBarCaption string `bson:"castBarCaption,omitempty" json:"castBarCaption,omitempty"`
	Unk1           string `bson:"unk1,omitempty" json:"unk1,omitempty"`
}

// GameObjectFlags mirrors AzerothCore's GameObjectFlags enum (SharedDefines.h).
type GameObjectFlags uint32

const (
	GOFlagInUse         GameObjectFlags = 0x00000001 // disables interaction while animated
	GOFlagLocked        GameObjectFlags = 0x00000002 // requires key/spell/event to open
	GOFlagInteractCond  GameObjectFlags = 0x00000004 // condition to interact
	GOFlagTransport     GameObjectFlags = 0x00000008 // object can transport (elevator, boat, car)
	GOFlagNotSelectable GameObjectFlags = 0x00000010 // not selectable, even in GM mode
	GOFlagNoDespawn     GameObjectFlags = 0x00000020 // never despawns, only changes state
	GOFlagTriggered     GameObjectFlags = 0x00000040 // typically summoned objects
	GOFlagDamaged       GameObjectFlags = 0x00000200
	GOFlagDestroyed     GameObjectFlags = 0x00000400
)

// Type-specific data accessors matching AzerothCore's GameObjectData.h.
// The meaning of Data[N] depends on the GO type.

// --- Door (type 0) ---
// Data[0]=startOpen, Data[1]=lockId, Data[2]=autoCloseTime
func (t *GameObjectTemplate) GetDoorStartOpen() uint32 { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetDoorLockID() uint32    { return uint32(t.Data[1]) }
func (t *GameObjectTemplate) GetDoorAutoClose() uint32 { return uint32(t.Data[2]) }

// --- Button (type 1) ---
// Data[0]=startOpen, Data[1]=lockId, Data[2]=autoCloseTime, Data[3]=linkedTrap
func (t *GameObjectTemplate) GetButtonStartOpen() uint32  { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetButtonLockID() uint32     { return uint32(t.Data[1]) }
func (t *GameObjectTemplate) GetButtonAutoClose() uint32  { return uint32(t.Data[2]) }
func (t *GameObjectTemplate) GetButtonLinkedTrap() uint32 { return uint32(t.Data[3]) }

// --- QuestGiver (type 2) ---
// Data[0]=lockId, Data[1]=questList, Data[2]=pageMaterial, Data[3]=gossipID
func (t *GameObjectTemplate) GetQuestGiverLockID() uint32   { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetQuestGiverGossipID() uint32 { return uint32(t.Data[3]) }

// --- Chest (type 3) ---
// Data[0]=lockId, Data[1]=lootId, Data[2]=chestRestockTime, Data[3]=consumable
func (t *GameObjectTemplate) GetChestLockID() uint32      { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetChestLootID() uint32      { return uint32(t.Data[1]) }
func (t *GameObjectTemplate) GetChestRestockTime() uint32 { return uint32(t.Data[2]) }
func (t *GameObjectTemplate) IsChestConsumable() bool     { return t.Data[3] != 0 }
func (t *GameObjectTemplate) GetChestEventID() uint32     { return uint32(t.Data[6]) }
func (t *GameObjectTemplate) GetChestQuestID() uint32     { return uint32(t.Data[8]) }
func (t *GameObjectTemplate) IsChestGroupLoot() bool      { return t.Data[15] != 0 }

// --- Generic (type 5) ---
// Data[0]=floatingTooltip, Data[1]=highlight, Data[2]=serverOnly, Data[5]=questID
func (t *GameObjectTemplate) IsGenericServerOnly() bool { return t.Data[2] != 0 }
func (t *GameObjectTemplate) GetGenericQuestID() int32  { return t.Data[5] }

// --- Trap (type 6) ---
// Data[0]=lockId, Data[1]=level, Data[2]=diameter, Data[3]=spellId, Data[5]=cooldown
func (t *GameObjectTemplate) GetTrapLockID() uint32   { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetTrapSpellID() uint32  { return uint32(t.Data[3]) }
func (t *GameObjectTemplate) GetTrapCooldown() uint32 { return uint32(t.Data[5]) }

// --- Goober (type 10) ---
// Data[0]=lockId, Data[1]=questId, Data[2]=eventId, Data[3]=autoCloseTime
// Data[4]=customAnim, Data[5]=consumable, Data[6]=cooldown, Data[7]=pageId
// Data[10]=spellId, Data[12]=linkedTrapId, Data[19]=gossipID
func (t *GameObjectTemplate) GetGooberLockID() uint32     { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetGooberQuestID() int32     { return t.Data[1] }
func (t *GameObjectTemplate) GetGooberEventID() uint32    { return uint32(t.Data[2]) }
func (t *GameObjectTemplate) GetGooberAutoClose() uint32  { return uint32(t.Data[3]) }
func (t *GameObjectTemplate) GetGooberCustomAnim() uint32 { return uint32(t.Data[4]) }
func (t *GameObjectTemplate) IsGooberConsumable() bool    { return t.Data[5] != 0 }
func (t *GameObjectTemplate) GetGooberCooldown() uint32   { return uint32(t.Data[6]) }
func (t *GameObjectTemplate) GetGooberPageText() uint32   { return uint32(t.Data[7]) }
func (t *GameObjectTemplate) GetGooberSpellID() uint32    { return uint32(t.Data[10]) }
func (t *GameObjectTemplate) GetGooberLinkedTrap() uint32 { return uint32(t.Data[12]) }
func (t *GameObjectTemplate) GetGooberGossipID() uint32   { return uint32(t.Data[19]) }

// --- SpellCaster (type 22) ---
// Data[0]=spellId, Data[1]=partyOnly
func (t *GameObjectTemplate) GetSpellCasterSpellID() uint32 { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) IsSpellCasterPartyOnly() bool  { return t.Data[1] != 0 }

// --- FishingHole (type 17) ---
// Data[0]=lockId, Data[1]=radius, Data[2]=lootId
func (t *GameObjectTemplate) GetFishingHoleLockID() uint32 { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetFishingHoleLootID() uint32 { return uint32(t.Data[2]) }

// --- AuctionHouse (type 20) ---
// Data[0]=factionId
func (t *GameObjectTemplate) GetAuctionHouseFaction() uint32 { return uint32(t.Data[0]) }

// --- Mailbox (type 19) ---
// (no data fields used in standard implementation)

// --- FlagStand (type 24) ---
// Data[0]=lockId
func (t *GameObjectTemplate) GetFlagStandLockID() uint32 { return uint32(t.Data[0]) }

// --- FlagDrop (type 26) ---
// Data[0]=lockId
func (t *GameObjectTemplate) GetFlagDropLockID() uint32 { return uint32(t.Data[0]) }

// --- BarberChair (type 33) ---
// Data[0]=chairHeight, Data[1]=spellId
func (t *GameObjectTemplate) GetBarberChairHeight() uint32 { return uint32(t.Data[0]) }
func (t *GameObjectTemplate) GetBarberChairSpell() uint32  { return uint32(t.Data[1]) }

// GetLockID returns the lock ID for any GO type that has one.
func (t *GameObjectTemplate) GetLockID() uint32 {
	switch t.Type {
	case 0: // Door
		return t.GetDoorLockID()
	case 1: // Button
		return t.GetButtonLockID()
	case 2: // QuestGiver
		return t.GetQuestGiverLockID()
	case 3: // Chest
		return t.GetChestLockID()
	case 6: // Trap
		return t.GetTrapLockID()
	case 10: // Goober
		return t.GetGooberLockID()
	case 12: // AreaDamage
		return uint32(t.Data[0])
	case 24: // FlagStand
		return t.GetFlagStandLockID()
	case 26: // FlagDrop
		return t.GetFlagDropLockID()
	default:
		return 0
	}
}

// GetLootID returns the loot ID for any GO type that drops loot.
func (t *GameObjectTemplate) GetLootID() uint32 {
	switch t.Type {
	case 3: // Chest
		return t.GetChestLootID()
	case 17: // FishingHole
		return t.GetFishingHoleLootID()
	default:
		return 0
	}
}

// GetAutoCloseTime returns the auto-close time (ms) for types that support it.
// Mirrors GameObjectTemplate::GetAutoCloseTime (GameObjectData.h).
func (t *GameObjectTemplate) GetAutoCloseTime() uint32 {
	switch t.Type {
	case 0: // Door
		return t.GetDoorAutoClose()
	case 1: // Button
		return t.GetButtonAutoClose()
	case 6: // Trap
		return uint32(t.Data[6])
	case 10: // Goober
		return t.GetGooberAutoClose()
	case 11: // Transport
		return uint32(t.Data[2])
	case 12: // AreaDamage
		return uint32(t.Data[5])
	default:
		return 0
	}
}

// IsDespawnAtAction reports whether the GO despawns once used (consumable
// chests and goobers). Mirrors GameObjectTemplate::IsDespawnAtAction.
func (t *GameObjectTemplate) IsDespawnAtAction() bool {
	switch t.Type {
	case 3: // Chest
		return t.IsChestConsumable()
	case 10: // Goober
		return t.IsGooberConsumable()
	default:
		return false
	}
}

// IsUsableMounted reports whether the GO may be used while mounted.
// Mirrors GameObjectTemplate::IsUsableMounted.
func (t *GameObjectTemplate) IsUsableMounted() bool {
	switch t.Type {
	case 2: // QuestGiver: Data[8]=allowMounted
		return t.Data[8] != 0
	case 9: // Text: Data[3]=allowMounted
		return t.Data[3] != 0
	case 10: // Goober: Data[17]=allowMounted
		return t.Data[17] != 0
	case 22: // SpellCaster: Data[3]=allowMounted
		return t.Data[3] != 0
	default:
		return false
	}
}

// GetCharges returns the number of uses before the GO despawns (0 = unlimited).
// Mirrors GameObjectTemplate::GetCharges.
func (t *GameObjectTemplate) GetCharges() uint32 {
	switch t.Type {
	case 21: // GuardPost: Data[1]=charges
		return uint32(t.Data[1])
	case 22: // SpellCaster: Data[1]=charges
		return uint32(t.Data[1])
	default:
		return 0
	}
}

// GetCooldown returns the cooldown (ms) discouraging repeated use of traps and
// goobers. Mirrors GameObjectTemplate::GetCooldown.
func (t *GameObjectTemplate) GetCooldown() uint32 {
	switch t.Type {
	case 6: // Trap: Data[5]=cooldown (secs)
		return uint32(t.Data[5]) * 1000
	case 10: // Goober: Data[6]=cooldown (secs)
		return uint32(t.Data[6]) * 1000
	default:
		return 0
	}
}

// GetGossipMenuID returns the gossip menu id for quest givers and goobers.
func (t *GameObjectTemplate) GetGossipMenuID() uint32 {
	switch t.Type {
	case 2: // QuestGiver: Data[3]=gossipID
		return uint32(t.Data[3])
	case 10: // Goober: Data[19]=gossipID
		return uint32(t.Data[19])
	default:
		return 0
	}
}

// GetLinkedGameObjectEntry returns the linked trap entry for types that have
// one. Mirrors GameObjectTemplate::GetLinkedGameObjectEntry.
func (t *GameObjectTemplate) GetLinkedGameObjectEntry() uint32 {
	switch t.Type {
	case 1: // Button: Data[3]=linkedTrap
		return uint32(t.Data[3])
	case 3: // Chest: Data[7]=linkedTrapId
		return uint32(t.Data[7])
	case 8: // SpellFocus: Data[2]=linkedTrapId
		return uint32(t.Data[2])
	case 10: // Goober: Data[12]=linkedTrapId
		return uint32(t.Data[12])
	default:
		return 0
	}
}

// GetSpellID returns the spell cast when the GO is used, if any.
func (t *GameObjectTemplate) GetSpellID() uint32 {
	switch t.Type {
	case 6: // Trap: Data[3]=spellId
		return t.GetTrapSpellID()
	case 10: // Goober: Data[10]=spellId
		return t.GetGooberSpellID()
	case 22: // SpellCaster: Data[0]=spellId
		return t.GetSpellCasterSpellID()
	default:
		return 0
	}
}

// GetQuestID returns the quest required to interact with the GO, or 0.
func (t *GameObjectTemplate) GetQuestID() int32 {
	switch t.Type {
	case 3: // Chest: Data[8]=questId
		return int32(t.Data[8])
	case 5: // Generic: Data[5]=questID
		return t.GetGenericQuestID()
	case 8: // SpellFocus: Data[4]=questID
		return int32(t.Data[4])
	case 10: // Goober: Data[1]=questId
		return t.GetGooberQuestID()
	default:
		return 0
	}
}

// IsGameObjectForQuests reports whether the GO participates in quest logic.
func (t *GameObjectTemplate) IsGameObjectForQuests() bool {
	switch t.Type {
	case 2, 3, 5, 8, 10: // QuestGiver, Chest, Generic, SpellFocus, Goober
		return true
	default:
		return false
	}
}

// GameObjectDynamicFlags mirrors AzerothCore's GameObjectDynamicLowFlags enum
// (SharedDefines.h). They occupy the low 16 bits of GAMEOBJECT_DYNAMIC.
type GameObjectDynamicFlags uint16

const (
	GODynFlagActivate   GameObjectDynamicFlags = 0x01 // enables interaction with the GO
	GODynFlagAnimate    GameObjectDynamicFlags = 0x02 // possibly more distinct animation
	GODynFlagNoInteract GameObjectDynamicFlags = 0x04 // appears to disable interaction
	GODynFlagSparkle    GameObjectDynamicFlags = 0x08 // makes the GO sparkle
	GODynFlagStopped    GameObjectDynamicFlags = 0x10 // transport is stopped
)

// GameObjectSpawn is the domain model for a game object spawn point.
type GameObjectSpawn struct {
	GUID          uint32     `bson:"guid" json:"guid"`
	Entry         uint32     `bson:"entry" json:"entry"`
	MapID         uint32     `bson:"map" json:"map"`
	PhaseMask     uint32     `bson:"phaseMask" json:"phaseMask"`
	PosX          float32    `bson:"positionX" json:"positionX"`
	PosY          float32    `bson:"positionY" json:"positionY"`
	PosZ          float32    `bson:"positionZ" json:"positionZ"`
	O             float32    `bson:"orientation" json:"orientation"`
	Rotation      [4]float32 `bson:"rotation" json:"rotation"`
	SpawnTimeSecs int32      `bson:"spawnTimeSecs" json:"spawnTimeSecs"`
	AnimProgress  uint8      `bson:"animProgress" json:"animProgress"`
	State         uint8      `bson:"state" json:"state"`
}
