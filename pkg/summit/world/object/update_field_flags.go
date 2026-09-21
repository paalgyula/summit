package object

// Update field visibility flags — matches AzerothCore's UpdatefieldFlags enum.
const (
	UFFlagNone         uint32 = 0x000
	UFFlagPublic       uint32 = 0x001
	UFFlagPrivate      uint32 = 0x002
	UFFlagOwner        uint32 = 0x004
	UFFlagUnused1      uint32 = 0x008
	UFFlagItemOwner    uint32 = 0x010
	UFFlagSpecialInfo  uint32 = 0x020
	UFFlagPartyMember  uint32 = 0x040
	UFFlagUnused2      uint32 = 0x080
	UFFlagDynamic      uint32 = 0x100
)

// ObjectFieldFlags covers fields [0, ObjectEnd).
var ObjectFieldFlags = [ObjectEnd]uint32{
	UFFlagPublic, // OBJECT_FIELD_GUID
	UFFlagPublic, // OBJECT_FIELD_GUID+1
	UFFlagPublic, // OBJECT_FIELD_TYPE
	UFFlagPublic, // OBJECT_FIELD_ENTRY
	UFFlagPublic, // OBJECT_FIELD_SCALE_X
	UFFlagNone,   // OBJECT_FIELD_PADDING
}

// ItemUpdateFieldFlags covers fields [ObjectEnd, ContainerEnd).
var ItemUpdateFieldFlags = [ContainerEnd - ObjectEnd]uint32{
	// -- ITEM_FIELD_* --
	UFFlagPublic,                                // ITEM_FIELD_OWNER
	UFFlagPublic,                                // ITEM_FIELD_OWNER+1
	UFFlagPublic,                                // ITEM_FIELD_CONTAINED
	UFFlagPublic,                                // ITEM_FIELD_CONTAINED+1
	UFFlagPublic,                                // ITEM_FIELD_CREATOR
	UFFlagPublic,                                // ITEM_FIELD_CREATOR+1
	UFFlagPublic,                                // ITEM_FIELD_GIFTCREATOR
	UFFlagPublic,                                // ITEM_FIELD_GIFTCREATOR+1
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_STACK_COUNT
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_DURATION
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_SPELL_CHARGES
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_SPELL_CHARGES+1
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_SPELL_CHARGES+2
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_SPELL_CHARGES+3
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_SPELL_CHARGES+4
	UFFlagPublic,                                // ITEM_FIELD_FLAGS
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_1_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_1_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_1_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_2_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_2_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_2_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_3_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_3_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_3_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_4_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_4_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_4_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_5_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_5_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_5_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_6_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_6_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_6_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_7_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_7_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_7_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_8_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_8_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_8_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_9_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_9_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_9_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_10_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_10_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_10_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_11_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_11_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_11_3
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_12_1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_12_1+1
	UFFlagPublic,                                // ITEM_FIELD_ENCHANTMENT_12_3
	UFFlagPublic,                                // ITEM_FIELD_PROPERTY_SEED
	UFFlagPublic,                                // ITEM_FIELD_RANDOM_PROPERTIES_ID
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_DURABILITY
	UFFlagOwner | UFFlagItemOwner,               // ITEM_FIELD_MAXDURABILITY
	UFFlagPublic,                                // ITEM_FIELD_CREATE_PLAYED_TIME
	UFFlagNone,                                  // ITEM_FIELD_PAD
	// -- CONTAINER_FIELD_* --
	UFFlagPublic, // CONTAINER_FIELD_NUM_SLOTS
	UFFlagNone,   // CONTAINER_ALIGN_PAD
	// CONTAINER_FIELD_SLOT_1: 72 LONG fields (72 entries)
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
}

