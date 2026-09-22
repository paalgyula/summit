// Package model defines domain models and MongoDB entity transformations for
// the store layer. Entity structs map 1:1 to MongoDB documents; ToEntity/FromEntity
// functions handle conversion between domain models and entities.
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- Account ---

// AccountEntity is the MongoDB document for the accounts collection.
type AccountEntity struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	IDString  string             `bson:"id"`
	Name      string             `bson:"name"`
	Email     string             `bson:"email"`
	Salt      string             `bson:"salt"`
	Verifier  string             `bson:"verifier"`
	CreatedAt time.Time          `bson:"createdAt"`
	LastLogin *time.Time         `bson:"lastLogin,omitempty"`
	Activated bool               `bson:"activated"`
	BanReason string             `bson:"banReason,omitempty"`
	BannedAt  *time.Time         `bson:"bannedAt,omitempty"`
	BannedBy  string             `bson:"bannedBy,omitempty"`
	BanExpiry *time.Time         `bson:"banExpiry,omitempty"`
}

// CollectionName returns the MongoDB collection name.
func (AccountEntity) CollectionName() string { return "accounts" }

// --- Character ---

// CharacterEntity is the MongoDB document for the characters collection.
type CharacterEntity struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	GUID    uint32             `bson:"guid"`
	Account string             `bson:"account"`
	Name    string             `bson:"name"`
	Race    uint8              `bson:"race"`
	Class   uint8              `bson:"class"`
	Gender  uint8              `bson:"gender"`

	// Appearance
	Skin       uint8 `bson:"skin"`
	Face       uint8 `bson:"face"`
	HairStyle  uint8 `bson:"hairStyle"`
	HairColor  uint8 `bson:"hairColor"`
	FacialHair uint8 `bson:"facialHair"`
	OutfitID   uint8 `bson:"outfitId"`

	// Position
	Location     LocationEntity `bson:"location"`
	BindLocation LocationEntity `bson:"bindLocation"`

	// Stats
	Level uint8  `bson:"level"`
	XP    uint32 `bson:"xp"`
	Money uint32 `bson:"money"`

	Health    uint32   `bson:"health"`
	MaxHealth uint32   `bson:"maxHealth"`
	Power     []uint32 `bson:"power"`
	MaxPower  []uint32 `bson:"maxPower"`

	// Display
	DisplayID       uint32 `bson:"displayId"`
	NativeDisplayID uint32 `bson:"nativeDisplayId"`

	// Flags
	GuildID         uint32 `bson:"guildId"`
	CharFlags       uint32 `bson:"charFlags"`
	PlayerFlags     uint32 `bson:"playerFlags"`
	Recustomization uint32 `bson:"recustomization"`
	FirstLogin      uint8  `bson:"firstLogin"`

	// Pet
	Pet PetEntity `bson:"pet"`

	// Inventory (embedded array, only non-empty slots)
	Inventory []ItemEntity `bson:"inventory"`

	// Spells
	KnownSpells []uint32 `bson:"knownSpells"`

	// Action bar (120 buttons, packed)
	Actions []uint32 `bson:"actions"`

	// Quests are embedded in the character document (no separate collection):
	// one sub-document per active quest plus the list of rewarded quest ids.
	Quests         []QuestProgressEntity `bson:"quests,omitempty"`
	RewardedQuests []uint32              `bson:"rewardedQuests,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// CollectionName returns the MongoDB collection name.
func (CharacterEntity) CollectionName() string { return "characters" }

// LocationEntity stores a position in the world.
type LocationEntity struct {
	Map  uint32  `bson:"map"`
	Zone uint32  `bson:"zone"`
	X    float32 `bson:"x"`
	Y    float32 `bson:"y"`
	Z    float32 `bson:"z"`
	O    float32 `bson:"o"`
}

// PetEntity stores pet display info shown on the character select screen.
type PetEntity struct {
	DisplayID uint32 `bson:"displayId"`
	Level     uint32 `bson:"level"`
	Family    uint32 `bson:"family"`
}

// ItemEntity stores a single item in a character's inventory.
type ItemEntity struct {
	Slot       uint16 `bson:"slot"`
	Entry      uint32 `bson:"entry"`
	Count      uint32 `bson:"count"`
	Enchant    uint32 `bson:"enchant"`
	Flags      uint16 `bson:"flags"`
	Durability uint16 `bson:"durability"`
	PropertyID int8   `bson:"propertyId"`
}

// QuestProgressEntity stores a character's progress in a single quest. It is
// embedded in the character document, so the whole quest log is loaded and
// saved with the character in one document read/write.
type QuestProgressEntity struct {
	QuestID           uint32    `bson:"questId"`
	Status            uint32    `bson:"status"`
	Timer             uint32    `bson:"timer,omitempty"`
	ItemCount         [6]uint16 `bson:"itemCount,omitempty"`
	CreatureOrGOCount [4]uint16 `bson:"creatureOrGoCount,omitempty"`
	PlayerCount       uint16    `bson:"playerCount,omitempty"`
	Explored          bool      `bson:"explored,omitempty"`
}

// --- Counter (for GUID allocation) ---

// CounterEntity is used for atomic sequence generation.
type CounterEntity struct {
	ID  string `bson:"_id"`
	Seq int64  `bson:"seq"`
}

// CollectionName returns the MongoDB collection name.
func (CounterEntity) CollectionName() string { return "counters" }
