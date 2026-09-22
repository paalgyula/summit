package store

import (
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

// AccountRepo is the interface for account persistence.
type AccountRepo interface {
	// FindAccount retrives an account from the store or returns nil if does not exists.
	FindAccount(name string) *Account

	// CreateAccount creates an account. The account verifier and salt should be set.
	CreateAccount(account *Account) error
}

// CharacterRepo is the interface for character persistence.
type CharacterRepo interface {
	// GetCharacters retrives characters from the store for the specified account.
	GetCharacters(account string) (player.Players, error)

	// GetCharacter retrieves a single character by GUID.
	GetCharacter(guid uint32) (*player.Player, error)

	// CreateCharacter persists the character in the store.
	CreateCharacter(account string, character *player.Player) error

	// UpdateCharacter updates an existing character in the store.
	UpdateCharacter(character *player.Player) error

	// DeleteCharacter removes character from db.
	DeleteCharacter(characterID int) error
}

// WorldRepo is the interface for world data persistence (creatures, quests, etc.).
type WorldRepo interface {
	// Creature templates
	GetCreatureTemplate(entry uint32) (*CreatureTemplate, error)
	GetCreatureTemplates() (map[uint32]*CreatureTemplate, error)

	// Creature spawns
	GetCreatureSpawns(mapID uint32) ([]*CreatureSpawn, error)
	GetCreatureSpawn(spawnID uint64) (*CreatureSpawn, error)

	// Creature addons (per-spawn visual/movement overrides)
	GetCreatureAddon(creatureGUID uint32) (*CreatureAddon, error)
	GetAllCreatureAddons() (map[uint32]*CreatureAddon, error)

	// Waypoint paths
	GetWaypointPath(pathID uint32) (*WaypointPath, error)
	GetAllWaypointPaths() (map[uint32]*WaypointPath, error)

	// Item templates (item_template), keyed by entry
	GetItemTemplates() (map[uint32]*basedata.ItemTemplate, error)

	// Quest templates
	GetQuestTemplate(id uint32) (*QuestTemplate, error)
	GetQuestTemplates() (map[uint32]*QuestTemplate, error)

	// Quest relations
	GetCreatureQuestRelations(creatureEntry uint32) ([]uint32, error)
	GetCreatureQuestInvolvedRelations(creatureEntry uint32) ([]uint32, error)

	// Starting spells and action bar of a new character (playercreateinfo_spell / _action)
	GetPlayerCreateSpells(race, class uint8) ([]uint32, error)
	GetPlayerCreateActions(race, class uint8) ([]PlayerCreateAction, error)

	// Bulk quest relations (creature_entry → quest IDs)
	GetAllCreatureQuestRelations() (map[uint32][]uint32, error)
	GetAllCreatureQuestInvolvedRelations() (map[uint32][]uint32, error)
}

// PlayerCreateAction is one starting action-bar button of a race / class.
type PlayerCreateAction struct {
	Button uint8
	Action uint32
	Type   uint8
}

// CreatureTemplate is a simplified creature template for the world server.
type CreatureTemplate struct {
	Entry            uint32
	Name             string
	SubName          string
	MinLevel         uint8
	MaxLevel         uint8
	Faction          uint32
	NpcFlag          uint32
	UnitFlags        uint32
	DynamicFlags     uint32
	Type             uint32
	Family           uint32
	Rank             uint32
	HealthMultiplier float32
	DamageMultiplier float32
	ArmorMultiplier  float32
	MovementType     uint32

	// ModelIDs are the CreatureDisplayInfo ids a spawn picks from (empty when unknown).
	ModelIDs       []uint32
	Scale          float32
	SpeedWalk      float32
	SpeedRun       float32
	BaseAttackTime uint32
	UnitClass      uint32
	// FlagsExtra are creature_template.flags_extra (CREATURE_FLAG_EXTRA_*).
	FlagsExtra uint32
}

// CreatureSpawn represents a single creature spawn point.
type CreatureSpawn struct {
	SpawnID         uint64
	Entry           uint32
	MapID           uint32
	SpawnMask       uint8
	PhaseMask       uint32
	PosX            float32
	PosY            float32
	PosZ            float32
	Orientation     float32
	SpawnTimeSecs   uint32
	WanderDistance  float32
	MovementType    uint8
	CurrentWaypoint uint32
	CurrentHealth   uint32
	CurrentMana     uint32
	NpcFlag         uint32
	UnitFlags       uint32
	DynamicFlags    uint32
}

// QuestTemplate is a simplified quest template for the world server.
type QuestTemplate struct {
	ID                  uint32
	QuestLevel          int16
	MinLevel            uint8
	QuestType           uint8
	QuestSortID         int16
	RequiredClasses     uint32
	RequiredRaces       uint32
	Flags               uint32
	SpecialFlags        uint32
	TimeAllowed         uint32
	StartItem           uint32
	RequiredPlayerKills uint32
	RewardMoney         int32
	RewardXP            uint32

	RequiredItemId        [6]uint32
	RequiredItemCount     [6]uint16
	RequiredNpcOrGo       [4]int32
	RequiredNpcOrGoCount  [4]uint16
	RewardItemId          [4]uint32
	RewardItemCount       [4]uint16
	RewardChoiceItemId    [6]uint32
	RewardChoiceItemCount [6]uint16
	RewardFactionId       [5]uint32
	RewardFactionValue    [5]int32

	Title              string
	Description        string
	Objectives         string
	AreaDescription    string
	QuestCompletionLog string
	ObjectiveText      [4]string

	PrevQuestId          int32
	NextQuestId          int32
	ExclusiveGroup       int32
	BreadcrumbForQuestId uint32
	RequiredSkillId      uint16
	RequiredSkillPoints  uint16
}

// Waypoint is a single waypoint along a creature's path.
type Waypoint struct {
	Point            uint32
	X, Y, Z         float32
	Orientation      float32
	Delay            uint32    // wait time at this waypoint (ms)
	MoveType         uint8     // 0 = walk, 1 = run
	Action           int32     // scripted action ID
	ActionChance     int32     // chance for action (0-100)
	Velocity         float32   // custom speed for this segment (0 = use default)
	SmoothTransition bool      // enable catmull-rom spline interpolation
	SplinePoints     []Vector3 // extra intermediate spline points
}

// Vector3 represents a 3D point for spline interpolation.
type Vector3 struct {
	X, Y, Z float32
}

// WaypointPath represents an ordered list of waypoints for a creature path.
type WaypointPath struct {
	PathID uint32
	Points []Waypoint
}

// CreatureAddon holds per-spawn visual and movement overrides from creature_addon.
type CreatureAddon struct {
	Creature               uint32 // spawn GUID
	PathID                 uint32 // references WaypointPath.PathID
	Mount                  uint32
	Bytes1                 uint32
	Emote                  uint32
	VisibilityDistanceType uint32
}

// Movement type constants matching AzerothCore's MovementGeneratorType.
const (
	MotionTypeIdle     uint8 = 0
	MotionTypeRandom   uint8 = 1
	MotionTypeWaypoint uint8 = 2
)

//go:generate mockgen -destination=mock_store/mock_account_repo.go -package=mock_store . AccountRepo
//go:generate mockgen -destination=mock_store/mock_character_repo.go -package=mock_store . CharacterRepo
//go:generate mockgen -destination=mock_store/mock_world_repo.go -package=mock_store . WorldRepo