// UnitUpdateFieldFlags covers fields [ObjectEnd, PlayerEnd) — unit + player portion.
// Index 0 corresponds to UnitFieldCharm (ObjectEnd+0).
var UnitUpdateFieldFlags = [PlayerEnd - ObjectEnd]uint32{
	// UNIT_FIELD_CHARM (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_SUMMON (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_CRITTER (2 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate,
	// UNIT_FIELD_CHARMEDBY (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_SUMMONEDBY (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_CREATEDBY (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_TARGET (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_CHANNEL_OBJECT (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// UNIT_CHANNEL_SPELL
	UFFlagPublic,
	// UNIT_FIELD_BYTES_0
	UFFlagPublic,
	// UNIT_FIELD_HEALTH
	UFFlagPublic,
	// UNIT_FIELD_POWER1..POWER7
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_MAXHEALTH
	UFFlagPublic,
	// UNIT_FIELD_MAXPOWER1..MAXPOWER7
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_POWER_REGEN_FLAT_MODIFIER (7 FLOAT) — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_POWER_REGEN_INTERRUPTED_FLAT_MODIFIER (7 FLOAT) — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_LEVEL
	UFFlagPublic,
	// UNIT_FIELD_FACTIONTEMPLATE
	UFFlagPublic,
	// UNIT_VIRTUAL_ITEM_SLOT_ID (3 INT)
	UFFlagPublic, UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_FLAGS
	UFFlagPublic,
	// UNIT_FIELD_FLAGS_2
	UFFlagPublic,
	// UNIT_FIELD_AURASTATE
	UFFlagPublic,
	// UNIT_FIELD_BASEATTACKTIME (2 INT)
	UFFlagPublic, UFFlagPublic,
	// UNIT_FIELD_RANGEDATTACKTIME — PRIVATE
	UFFlagPrivate,
	// UNIT_FIELD_BOUNDINGRADIUS
	UFFlagPublic,
	// UNIT_FIELD_COMBATREACH
	UFFlagPublic,
	// UNIT_FIELD_DISPLAYID
	UFFlagPublic,
	// UNIT_FIELD_NATIVEDISPLAYID
	UFFlagPublic,
	// UNIT_FIELD_MOUNTDISPLAYID
	UFFlagPublic,
	// UNIT_FIELD_MINDAMAGE — PRIVATE|OWNER|SPECIAL_INFO
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	// UNIT_FIELD_MAXDAMAGE
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	// UNIT_FIELD_MINOFFHANDDAMAGE
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	// UNIT_FIELD_MAXOFFHANDDAMAGE
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	// UNIT_FIELD_BYTES_1
	UFFlagPublic,
	// UNIT_FIELD_PETNUMBER
	UFFlagPublic,
	// UNIT_FIELD_PET_NAME_TIMESTAMP
	UFFlagPublic,
	// UNIT_FIELD_PETEXPERIENCE — OWNER
	UFFlagOwner,
	// UNIT_FIELD_PETNEXTLEVELEXP — OWNER
	UFFlagOwner,
	// UNIT_DYNAMIC_FLAGS — DYNAMIC
	UFFlagDynamic,
	// UNIT_MOD_CAST_SPEED
	UFFlagPublic,
	// UNIT_CREATED_BY_SPELL
	UFFlagPublic,
	// UNIT_NPC_FLAGS — DYNAMIC
	UFFlagDynamic,
	// UNIT_NPC_EMOTESTATE
	UFFlagPublic,
	// UNIT_FIELD_STAT0..STAT4 — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_POSSTAT0..POSSTAT4 — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_NEGSTAT0..NEGSTAT4 — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_RESISTANCES (7 INT) — PRIVATE|OWNER|SPECIAL_INFO
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	UFFlagPrivate | UFFlagOwner | UFFlagSpecialInfo,
	// UNIT_FIELD_RESISTANCEBUFFMODSPOSITIVE (7 INT) — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_RESISTANCEBUFFMODSNEGATIVE (7 INT) — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_BASE_MANA
	UFFlagPublic,
	// UNIT_FIELD_BASE_HEALTH — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_BYTES_2
	UFFlagPublic,
	// UNIT_FIELD_ATTACK_POWER — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_ATTACK_POWER_MODS — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_ATTACK_POWER_MULTIPLIER — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_RANGED_ATTACK_POWER — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_RANGED_ATTACK_POWER_MODS — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_RANGED_ATTACK_POWER_MULTIPLIER — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_MINRANGEDDAMAGE — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_MAXRANGEDDAMAGE — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_POWER_COST_MODIFIER (7 INT) — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_POWER_COST_MULTIPLIER (7 FLOAT) — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner, UFFlagPrivate | UFFlagOwner,
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_MAXHEALTHMODIFIER — PRIVATE|OWNER
	UFFlagPrivate | UFFlagOwner,
	// UNIT_FIELD_HOVERHEIGHT
	UFFlagPublic,
	// UNIT_FIELD_PADDING
	UFFlagNone,
	// --- Player fields (UnitEnd..PlayerEnd) ---
	// PLAYER_DUEL_ARBITER (2 LONG)
	UFFlagPublic, UFFlagPublic,
	// PLAYER_FLAGS
	UFFlagPublic,
	// PLAYER_GUILDID
	UFFlagPublic,
	// PLAYER_GUILDRANK
	UFFlagPublic,
	// PLAYER_BYTES
	UFFlagPublic,
	// PLAYER_BYTES_2
	UFFlagPublic,
	// PLAYER_BYTES_3
	UFFlagPublic,
	// PLAYER_DUEL_TEAM
	UFFlagPublic,
	// PLAYER_GUILD_TIMESTAMP
	UFFlagPublic,
	// PLAYER_QUEST_LOG_1..25 — 5 fields each (PARTY_MEMBER, PRIVATE, PRIVATE(2SHORT), PRIVATE)
	// Quest 1
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 2
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 3
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 4
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 5
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 6
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 7
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 8
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 9
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 10
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 11
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 12
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 13
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 14
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 15
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 16
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 17
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 18
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 19
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 20
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 21
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 22
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 23
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 24
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// Quest 25
	UFFlagPartyMember, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// PLAYER_VISIBLE_ITEM_1..19 (2 fields each: ENTRYID + ENCHANTMENT) — PUBLIC
	UFFlagPublic, UFFlagPublic, // 1
	UFFlagPublic, UFFlagPublic, // 2
	UFFlagPublic, UFFlagPublic, // 3
	UFFlagPublic, UFFlagPublic, // 4
	UFFlagPublic, UFFlagPublic, // 5
	UFFlagPublic, UFFlagPublic, // 6
	UFFlagPublic, UFFlagPublic, // 7
	UFFlagPublic, UFFlagPublic, // 8
	UFFlagPublic, UFFlagPublic, // 9
	UFFlagPublic, UFFlagPublic, // 10
	UFFlagPublic, UFFlagPublic, // 11
	UFFlagPublic, UFFlagPublic, // 12
	UFFlagPublic, UFFlagPublic, // 13
	UFFlagPublic, UFFlagPublic, // 14
	UFFlagPublic, UFFlagPublic, // 15
	UFFlagPublic, UFFlagPublic, // 16
	UFFlagPublic, UFFlagPublic, // 17
	UFFlagPublic, UFFlagPublic, // 18
	UFFlagPublic, UFFlagPublic, // 19
	// PLAYER_CHOSEN_TITLE
	UFFlagPublic,
	// PLAYER_FAKE_INEBRIATION
	UFFlagPublic,
	// PLAYER_FIELD_PAD_0
	UFFlagNone,
	// PLAYER_FIELD_INV_SLOT_HEAD (46 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_PACK_SLOT_1 (32 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_BANK_SLOT_1 (56 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_BANKBAG_SLOT_1 (14 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_VENDORBUYBACK_SLOT_1 (24 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_KEYRING_SLOT_1 (64 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_CURRENCYTOKEN_SLOT_1 (64 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	UFFlagPrivate, UFFlagPrivate, UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FARSIGHT (2 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_KNOWN_TITLES (2 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_KNOWN_TITLES1 (2 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_KNOWN_TITLES2 (2 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate,
	// PLAYER_FIELD_KNOWN_CURRENCIES (2 LONG) — PRIVATE
	UFFlagPrivate, UFFlagPrivate,
	// PLAYER_XP
	UFFlagPrivate,
	// PLAYER_NEXT_LEVEL_XP
	UFFlagPrivate,
	// PLAYER_SKILL_INFO_1_1 (384 TWO_SHORT) — PRIVATE
	// (384 entries — all PRIVATE)
	// Emitting as a block for brevity — generated programmatically below.
}

