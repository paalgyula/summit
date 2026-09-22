package model

// CreatureTemplateEntity is the MongoDB document for the creatureTemplate collection.
type CreatureTemplateEntity struct {
	Entry            uint32  `bson:"entry"`
	Name             string  `bson:"name"`
	SubName          string  `bson:"subName,omitempty"`
	MinLevel         uint8   `bson:"minLevel"`
	MaxLevel         uint8   `bson:"maxLevel"`
	Faction          uint32  `bson:"faction"`
	NpcFlag          uint32  `bson:"npcFlag"`
	UnitFlags        uint32  `bson:"unitFlags"`
	DynamicFlags     uint32  `bson:"dynamicFlags"`
	Type             uint32  `bson:"type"`
	Family           uint32  `bson:"family"`
	Rank             uint32  `bson:"rank"`
	HealthMultiplier float32 `bson:"healthMultiplier"`
	DamageMultiplier float32 `bson:"damageMultiplier"`
	ArmorMultiplier  float32 `bson:"armorMultiplier"`
	MovementType     uint32  `bson:"movementType"`

	// Display ids (creature_template.modelid1-4, zero slots dropped), scale and speeds
	ModelIDs       []uint32 `bson:"modelIds"`
	Scale          float32  `bson:"scale"`
	SpeedWalk      float32  `bson:"speedWalk"`
	SpeedRun       float32  `bson:"speedRun"`
	BaseAttackTime uint32   `bson:"baseAttackTime"`
	UnitClass      uint32   `bson:"unitClass"`
	FlagsExtra     uint32   `bson:"flagsExtra"`
}

func (CreatureTemplateEntity) CollectionName() string { return "creatureTemplate" }

// CreatureSpawnEntity is the MongoDB document for the creature collection.
type CreatureSpawnEntity struct {
	SpawnID         uint64  `bson:"guid"`
	Entry           uint32  `bson:"entry"`
	MapID           uint32  `bson:"map"`
	SpawnMask       uint8   `bson:"spawnMask"`
	PhaseMask       uint32  `bson:"phaseMask"`
	PosX            float32 `bson:"positionX"`
	PosY            float32 `bson:"positionY"`
	PosZ            float32 `bson:"positionZ"`
	Orientation     float32 `bson:"orientation"`
	SpawnTimeSecs   uint32  `bson:"spawnTime"`
	WanderDistance  float32 `bson:"wander_distance"`
	MovementType    uint8   `bson:"movementType"`
	CurrentWaypoint uint32  `bson:"currentwaypoint"`
	CurrentHealth   uint32  `bson:"curhealth"`
	CurrentMana     uint32  `bson:"curmana"`
	NpcFlag         uint32  `bson:"npcflag"`
	UnitFlags       uint32  `bson:"unit_flags"`
	DynamicFlags    uint32  `bson:"dynamicflags"`
}

func (CreatureSpawnEntity) CollectionName() string { return "creature" }

