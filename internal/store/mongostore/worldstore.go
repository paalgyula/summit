package mongostore

import (
	"context"
	"fmt"
	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/wow"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/store/model"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WorldStore implements store.WorldRepo backed by MongoDB.
type WorldStore struct {
	db                  *mongo.Database
	creatureTemplates   *mongo.Collection
	creatureSpawns      *mongo.Collection
	creatureAddons      *mongo.Collection
	waypointData        *mongo.Collection
	questTemplates      *mongo.Collection
	creatureQuestRel    *mongo.Collection
	creatureQuestInvRel *mongo.Collection
	itemTemplates       *mongo.Collection
}

// NewWorldStore creates a new WorldStore using the given database.
func NewWorldStore(db *mongo.Database) *WorldStore {
	return &WorldStore{
		db:                  db,
		creatureTemplates:   db.Collection("creature_template"),
		creatureSpawns:      db.Collection("creature"),
		creatureAddons:      db.Collection("creature_addon"),
		waypointData:        db.Collection("waypoint_data"),
		questTemplates:      db.Collection("quest_template"),
		creatureQuestRel:    db.Collection("creature_queststarter"),
		creatureQuestInvRel: db.Collection("creature_questender"),
		itemTemplates:       db.Collection("item_template"),
	}
}

// GetItemTemplates retrieves every item template, keyed by entry.
func (w *WorldStore) GetItemTemplates() (map[uint32]*basedata.ItemTemplate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cursor, err := w.itemTemplates.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetItemTemplates: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	result := make(map[uint32]*basedata.ItemTemplate)

	for cursor.Next(ctx) {
		var e model.ItemTemplateEntity
		if err := cursor.Decode(&e); err != nil {
			return nil, fmt.Errorf("GetItemTemplates decode: %w", err)
		}

		result[e.Entry] = itemEntityToTemplate(e)
	}

	return result, cursor.Err()
}

// itemEntityToTemplate converts an item_template document to the server template.
func itemEntityToTemplate(e model.ItemTemplateEntity) *basedata.ItemTemplate {
	t := &basedata.ItemTemplate{
		Entry:              e.Entry,
		Class:              e.Class,
		SubClass:           e.SubClass,
		DisplayID:          e.DisplayID,
		Name:               e.Name,
		Quality:            e.Quality,
		InventoryType:      wow.InventoryType(e.InventoryType),
		AllowableClass:     e.AllowableClass,
		AllowableRace:      e.AllowableRace,
		RequiredLevel:      e.RequiredLevel,
		RequiredSkill:      e.RequiredSkill,
		RequiredSkillRank:  e.RequiredSkillRank,
		RequiredHonorRank:  e.RequiredHonorRank,
		RequiredCityRank:   e.RequiredCityRank,
		RequiredRepFaction: e.RequiredRepFaction,
		RequiredRepRank:    e.RequiredRepRank,
		ItemLevel:          e.ItemLevel,
		StatsCount:         e.StatsCount,
		Armor:              e.Armor,
		HolyRes:            e.HolyRes,
		FireRes:            e.FireRes,
		NatureRes:          e.NatureRes,
		FrostRes:           e.FrostRes,
		ShadowRes:          e.ShadowRes,
		ArcaneRes:          e.ArcaneRes,
		Delay:              e.Delay,
		AmmoType:           e.AmmoType,
		RangedModRange:     e.RangedModRange,
		SocketBonus:        e.SocketBonus,
		BuyPrice:           int32(e.BuyPrice),
		SellPrice:          e.SellPrice,
		BuyCount:           e.BuyCount,
		MaxDurability:      e.MaxDurability,
		MaxCount:           e.MaxCount,
		Stackable:          e.Stackable,
		ContainerSlots:     e.ContainerSlots,
		Flags:              e.Flags,
		Bonding:            e.Bonding,
		SheatheType:        e.Sheath,
		Material:           e.Material,
		BagFamily:          e.BagFamily,
		Block:              uint32(e.Block),
		ItemSet:            e.ItemSet,
		LockID:             e.LockID,
		RandomProperty:     e.RandomProperty,
		RandomSuffix:       e.RandomSuffix,
		PageText:           e.PageText,
		LanguageID:         e.LanguageID,
		PageMaterial:       e.PageMaterial,
		StartQuest:         e.StartQuest,
		DisenchantID:       e.DisenchantID,
		FoodType:           e.FoodType,
		Duration:           e.Duration,
		MinMoneyLoot:       e.MinMoneyLoot,
		MaxMoneyLoot:       e.MaxMoneyLoot,
	}

	for i, st := range e.Stats {
		if i < len(t.Stats) {
			t.Stats[i] = basedata.ItemStat{Type: st.Type, Value: st.Value}
		}
	}

	for i, d := range e.Damage {
		if i < len(t.Damage) {
			t.Damage[i] = basedata.ItemDamage{Type: d.Type, Min: d.Min, Max: d.Max}
		}
	}

	for i, sp := range e.Spells {
		if i < len(t.Spells) {
			t.Spells[i] = basedata.ItemSpell{
				SpellID: sp.SpellID, Trigger: sp.Trigger, Charges: sp.Charges, PPMRate: sp.PPMRate,
				Cooldown: sp.Cooldown, Category: sp.Category, CategoryCooldown: sp.CategoryCooldown,
			}
		}
	}

	for i, so := range e.Sockets {
		if i < len(t.Sockets) {
			t.Sockets[i] = basedata.ItemSocket{Color: so.Color, Content: so.Content}
		}
	}

	return t
}

// GetCreatureTemplate retrieves a creature template by entry.
func (w *WorldStore) GetCreatureTemplate(entry uint32) (*store.CreatureTemplate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var entity model.CreatureTemplateEntity

	err := w.creatureTemplates.FindOne(ctx, bson.M{"entry": entry}).Decode(&entity)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}

		return nil, fmt.Errorf("GetCreatureTemplate: %w", err)
	}

	return creatureEntityToTemplate(entity), nil
}