func init() {
	// Fill the tail of UnitUpdateFieldFlags (PLAYER_SKILL_INFO_1_1 through PlayerEnd) with PRIVATE.
	// PLAYER_SKILL_INFO_1_1 starts at UnitEnd + 0x01E8, size 384.
	// Index relative to ObjectEnd: (UnitFieldSkillInfo1_1 - ObjectEnd).
	skillInfoStart := int(PlayerSkillInfo1_1 - ObjectEnd)
	for i := skillInfoStart; i < len(UnitUpdateFieldFlags); i++ {
		UnitUpdateFieldFlags[i] = UFFlagPrivate
	}
}

// GameObjectFieldFlags covers fields [ObjectEnd, GameobjectEnd).
var GameObjectFieldFlags = [GameobjectEnd - ObjectEnd]uint32{
	UFFlagPublic, UFFlagPublic, // OBJECT_FIELD_CREATEDBY (2 LONG)
	UFFlagPublic,    // GAMEOBJECT_DISPLAYID
	UFFlagPublic,    // GAMEOBJECT_FLAGS
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic, // GAMEOBJECT_PARENTROTATION (4 FLOAT)
	UFFlagDynamic,   // GAMEOBJECT_DYNAMIC
	UFFlagPublic,    // GAMEOBJECT_FACTION
	UFFlagPublic,    // GAMEOBJECT_LEVEL
	UFFlagPublic,    // GAMEOBJECT_BYTES_1
}