// QuestTemplateEntity is the MongoDB document for the questTemplate collection.
type QuestTemplateEntity struct {
	ID                  uint32 `bson:"id"`
	QuestLevel          int16  `bson:"questLevel"`
	MinLevel            uint8  `bson:"minLevel"`
	QuestType           uint8  `bson:"questType"`
	QuestSortID         int16  `bson:"questSortID"`
	RequiredClasses     uint32 `bson:"requiredClasses"`
	RequiredRaces       uint32 `bson:"requiredRaces"`
	Flags               uint32 `bson:"flags"`
	SpecialFlags        uint32 `bson:"specialFlags"`
	TimeAllowed         uint32 `bson:"timeAllowed"`
	StartItem           uint32 `bson:"startItem"`
	RequiredPlayerKills uint32 `bson:"requiredPlayerKills"`
	RewardMoney         int32  `bson:"rewardMoney"`
	RewardXP            uint32 `bson:"rewardXP"`

	RequiredItemId        [6]uint32 `bson:"requiredItemId"`
	RequiredItemCount     [6]uint16 `bson:"requiredItemCount"`
	RequiredNpcOrGo       [4]int32  `bson:"requiredNpcOrGo"`
	RequiredNpcOrGoCount  [4]uint16 `bson:"requiredNpcOrGoCount"`
	RewardItemId          [4]uint32 `bson:"rewardItemId"`
	RewardItemCount       [4]uint16 `bson:"rewardItemCount"`
	RewardChoiceItemId    [6]uint32 `bson:"rewardChoiceItemId"`
	RewardChoiceItemCount [6]uint16 `bson:"rewardChoiceItemCount"`
	RewardFactionId       [5]uint32 `bson:"rewardFactionId"`
	RewardFactionValue    [5]int32  `bson:"rewardFactionValue"`

	Title              string    `bson:"title"`
	Description        string    `bson:"description"`
	Objectives         string    `bson:"objectives"`
	AreaDescription    string    `bson:"areaDescription"`
	QuestCompletionLog string    `bson:"questCompletionLog"`
	ObjectiveText      [4]string `bson:"objectiveText"`

	PrevQuestId          int32  `bson:"prevQuestId"`
	NextQuestId          int32  `bson:"nextQuestId"`
	ExclusiveGroup       int32  `bson:"exclusiveGroup"`
	BreadcrumbForQuestId uint32 `bson:"breadcrumbForQuestId"`
	RequiredSkillId      uint16 `bson:"requiredSkillId"`
	RequiredSkillPoints  uint16 `bson:"requiredSkillPoints"`
}

func (QuestTemplateEntity) CollectionName() string { return "questTemplate" }

// CreatureQuestRelationEntity maps creature entries to quest IDs they offer.
type CreatureQuestRelationEntity struct {
	CreatureEntry uint32 `bson:"creatureEntry"`
	QuestID       uint32 `bson:"questId"`
}

func (CreatureQuestRelationEntity) CollectionName() string { return "creatureQuestRelation" }

// CreatureQuestInvolvedRelationEntity maps creature entries to quest IDs they complete.
type CreatureQuestInvolvedRelationEntity struct {
	CreatureEntry uint32 `bson:"creatureEntry"`
	QuestID       uint32 `bson:"questId"`
}

func (CreatureQuestInvolvedRelationEntity) CollectionName() string { return "creatureInvolvedRelation" }

// --- GameObject ---

// GameObjectTemplateEntity is the MongoDB document for the gameobjectTemplate collection.
type GameObjectTemplateEntity struct {
	Entry          uint32    `bson:"entry"`
	Type           uint8     `bson:"type"`
	DisplayID      uint32    `bson:"displayId"`
	Name           string    `bson:"name"`
	IconName       string    `bson:"iconName,omitempty"`
	Size           float32   `bson:"size"`
	Data           [24]int32 `bson:"data"`
	AIName         string    `bson:"aiName,omitempty"`
	ScriptName     string    `bson:"scriptName,omitempty"`
	Faction        uint32    `bson:"faction,omitempty"`
	Flags          uint32    `bson:"flags,omitempty"`
	MinGold        uint32    `bson:"mingold,omitempty"`
	MaxGold        uint32    `bson:"maxgold,omitempty"`
	ArtKit         uint32    `bson:"artkit,omitempty"`
	CastBarCaption string    `bson:"castBarCaption,omitempty"`
	Unk1           string    `bson:"unk1,omitempty"`
}

func (GameObjectTemplateEntity) CollectionName() string { return "gameobjectTemplate" }

// GameObjectSpawnEntity is the MongoDB document for the gameobject collection.
type GameObjectSpawnEntity struct {
	GUID          uint32     `bson:"guid"`
	Entry         uint32     `bson:"entry"`
	MapID         uint16     `bson:"map"`
	ZoneID        uint16     `bson:"zoneId,omitempty"`
	AreaID        uint16     `bson:"areaId,omitempty"`
	SpawnMask     uint8      `bson:"spawnMask"`
	PhaseMask     uint32     `bson:"phaseMask"`
	PosX          float32    `bson:"positionX"`
	PosY          float32    `bson:"positionY"`
	PosZ          float32    `bson:"positionZ"`
	Orientation   float32    `bson:"orientation"`
	Rotation      [4]float32 `bson:"rotation"`
	SpawnTimeSecs int32      `bson:"spawnTimeSecs"`
	AnimProgress  uint8      `bson:"animProgress"`
	State         uint8      `bson:"state"`
	ScriptName    string     `bson:"scriptName,omitempty"`
}