// GetCreatureTemplates retrieves all creature templates.
func (w *WorldStore) GetCreatureTemplates() (map[uint32]*store.CreatureTemplate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := w.creatureTemplates.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetCreatureTemplates: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.CreatureTemplateEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetCreatureTemplates decode: %w", err)
	}

	result := make(map[uint32]*store.CreatureTemplate, len(entities))

	for _, e := range entities {
		result[e.Entry] = creatureEntityToTemplate(e)
	}

	return result, nil
}

// GetCreatureSpawns retrieves all creature spawns for a map.
func (w *WorldStore) GetCreatureSpawns(mapID uint32) ([]*store.CreatureSpawn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	filter := bson.M{}
	if mapID > 0 {
		filter["map"] = mapID
	}

	opts := options.Find().SetBatchSize(1000)

	cursor, err := w.creatureSpawns.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("GetCreatureSpawns: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.CreatureSpawnEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetCreatureSpawns decode: %w", err)
	}

	result := make([]*store.CreatureSpawn, 0, len(entities))

	for _, e := range entities {
		result = append(result, creatureSpawnEntityToSpawn(e))
	}

	return result, nil
}

// GetCreatureSpawn retrieves a single creature spawn by spawn ID.
func (w *WorldStore) GetCreatureSpawn(spawnID uint64) (*store.CreatureSpawn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var entity model.CreatureSpawnEntity

	err := w.creatureSpawns.FindOne(ctx, bson.M{"guid": spawnID}).Decode(&entity)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}

		return nil, fmt.Errorf("GetCreatureSpawn: %w", err)
	}

	return creatureSpawnEntityToSpawn(entity), nil
}

// GetQuestTemplate retrieves a quest template by ID.
func (w *WorldStore) GetQuestTemplate(id uint32) (*store.QuestTemplate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var entity model.QuestTemplateEntity

	err := w.questTemplates.FindOne(ctx, bson.M{"id": id}).Decode(&entity)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}

		return nil, fmt.Errorf("GetQuestTemplate: %w", err)
	}

	return questEntityToTemplate(entity), nil
}

// GetQuestTemplates retrieves all quest templates.
func (w *WorldStore) GetQuestTemplates() (map[uint32]*store.QuestTemplate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := w.questTemplates.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetQuestTemplates: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.QuestTemplateEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetQuestTemplates decode: %w", err)
	}

	result := make(map[uint32]*store.QuestTemplate, len(entities))

	for _, e := range entities {
		result[e.ID] = questEntityToTemplate(e)
	}

	return result, nil
}

