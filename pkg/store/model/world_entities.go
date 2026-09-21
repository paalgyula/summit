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
	WanderDistance   float32 `bson:"wander_distance"`
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
	ID                 uint32   `bson:"id"`
	QuestLevel         int16    `bson:"questLevel"`
	MinLevel           uint8    `bson:"minLevel"`
	QuestType          uint8    `bson:"questType"`
	QuestSortID        int16    `bson:"questSortID"`
	RequiredClasses    uint32   `bson:"requiredClasses"`
	RequiredRaces      uint32   `bson:"requiredRaces"`
	Flags              uint32   `bson:"flags"`
	SpecialFlags       uint32   `bson:"specialFlags"`
	TimeAllowed        uint32   `bson:"timeAllowed"`
	StartItem          uint32   `bson:"startItem"`
	RequiredPlayerKills uint32  `bson:"requiredPlayerKills"`
	RewardMoney        int32    `bson:"rewardMoney"`
	RewardXP           uint32   `bson:"rewardXP"`

	RequiredItemId        [6]uint32  `bson:"requiredItemId"`
	RequiredItemCount     [6]uint16  `bson:"requiredItemCount"`
	RequiredNpcOrGo       [4]int32   `bson:"requiredNpcOrGo"`
	RequiredNpcOrGoCount  [4]uint16  `bson:"requiredNpcOrGoCount"`
	RewardItemId          [4]uint32  `bson:"rewardItemId"`
	RewardItemCount       [4]uint16  `bson:"rewardItemCount"`
	RewardChoiceItemId    [6]uint32  `bson:"rewardChoiceItemId"`
	RewardChoiceItemCount [6]uint16  `bson:"rewardChoiceItemCount"`
	RewardFactionId       [5]uint32  `bson:"rewardFactionId"`
	RewardFactionValue    [5]int32   `bson:"rewardFactionValue"`

	Title              string   `bson:"title"`
	Description        string   `bson:"description"`
	Objectives         string   `bson:"objectives"`
	AreaDescription    string   `bson:"areaDescription"`
	QuestCompletionLog string   `bson:"questCompletionLog"`
	ObjectiveText      [4]string `bson:"objectiveText"`

	PrevQuestId          int32    `bson:"prevQuestId"`
	NextQuestId          uint32   `bson:"nextQuestId"`
	ExclusiveGroup       int32    `bson:"exclusiveGroup"`
	BreadcrumbForQuestId uint32   `bson:"breadcrumbForQuestId"`
	RequiredSkillId      uint16   `bson:"requiredSkillId"`
	RequiredSkillPoints  uint16   `bson:"requiredSkillPoints"`
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