func (GameObjectSpawnEntity) CollectionName() string { return "gameobject" }

// GameObjectLootTemplateEntity is the MongoDB document for the gameobjectLootTemplate collection.
type GameObjectLootTemplateEntity struct {
	Entry         uint32  `bson:"entry"`
	Item          uint32  `bson:"item"`
	Reference     int32   `bson:"reference"`
	Chance        float32 `bson:"challenge"`
	QuestRequired int8    `bson:"questRequired"`
	LootMode      uint16  `bson:"lootMode"`
	GroupID       uint8   `bson:"groupId"`
	MinCount      uint8   `bson:"minCount"`
	MaxCount      uint8   `bson:"maxCount"`
}

func (GameObjectLootTemplateEntity) CollectionName() string { return "gameobjectLootTemplate" }

// WaypointDataEntity is the MongoDB document for the waypoint_data collection.
// Each document represents all waypoints for a single path, grouped by path_id.
type WaypointDataEntity struct {
	PathID uint32           `bson:"pathId"`
	Points []WaypointEntity `bson:"points"`
}

// WaypointEntity is a single waypoint within a path.
type WaypointEntity struct {
	Point            uint32              `bson:"point"`
	PositionX        float32             `bson:"positionX"`
	PositionY        float32             `bson:"positionY"`
	PositionZ        float32             `bson:"positionZ"`
	Orientation      float32             `bson:"orientation"`
	Delay            uint32              `bson:"delay"`
	MoveType         uint8               `bson:"moveType"`
	Action           int32               `bson:"action"`
	ActionChance     int32               `bson:"actionChance"`
	Velocity         float32             `bson:"velocity"`
	SmoothTransition bool                `bson:"smoothTransition"`
	SplinePoints     []SplinePointEntity `bson:"splinePoints,omitempty"`
}

// SplinePointEntity is an intermediate spline point between waypoints.
type SplinePointEntity struct {
	PositionX float32 `bson:"positionX"`
	PositionY float32 `bson:"positionY"`
	PositionZ float32 `bson:"positionZ"`
}

func (WaypointDataEntity) CollectionName() string { return "waypoint_data" }

// CreatureAddonEntity is the MongoDB document for the creature_addon collection.
// Maps creature GUIDs to their path_id (referencing waypoint_data) and visual overrides.
type CreatureAddonEntity struct {
	Creature               uint32 `bson:"creature"`
	PathID                 uint32 `bson:"pathId"`
	Mount                  uint32 `bson:"mount"`
	Bytes1                 uint32 `bson:"bytes1"`
	Emote                  uint32 `bson:"emote"`
	VisibilityDistanceType uint32 `bson:"visibilityDistanceType"`
}

func (CreatureAddonEntity) CollectionName() string { return "creature_addon" }