// GetCreatureQuestRelations retrieves all quest IDs offered by a creature.
func (w *WorldStore) GetCreatureQuestRelations(creatureEntry uint32) ([]uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := w.creatureQuestRel.Find(ctx, bson.M{"creatureEntry": creatureEntry})
	if err != nil {
		return nil, fmt.Errorf("GetCreatureQuestRelations: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.CreatureQuestRelationEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetCreatureQuestRelations decode: %w", err)
	}

	ids := make([]uint32, len(entities))

	for i, e := range entities {
		ids[i] = e.QuestID
	}

	return ids, nil
}

// GetCreatureQuestInvolvedRelations retrieves all quest IDs completed by a creature.
func (w *WorldStore) GetCreatureQuestInvolvedRelations(creatureEntry uint32) ([]uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := w.creatureQuestInvRel.Find(ctx, bson.M{"creatureEntry": creatureEntry})
	if err != nil {
		return nil, fmt.Errorf("GetCreatureQuestInvolvedRelations: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.CreatureQuestInvolvedRelationEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetCreatureQuestInvolvedRelations decode: %w", err)
	}

	ids := make([]uint32, len(entities))

	for i, e := range entities {
		ids[i] = e.QuestID
	}

	return ids, nil
}

// GetAllCreatureQuestRelations loads all creature quest starter relations at once.
func (w *WorldStore) GetAllCreatureQuestRelations() (map[uint32][]uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := w.creatureQuestRel.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetAllCreatureQuestRelations: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.CreatureQuestRelationEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetAllCreatureQuestRelations decode: %w", err)
	}

	result := make(map[uint32][]uint32, len(entities))

	for _, e := range entities {
		result[e.CreatureEntry] = append(result[e.CreatureEntry], e.QuestID)
	}

	return result, nil
}

// GetAllCreatureQuestInvolvedRelations loads all creature quest ender relations at once.
func (w *WorldStore) GetAllCreatureQuestInvolvedRelations() (map[uint32][]uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := w.creatureQuestInvRel.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetAllCreatureQuestInvolvedRelations: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.CreatureQuestInvolvedRelationEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetAllCreatureQuestInvolvedRelations decode: %w", err)
	}

	result := make(map[uint32][]uint32, len(entities))

	for _, e := range entities {
		result[e.CreatureEntry] = append(result[e.CreatureEntry], e.QuestID)
	}

	return result, nil
}

// --- Entity conversion helpers ---

func creatureEntityToTemplate(e model.CreatureTemplateEntity) *store.CreatureTemplate {
	return &store.CreatureTemplate{
		Entry:            e.Entry,
		Name:             e.Name,
		SubName:          e.SubName,
		MinLevel:         e.MinLevel,
		MaxLevel:         e.MaxLevel,
		Faction:          e.Faction,
		NpcFlag:          e.NpcFlag,
		UnitFlags:        e.UnitFlags,
		DynamicFlags:     e.DynamicFlags,
		Type:             e.Type,
		Family:           e.Family,
		Rank:             e.Rank,
		HealthMultiplier: e.HealthMultiplier,
		DamageMultiplier: e.DamageMultiplier,
		ArmorMultiplier:  e.ArmorMultiplier,
		MovementType:     e.MovementType,
		ModelIDs:         e.ModelIDs,
		Scale:            e.Scale,
		SpeedWalk:        e.SpeedWalk,
		SpeedRun:         e.SpeedRun,
		BaseAttackTime:   e.BaseAttackTime,
		UnitClass:        e.UnitClass,
		FlagsExtra:       e.FlagsExtra,
	}
}

func creatureSpawnEntityToSpawn(e model.CreatureSpawnEntity) *store.CreatureSpawn {
	return &store.CreatureSpawn{
		SpawnID:         e.SpawnID,
		Entry:           e.Entry,
		MapID:           e.MapID,
		SpawnMask:       e.SpawnMask,
		PhaseMask:       e.PhaseMask,
		PosX:            e.PosX,
		PosY:            e.PosY,
		PosZ:            e.PosZ,
		Orientation:     e.Orientation,
		SpawnTimeSecs:   e.SpawnTimeSecs,
		WanderDistance:  e.WanderDistance,
		MovementType:    e.MovementType,
		CurrentWaypoint: e.CurrentWaypoint,
		CurrentHealth:   e.CurrentHealth,
		CurrentMana:     e.CurrentMana,
		NpcFlag:         e.NpcFlag,
		UnitFlags:       e.UnitFlags,
		DynamicFlags:    e.DynamicFlags,
	}
}