// DynamicObjectFieldFlags covers fields [ObjectEnd, DynamicobjectEnd).
var DynamicObjectFieldFlags = [DynamicobjectEnd - ObjectEnd]uint32{
	UFFlagPublic, UFFlagPublic, // DYNAMICOBJECT_CASTER (2 LONG)
	UFFlagPublic, // DYNAMICOBJECT_BYTES
	UFFlagPublic, // DYNAMICOBJECT_SPELLID
	UFFlagPublic, // DYNAMICOBJECT_RADIUS
	UFFlagPublic, // DYNAMICOBJECT_CASTTIME
}

// CorpseUpdateFieldFlags covers fields [ObjectEnd, CorpseEnd).
var CorpseUpdateFieldFlags = [CorpseEnd - ObjectEnd]uint32{
	UFFlagPublic, UFFlagPublic, // CORPSE_FIELD_OWNER (2 LONG)
	UFFlagPublic, UFFlagPublic, // CORPSE_FIELD_PARTY (2 LONG)
	UFFlagPublic,                                      // CORPSE_FIELD_DISPLAYID
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic, // CORPSE_FIELD_ITEM (19 INT) — first 5
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, UFFlagPublic, UFFlagPublic, UFFlagPublic,
	UFFlagPublic, // CORPSE_FIELD_BYTES_1
	UFFlagPublic, // CORPSE_FIELD_BYTES_2
	UFFlagPublic, // CORPSE_FIELD_GUILD
	UFFlagPublic, // CORPSE_FIELD_FLAGS
	UFFlagDynamic, // CORPSE_FIELD_DYNAMIC_FLAGS
	UFFlagNone,    // CORPSE_FIELD_PAD
}