// ItemTemplateEntity is the MongoDB document for the item_template collection
// (written by `datagen migrate`).
type ItemTemplateEntity struct {
	Entry              uint32             `bson:"entry"`
	Class              uint32             `bson:"class"`
	SubClass           uint32             `bson:"subclass"`
	Name               string             `bson:"name"`
	DisplayID          uint32             `bson:"displayId"`
	Quality            uint32             `bson:"quality"`
	InventoryType      uint32             `bson:"inventoryType"`
	AllowableClass     int32              `bson:"allowableClass"`
	AllowableRace      int32              `bson:"allowableRace"`
	ItemLevel          uint32             `bson:"itemLevel"`
	RequiredLevel      uint32             `bson:"requiredLevel"`
	RequiredSkill      uint32             `bson:"requiredSkill"`
	RequiredSkillRank  uint32             `bson:"requiredSkillRank"`
	BuyCount           uint32             `bson:"buyCount"`
	BuyPrice           int64              `bson:"buyPrice"`
	SellPrice          uint32             `bson:"sellPrice"`
	MaxCount           int32              `bson:"maxCount"`
	Stackable          int32              `bson:"stackable"`
	StatsCount         uint32             `bson:"statsCount"`
	Stats              []ItemStatEntity   `bson:"stats"`
	Damage             []ItemDamageEntity `bson:"damage"`
	Armor              uint32             `bson:"armor"`
	HolyRes            int32              `bson:"holyRes"`
	FireRes            int32              `bson:"fireRes"`
	NatureRes          int32              `bson:"natureRes"`
	FrostRes           int32              `bson:"frostRes"`
	ShadowRes          int32              `bson:"shadowRes"`
	ArcaneRes          int32              `bson:"arcaneRes"`
	Delay              uint32             `bson:"delay"`
	AmmoType           uint32             `bson:"ammoType"`
	RangedModRange     float32            `bson:"rangedModRange"`
	Spells             []ItemSpellEntity  `bson:"spells"`
	Sockets            []ItemSocketEntity `bson:"sockets"`
	SocketBonus        uint32             `bson:"socketBonus"`
	Bonding            uint32             `bson:"bonding"`
	LockID             uint32             `bson:"lockId"`
	Material           int32              `bson:"material"`
	Sheath             uint32             `bson:"sheath"`
	ItemSet            uint32             `bson:"itemSet"`
	MaxDurability      uint32             `bson:"maxDurability"`
	RandomProperty     int32              `bson:"randomProperty"`
	RandomSuffix       int32              `bson:"randomSuffix"`
	Block              int32              `bson:"block"`
	ContainerSlots     uint32             `bson:"containerSlots"`
	BagFamily          uint32             `bson:"bagFamily"`
	Flags              uint32             `bson:"flags"`
	Duration           uint32             `bson:"duration"`
	DisenchantID       uint32             `bson:"disenchantId"`
	FoodType           uint32             `bson:"foodType"`
	MinMoneyLoot       uint32             `bson:"minMoneyLoot"`
	MaxMoneyLoot       uint32             `bson:"maxMoneyLoot"`
	Description        string             `bson:"description"`
	PageText           uint32             `bson:"pageText"`
	LanguageID         uint32             `bson:"languageId"`
	PageMaterial       uint32             `bson:"pageMaterial"`
	StartQuest         uint32             `bson:"startQuest"`
	RequiredHonorRank  uint32             `bson:"requiredHonorRank"`
	RequiredCityRank   uint32             `bson:"requiredCityRank"`
	RequiredRepFaction uint32             `bson:"requiredRepFaction"`
	RequiredRepRank    uint32             `bson:"requiredRepRank"`
}

// ItemStatEntity is one stat modifier of an item.
type ItemStatEntity struct {
	Type  uint32 `bson:"type"`
	Value int32  `bson:"value"`
}

// ItemDamageEntity is one damage range of an item.
type ItemDamageEntity struct {
	Type uint32  `bson:"type"`
	Min  float32 `bson:"min"`
	Max  float32 `bson:"max"`
}

// ItemSpellEntity is one spell triggered by an item.
type ItemSpellEntity struct {
	SpellID          int32   `bson:"spellId"`
	Trigger          uint32  `bson:"trigger"`
	Charges          int32   `bson:"charges"`
	PPMRate          float32 `bson:"ppmRate"`
	Cooldown         int32   `bson:"cooldown"`
	Category         uint32  `bson:"category"`
	CategoryCooldown int32   `bson:"categoryCooldown"`
}

// ItemSocketEntity is one gem socket of an item.
type ItemSocketEntity struct {
	Color   uint32 `bson:"color"`
	Content uint32 `bson:"content"`
}

// CollectionName returns the MongoDB collection name.
func (ItemTemplateEntity) CollectionName() string { return "item_template" }