func questEntityToTemplate(e model.QuestTemplateEntity) *store.QuestTemplate {
	return &store.QuestTemplate{
		ID:                  e.ID,
		QuestLevel:          e.QuestLevel,
		MinLevel:            e.MinLevel,
		QuestType:           e.QuestType,
		QuestSortID:         e.QuestSortID,
		RequiredClasses:     e.RequiredClasses,
		RequiredRaces:       e.RequiredRaces,
		Flags:               e.Flags,
		SpecialFlags:        e.SpecialFlags,
		TimeAllowed:         e.TimeAllowed,
		StartItem:           e.StartItem,
		RequiredPlayerKills: e.RequiredPlayerKills,
		RewardMoney:         e.RewardMoney,
		RewardXP:            e.RewardXP,
		RewardXPDifficulty:  e.RewardXPDifficulty,

		RequiredItemId:        e.RequiredItemId,
		RequiredItemCount:     e.RequiredItemCount,
		RequiredNpcOrGo:       e.RequiredNpcOrGo,
		RequiredNpcOrGoCount:  e.RequiredNpcOrGoCount,
		RewardItemId:          e.RewardItemId,
		RewardItemCount:       e.RewardItemCount,
		RewardChoiceItemId:    e.RewardChoiceItemId,
		RewardChoiceItemCount: e.RewardChoiceItemCount,
		RewardFactionId:       e.RewardFactionId,
		RewardFactionValue:    e.RewardFactionValue,

		Title:              e.Title,
		Description:        e.Description,
		Objectives:         e.Objectives,
		AreaDescription:    e.AreaDescription,
		QuestCompletionLog: e.QuestCompletionLog,
		ObjectiveText:      e.ObjectiveText,

		PrevQuestId:          e.PrevQuestId,
		NextQuestId:          e.NextQuestId,
		ExclusiveGroup:       e.ExclusiveGroup,
		BreadcrumbForQuestId: e.BreadcrumbForQuestId,
		RequiredSkillId:      e.RequiredSkillId,
		RequiredSkillPoints:  e.RequiredSkillPoints,
	}
}

// GetCreatureAddon retrieves the creature_addon entry for a given spawn GUID.
func (w *WorldStore) GetCreatureAddon(creatureGUID uint32) (*store.CreatureAddon, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var e model.CreatureAddonEntity
	err := w.creatureAddons.FindOne(ctx, bson.M{"creature": creatureGUID}).Decode(&e)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		return nil, fmt.Errorf("GetCreatureAddon: %w", err)
	}

	return &store.CreatureAddon{
		Creature:               e.Creature,
		PathID:                 e.PathID,
		Mount:                  e.Mount,
		Bytes1:                 e.Bytes1,
		Emote:                  e.Emote,
		VisibilityDistanceType: e.VisibilityDistanceType,
	}, nil
}

// GetAllCreatureAddons retrieves all creature_addon entries, keyed by creature GUID.
func (w *WorldStore) GetAllCreatureAddons() (map[uint32]*store.CreatureAddon, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := w.creatureAddons.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetAllCreatureAddons: %w", err)
	}
	defer cursor.Close(ctx) //nolint:errcheck

	result := make(map[uint32]*store.CreatureAddon)
	for cursor.Next(ctx) {
		var e model.CreatureAddonEntity
		if err := cursor.Decode(&e); err != nil {
			return nil, fmt.Errorf("GetAllCreatureAddons decode: %w", err)
		}
		result[e.Creature] = &store.CreatureAddon{
			Creature:               e.Creature,
			PathID:                 e.PathID,
			Mount:                  e.Mount,
			Bytes1:                 e.Bytes1,
			Emote:                  e.Emote,
			VisibilityDistanceType: e.VisibilityDistanceType,
		}
	}
	return result, cursor.Err()
}

// GetWaypointPath retrieves a single waypoint path by its ID.
func (w *WorldStore) GetWaypointPath(pathID uint32) (*store.WaypointPath, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var e model.WaypointDataEntity
	err := w.waypointData.FindOne(ctx, bson.M{"_id": pathID}).Decode(&e)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		return nil, fmt.Errorf("GetWaypointPath: %w", err)
	}

	return waypointEntityToPath(e), nil
}

// GetAllWaypointPaths retrieves all waypoint paths, keyed by path_id.
func (w *WorldStore) GetAllWaypointPaths() (map[uint32]*store.WaypointPath, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := w.waypointData.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetAllWaypointPaths: %w", err)
	}
	defer cursor.Close(ctx) //nolint:errcheck

	result := make(map[uint32]*store.WaypointPath)
	for cursor.Next(ctx) {
		var e model.WaypointDataEntity
		if err := cursor.Decode(&e); err != nil {
			return nil, fmt.Errorf("GetAllWaypointPaths decode: %w", err)
		}
		path := waypointEntityToPath(e)
		result[path.PathID] = path
	}
	return result, cursor.Err()
}

func waypointEntityToPath(e model.WaypointDataEntity) *store.WaypointPath {
	points := make([]store.Waypoint, len(e.Points))
	for i, p := range e.Points {
		wp := store.Waypoint{
			Point:            p.Point,
			X:                p.PositionX,
			Y:                p.PositionY,
			Z:                p.PositionZ,
			Orientation:      p.Orientation,
			Delay:            p.Delay,
			MoveType:         p.MoveType,
			Action:           p.Action,
			ActionChance:     p.ActionChance,
			Velocity:         p.Velocity,
			SmoothTransition: p.SmoothTransition,
		}

		// Convert spline points
		if len(p.SplinePoints) > 0 {
			wp.SplinePoints = make([]store.Vector3, len(p.SplinePoints))
			for j, sp := range p.SplinePoints {
				wp.SplinePoints[j] = store.Vector3{
					X: sp.PositionX,
					Y: sp.PositionY,
					Z: sp.PositionZ,
				}
			}
		}

		points[i] = wp
	}
	return &store.WaypointPath{PathID: e.PathID, Points: points}
}

// Compile-time interface check.
var _ store.WorldRepo = (*WorldStore)(nil)

func init() {
	// Ensure the log import is used.
	_ = log.Logger
}

// GetPlayerCreateSpells returns the spells every new character of a race /
// class knows: rows whose race and class masks include them (0 = any).
func (w *WorldStore) GetPlayerCreateSpells(race, class uint8) ([]uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := w.db.Collection("playercreateinfo_spell").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("GetPlayerCreateSpells: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	raceBit := uint32(1) << (race - 1)
	classBit := uint32(1) << (class - 1)

	var spells []uint32

	for cursor.Next(ctx) {
		var row struct {
			RaceMask  uint32 `bson:"raceMask"`
			ClassMask uint32 `bson:"classMask"`
			Spell     uint32 `bson:"spell"`
		}

		if err := cursor.Decode(&row); err != nil {
			return nil, fmt.Errorf("GetPlayerCreateSpells decode: %w", err)
		}

		if (row.RaceMask == 0 || row.RaceMask&raceBit != 0) && (row.ClassMask == 0 || row.ClassMask&classBit != 0) {
			spells = append(spells, row.Spell)
		}
	}

	return spells, nil
}

// GetPlayerCreateActions returns the starting action bar of a race / class.
func (w *WorldStore) GetPlayerCreateActions(race, class uint8) ([]store.PlayerCreateAction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := w.db.Collection("playercreateinfo_action").Find(ctx, bson.M{"race": race, "class": class})
	if err != nil {
		return nil, fmt.Errorf("GetPlayerCreateActions: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var actions []store.PlayerCreateAction

	for cursor.Next(ctx) {
		var row struct {
			Button uint8  `bson:"button"`
			Action uint32 `bson:"action"`
			Type   uint8  `bson:"type"`
		}

		if err := cursor.Decode(&row); err != nil {
			return nil, fmt.Errorf("GetPlayerCreateActions decode: %w", err)
		}

		actions = append(actions, store.PlayerCreateAction{Button: row.Button, Action: row.Action, Type: row.Type})
	}

	return actions, nil
}
