package main

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/paalgyula/summit/internal/store/mongostore"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data from MySQL (AzerothCore) to MongoDB",
	Long: `Migrate imports the AzerothCore world database from MySQL into MongoDB.

It imports:
  - playercreateinfo → world.playerCreateInfo
  - playercreateinfo_item → world.playerCreateInfoItem
  - playercreateinfo_action → world.playerCreateInfoAction
  - playercreateinfo_spell → world.playerCreateInfoSpell
  - item_template → world.itemTemplate
  - creature_template → world.creatureTemplate
  - creature → world.creature
  - quest_template → world.questTemplate
  - player_levelstats → world.playerLevelStats
  - player_classlevelstats → world.playerClassLevelStats

Existing documents in the target collections are replaced.`,
	RunE: runMigrate,
}

func init() {
	// godotenv loads .env into os.Environ — viper picks it up via BindEnv
	_ = godotenv.Load()

	migrateCmd.Flags().String("mysql-dsn", "root:ac_password@tcp(127.0.0.1:3307)/world", "MySQL DSN")
	migrateCmd.Flags().String("mongo-uri", "mongodb://localhost:27017", "MongoDB URI")
	migrateCmd.Flags().String("mongo-db", "summit", "MongoDB database name")
	migrateCmd.Flags().Bool("drop", false, "Drop existing collections before import")
	migrateCmd.Flags().StringSlice("tables", nil, "Only import these tables (default: all)")

	// Bind env vars to flags: env overrides flag default, flag overrides env
	viper.BindEnv("mysql-dsn", "MYSQL_DSN")  //nolint:errcheck
	viper.BindEnv("mongo-uri", "MONGO_URI")  //nolint:errcheck
	viper.BindEnv("mongo-db", "MONGO_DB")    //nolint:errcheck
}

func runMigrate(cmd *cobra.Command, args []string) error {
	// viper resolves: CLI flag > env var > default
	_ = viper.BindPFlag("mysql-dsn", cmd.Flags().Lookup("mysql-dsn"))
	_ = viper.BindPFlag("mongo-uri", cmd.Flags().Lookup("mongo-uri"))
	_ = viper.BindPFlag("mongo-db", cmd.Flags().Lookup("mongo-db"))

	mysqlDSN := viper.GetString("mysql-dsn")
	mongoURI := viper.GetString("mongo-uri")
	mongoDBName := viper.GetString("mongo-db")
	drop, _ := cmd.Flags().GetBool("drop")
	only, _ := cmd.Flags().GetStringSlice("tables")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Connect to MySQL
	log.Info().Str("dsn", mysqlDSN).Msg("Connecting to MySQL")

	mysqlCfg, err := mysql.ParseDSN(mysqlDSN)
	if err != nil {
		return fmt.Errorf("mysql DSN parse: %w", err)
	}

	mysqlCfg.ParseTime = true
	mysqlCfg.MultiStatements = true

	mysqlDB, err := sql.Open("mysql", mysqlCfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("mysql open: %w", err)
	}

	defer mysqlDB.Close() //nolint:errcheck

	if err := mysqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("mysql ping: %w", err)
	}

	// Connect to MongoDB
	log.Info().Str("uri", mongoURI).Str("db", mongoDBName).Msg("Connecting to MongoDB")

	store, err := mongostore.Connect(ctx, mongoURI, mongoDBName)
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}

	defer store.Close(ctx) //nolint:errcheck

	client := store.Client()
	db := client.Database(mongoDBName)

	// Drop collections if requested
	if drop {
		log.Warn().Msg("Dropping existing collections")

		collections := []string{
			"playercreateinfo", "playercreateinfo_item", "playercreateinfo_action",
			"playercreateinfo_spell", "item_template", "creature_template",
			"creature", "quest_template", "player_levelstats", "player_classlevelstats",
			"creature_queststarter", "creature_questender",
			"gameobjectTemplate", "gameobject", "gameobjectLootTemplate",
		}

		for _, name := range collections {
			_ = db.Collection(name).Drop(ctx)
		}
	}

	// Import tables
	type importJob struct {
		name    string
		importF func(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error
	}

	jobs := []importJob{
		{"playercreateinfo", importPlayerCreateInfo},
		{"playercreateinfo_item", importPlayerCreateInfoItem},
		{"playercreateinfo_action", importPlayerCreateInfoAction},
		{"playercreateinfo_spell", importPlayerCreateInfoSpell},
		{"item_template", importItemTemplate},
		{"creature_template", importCreatureTemplate},
		{"creature", importCreature},
		{"quest_template", importQuestTemplate},
		{"creature_queststarter", importCreatureQuestStarter},
		{"creature_questender", importCreatureQuestEnder},
		{"player_levelstats", importPlayerLevelStats},
		{"player_classlevelstats", importPlayerClassLevelStats},
		{"gameobject_template", importGameObjectTemplate},
		{"gameobject", importGameObjectSpawn},
		{"gameobject_loot_template", importGameObjectLootTemplate},
	}

	total := len(jobs)

	for i, job := range jobs {
		if len(only) > 0 && !slices.Contains(only, job.name) {
			continue
		}

		log.Info().Msgf("[%d/%d] Importing %s...", i+1, total, job.name)

		start := time.Now()
		coll := db.Collection(job.name)

		if err := job.importF(ctx, mysqlDB, coll); err != nil {
			log.Error().Err(err).Str("table", job.name).Msg("Import failed")
			continue
		}

		count, _ := coll.CountDocuments(ctx, bson.M{})

		log.Info().Msgf("[%d/%d] %s imported (%d docs) in %s", i+1, total, job.name, count, time.Since(start))
	}

	log.Info().Msg("Migration complete!")

	return nil
}

// --- Import functions ---

func importPlayerCreateInfo(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx, "SELECT race, class, map, zone, position_x, position_y, position_z, orientation FROM playercreateinfo")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			race, class uint8
			mapID       uint16
			zone        uint32
			x, y, z, o  float32
		)

		if err := rows.Scan(&race, &class, &mapID, &zone, &x, &y, &z, &o); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":         fmt.Sprintf("%d_%d", race, class),
			"race":        race,
			"class":       class,
			"map":         mapID,
			"zone":        zone,
			"positionX":   x,
			"positionY":   y,
			"positionZ":   z,
			"orientation": o,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importPlayerCreateInfoItem(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx, "SELECT race, class, itemid, amount FROM playercreateinfo_item")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			race, class uint8
			itemID      uint32
			amount      int8
		)

		if err := rows.Scan(&race, &class, &itemID, &amount); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":    fmt.Sprintf("%d_%d_%d", race, class, itemID),
			"race":   race,
			"class":  class,
			"itemId": itemID,
			"amount": amount,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importPlayerCreateInfoAction(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx, "SELECT race, class, button, action, type FROM playercreateinfo_action")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			race, class uint8
			button      uint8
			action      uint32
			actionType  uint16
		)

		if err := rows.Scan(&race, &class, &button, &action, &actionType); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":    fmt.Sprintf("%d_%d_%d", race, class, button),
			"race":   race,
			"class":  class,
			"button": button,
			"action": action,
			"type":   actionType,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importPlayerCreateInfoSpell(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx, "SELECT racemask, classmask, Spell FROM playercreateinfo_spell")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			raceMask, classMask, spell uint32
		)

		if err := rows.Scan(&raceMask, &classMask, &spell); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":       fmt.Sprintf("%d_%d_%d", raceMask, classMask, spell),
			"raceMask":  raceMask,
			"classMask": classMask,
			"spell":     spell,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importItemTemplate(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		`SELECT entry, class, subclass, name, displayid, quality, InventoryType,
		        AllowableClass, AllowableRace, ItemLevel, RequiredLevel, RequiredSkill, RequiredSkillRank,
		        BuyCount, BuyPrice, SellPrice, MaxCount, Stackable,
		        StatsCount, stat_type1, stat_value1, stat_type2, stat_value2,
		        stat_type3, stat_value3, stat_type4, stat_value4, stat_type5, stat_value5,
		        stat_type6, stat_value6, stat_type7, stat_value7, stat_type8, stat_value8,
		        stat_type9, stat_value9, stat_type10, stat_value10,
		        dmg_type1, dmg_min1, dmg_max1, dmg_type2, dmg_min2, dmg_max2,
		        armor, holy_res, fire_res, nature_res, frost_res, shadow_res, arcane_res,
		        delay, AmmoType, RangedModRange,
		        spellid_1, spelltrigger_1, spellcharges_1, spellppmrate_1, spellcooldown_1, spellcategory_1, spellcategorycooldown_1,
		        spellid_2, spelltrigger_2, spellcharges_2, spellppmrate_2, spellcooldown_2, spellcategory_2, spellcategorycooldown_2,
		        spellid_3, spelltrigger_3, spellcharges_3, spellppmrate_3, spellcooldown_3, spellcategory_3, spellcategorycooldown_3,
		        spellid_4, spelltrigger_4, spellcharges_4, spellppmrate_4, spellcooldown_4, spellcategory_4, spellcategorycooldown_4,
		        spellid_5, spelltrigger_5, spellcharges_5, spellppmrate_5, spellcooldown_5, spellcategory_5, spellcategorycooldown_5,
		        socket_color_1, socket_content_1, socket_color_2, socket_content_2, socket_color_3, socket_content_3, socketBonus,
		        Bonding, LockID, Material, Sheath, ItemSet, MaxDurability,
		        RandomProperty, RandomSuffix, Block,
		        ContainerSlots, BagFamily, Flags,
		        Duration, DisenchantID, FoodType, MinMoneyLoot, MaxMoneyLoot,
		        Description, PageText, LanguageID, PageMaterial, StartQuest,
		        RequiredHonorRank, RequiredCityRank, RequiredReputationFaction, RequiredReputationRank
		 FROM item_template`)
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			entry, displayID, quality, invType                              uint32
			class, subclass                                                 uint32
			allowableClass                                                  int32
			allowableRace                                                   int32
			itemLevel, requiredLevel, requiredSkill, requiredSkillRank       uint32
			buyCount                                                        uint32
			buyPrice                                                        int64
			sellPrice, maxCount, stackable                                  uint32
			statsCount                                                      uint32
			statType [10]uint32
			statVal  [10]int32
			dmgType  [2]uint32
			dmgMin   [2]float32
			dmgMax   [2]float32
			armor                                                            uint32
			holyRes, fireRes, natureRes, frostRes, shadowRes, arcaneRes      int32
			delay                                                            uint32
			ammoType, rangedModRange                                         uint32
			spellID        [5]int32
			spellTrigger   [5]uint32
			spellCharges   [5]int32
			spellPPMRate   [5]float32
			spellCooldown  [5]int32
			spellCategory  [5]uint32
			spellCatCD     [5]int32
			socketColor    [3]uint32
			socketContent  [3]uint32
			socketBonus                                                       uint32
			bonding, lockID, material, sheath, itemSet, maxDurability         uint32
			randomProperty, randomSuffix, block                              int32
			containerSlots, bagFamily, flags                                 uint32
			duration                                                          uint32
			disenchantID, foodType, minMoneyLoot, maxMoneyLoot               uint32
			description                                                       string
			pageText, languageID, pageMaterial, startQuest                    uint32
			reqHonorRank, reqCityRank, reqRepFaction, reqRepRank             uint32
		)

		if err := rows.Scan(
			&entry, &class, &subclass, &displayID, &quality, &invType,
			&allowableClass, &allowableRace, &itemLevel, &requiredLevel, &requiredSkill, &requiredSkillRank,
			&buyCount, &buyPrice, &sellPrice, &maxCount, &stackable,
			&statsCount,
			&statType[0], &statVal[0], &statType[1], &statVal[1],
			&statType[2], &statVal[2], &statType[3], &statVal[3],
			&statType[4], &statVal[4], &statType[5], &statVal[5],
			&statType[6], &statVal[6], &statType[7], &statVal[7],
			&statType[8], &statVal[8], &statType[9], &statVal[9],
			&dmgType[0], &dmgMin[0], &dmgMax[0], &dmgType[1], &dmgMin[1], &dmgMax[1],
			&armor, &holyRes, &fireRes, &natureRes, &frostRes, &shadowRes, &arcaneRes,
			&delay, &ammoType, &rangedModRange,
			&spellID[0], &spellTrigger[0], &spellCharges[0], &spellPPMRate[0], &spellCooldown[0], &spellCategory[0], &spellCatCD[0],
			&spellID[1], &spellTrigger[1], &spellCharges[1], &spellPPMRate[1], &spellCooldown[1], &spellCategory[1], &spellCatCD[1],
			&spellID[2], &spellTrigger[2], &spellCharges[2], &spellPPMRate[2], &spellCooldown[2], &spellCategory[2], &spellCatCD[2],
			&spellID[3], &spellTrigger[3], &spellCharges[3], &spellPPMRate[3], &spellCooldown[3], &spellCategory[3], &spellCatCD[3],
			&spellID[4], &spellTrigger[4], &spellCharges[4], &spellPPMRate[4], &spellCooldown[4], &spellCategory[4], &spellCatCD[4],
			&socketColor[0], &socketContent[0], &socketColor[1], &socketContent[1], &socketColor[2], &socketContent[2], &socketBonus,
			&bonding, &lockID, &material, &sheath, &itemSet, &maxDurability,
			&randomProperty, &randomSuffix, &block,
			&containerSlots, &bagFamily, &flags,
			&duration, &disenchantID, &foodType, &minMoneyLoot, &maxMoneyLoot,
			&description, &pageText, &languageID, &pageMaterial, &startQuest,
			&reqHonorRank, &reqCityRank, &reqRepFaction, &reqRepRank,
		); err != nil {
			return err
		}

		stats := make([]bson.M, 0, statsCount)

		for i := uint32(0); i < statsCount && i < 10; i++ {
			if statType[i] != 0 {
				stats = append(stats, bson.M{"type": statType[i], "value": statVal[i]})
			}
		}

		damage := make([]bson.M, 0, 2)

		for i := uint32(0); i < 2; i++ {
			if dmgMin[i] > 0 || dmgMax[i] > 0 {
				damage = append(damage, bson.M{"type": dmgType[i], "min": dmgMin[i], "max": dmgMax[i]})
			}
		}

		spells := make([]bson.M, 0, 5)

		for i := uint32(0); i < 5; i++ {
			if spellID[i] != 0 {
				spells = append(spells, bson.M{
					"spellId":          spellID[i],
					"trigger":          spellTrigger[i],
					"charges":          spellCharges[i],
					"ppmRate":          spellPPMRate[i],
					"cooldown":         spellCooldown[i],
					"category":         spellCategory[i],
					"categoryCooldown": spellCatCD[i],
				})
			}
		}

		sockets := make([]bson.M, 0, 3)

		for i := uint32(0); i < 3; i++ {
			if socketColor[i] != 0 {
				sockets = append(sockets, bson.M{"color": socketColor[i], "content": socketContent[i]})
			}
		}

		docs = append(docs, bson.M{
			"_id":             entry,
			"entry":           entry,
			"class":           class,
			"subclass":        subclass,
			"name":            strings.TrimRight(description, "\x00"),
			"displayId":       displayID,
			"quality":         quality,
			"inventoryType":   invType,
			"allowableClass":  allowableClass,
			"allowableRace":   allowableRace,
			"itemLevel":       itemLevel,
			"requiredLevel":   requiredLevel,
			"requiredSkill":   requiredSkill,
			"requiredSkillRank": requiredSkillRank,
			"buyCount":        buyCount,
			"buyPrice":        buyPrice,
			"sellPrice":       sellPrice,
			"maxCount":        int32(maxCount),
			"stackable":       int32(stackable),
			"statsCount":      statsCount,
			"stats":           stats,
			"damage":          damage,
			"armor":           armor,
			"holyRes":         holyRes,
			"fireRes":         fireRes,
			"natureRes":       natureRes,
			"frostRes":        frostRes,
			"shadowRes":       shadowRes,
			"arcaneRes":       arcaneRes,
			"delay":           delay,
			"ammoType":        ammoType,
			"rangedModRange":  rangedModRange,
			"spells":          spells,
			"sockets":         sockets,
			"socketBonus":     socketBonus,
			"bonding":         bonding,
			"lockId":          lockID,
			"material":        int32(material),
			"sheath":          sheath,
			"itemSet":         itemSet,
			"maxDurability":   maxDurability,
			"randomProperty":  randomProperty,
			"randomSuffix":    randomSuffix,
			"block":           block,
			"containerSlots":  containerSlots,
			"bagFamily":       bagFamily,
			"flags":           flags,
			"duration":        duration,
			"disenchantId":    disenchantID,
			"foodType":        foodType,
			"minMoneyLoot":    minMoneyLoot,
			"maxMoneyLoot":    maxMoneyLoot,
			"pageText":        pageText,
			"languageId":      languageID,
			"pageMaterial":    pageMaterial,
			"startQuest":      startQuest,
			"requiredHonorRank":  reqHonorRank,
			"requiredCityRank":   reqCityRank,
			"requiredRepFaction": reqRepFaction,
			"requiredRepRank":    reqRepRank,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importCreatureTemplate(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		`SELECT entry, name, subname, minlevel, maxlevel, faction, npcflag,
		        modelid1, modelid2, modelid3, modelid4, scale,
		        rank, type, family, unit_class, unit_flags, dynamicflags,
		        speed_walk, speed_run, baseattacktime, MovementType,
		        Health_mod, dmg_multiplier, Armor_mod, flags_extra
		 FROM creature_template`)
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			entry                          uint32
			name                           string
			subName                        sql.NullString
			minLevel, maxLevel             uint8
			faction                        uint16
			npcFlag                        uint32
			model1, model2, model3, model4 uint32
			scale                          float32
			rank, ctype, family, unitClass uint32
			unitFlags, dynamicFlags        uint32
			speedWalk, speedRun            float32
			baseAttackTime, movementType   uint32
			healthMod, damageMod, armorMod float32
			flagsExtra                     uint32
		)

		if err := rows.Scan(&entry, &name, &subName, &minLevel, &maxLevel, &faction, &npcFlag,
			&model1, &model2, &model3, &model4, &scale,
			&rank, &ctype, &family, &unitClass, &unitFlags, &dynamicFlags,
			&speedWalk, &speedRun, &baseAttackTime, &movementType,
			&healthMod, &damageMod, &armorMod, &flagsExtra); err != nil {
			return err
		}

		// Display ids the client picks from at random; zero entries are unused slots
		modelIDs := make([]uint32, 0, 4)
		for _, id := range []uint32{model1, model2, model3, model4} {
			if id != 0 {
				modelIDs = append(modelIDs, id)
			}
		}

		docs = append(docs, bson.M{
			"_id":              entry,
			"entry":            entry,
			"name":             strings.TrimRight(name, "\x00"),
			"subName":          subName.String,
			"minLevel":         minLevel,
			"maxLevel":         maxLevel,
			"faction":          faction,
			"npcFlag":          npcFlag,
			"modelIds":         modelIDs,
			"scale":            scale,
			"rank":             rank,
			"type":             ctype,
			"family":           family,
			"unitClass":        unitClass,
			"unitFlags":        unitFlags,
			"dynamicFlags":     dynamicFlags,
			"speedWalk":        speedWalk,
			"speedRun":         speedRun,
			"baseAttackTime":   baseAttackTime,
			"movementType":     movementType,
			"healthMultiplier": healthMod,
			"damageMultiplier": damageMod,
			"armorMultiplier":  armorMod,
			"flagsExtra":       flagsExtra,
		})
	}

	return upsertDocs(ctx, coll, docs)
}

func importCreature(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		"SELECT guid, id, map, position_x, position_y, position_z, orientation, spawntimesecs FROM creature")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			guid       uint32
			entry      uint32
			mapID      uint16
			x, y, z, o float32
			spawnTime  uint32
		)

		if err := rows.Scan(&guid, &entry, &mapID, &x, &y, &z, &o, &spawnTime); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":         guid,
			"guid":        guid,
			"entry":       entry,
			"map":         mapID,
			"positionX":   x,
			"positionY":   y,
			"positionZ":   z,
			"orientation": o,
			"spawnTime":   spawnTime,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importQuestTemplate(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		"SELECT ID, QuestLevel, MinLevel, QuestType, RequiredClasses, RequiredRaces FROM quest_template")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			id              uint32
			questLevel      int16
			minLevel        uint8
			questType       uint16
			requiredClasses uint16
			requiredRaces   uint16
		)

		if err := rows.Scan(&id, &questLevel, &minLevel, &questType, &requiredClasses, &requiredRaces); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":             id,
			"id":              id,
			"questLevel":      questLevel,
			"minLevel":        minLevel,
			"questType":       questType,
			"requiredClasses": requiredClasses,
			"requiredRaces":   requiredRaces,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importCreatureQuestStarter(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		"SELECT id, Quest FROM creature_queststarter")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			creatureEntry uint32
			questID       uint32
		)

		if err := rows.Scan(&creatureEntry, &questID); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":           fmt.Sprintf("%d_%d", creatureEntry, questID),
			"creatureEntry": creatureEntry,
			"questId":       questID,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importCreatureQuestEnder(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		"SELECT id, Quest FROM creature_questender")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			creatureEntry uint32
			questID       uint32
		)

		if err := rows.Scan(&creatureEntry, &questID); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":           fmt.Sprintf("%d_%d", creatureEntry, questID),
			"creatureEntry": creatureEntry,
			"questId":       questID,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importPlayerLevelStats(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		"SELECT race, class, level, str, agi, sta, inte, spi FROM player_levelstats")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			race, class, level, str, agi, sta, intVal, spi uint8
		)

		if err := rows.Scan(&race, &class, &level, &str, &agi, &sta, &intVal, &spi); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":   fmt.Sprintf("%d_%d_%d", race, class, level),
			"race":  race,
			"class": class,
			"level": level,
			"str":   str,
			"agi":   agi,
			"sta":   sta,
			"int":   intVal,
			"spi":   spi,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importPlayerClassLevelStats(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		"SELECT class, level, basehp, basemana FROM player_classlevelstats")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			class    uint8
			level    uint8
			baseHP   uint32
			baseMana uint32
		)

		if err := rows.Scan(&class, &level, &baseHP, &baseMana); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":      fmt.Sprintf("%d_%d", class, level),
			"class":    class,
			"level":    level,
			"baseHP":   baseHP,
			"baseMana": baseMana,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importGameObjectTemplate(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		`SELECT entry, type, displayId, name, IconName, size,
		        Data0,Data1,Data2,Data3,Data4,Data5,Data6,Data7,
		        Data8,Data9,Data10,Data11,Data12,Data13,Data14,Data15,
		        Data16,Data17,Data18,Data19,Data20,Data21,Data22,Data23,
		        AIName, ScriptName
		 FROM gameobject_template`)
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			entry      uint32
			goType     uint8
			displayID  uint32
			name       string
			iconName   string
			size       float32
			data       [24]int32
			aiName     string
			scriptName string
		)

		if err := rows.Scan(
			&entry, &goType, &displayID, &name, &iconName, &size,
			&data[0], &data[1], &data[2], &data[3], &data[4], &data[5], &data[6], &data[7],
			&data[8], &data[9], &data[10], &data[11], &data[12], &data[13], &data[14], &data[15],
			&data[16], &data[17], &data[18], &data[19], &data[20], &data[21], &data[22], &data[23],
			&aiName, &scriptName,
		); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":        entry,
			"entry":      entry,
			"type":       goType,
			"displayId":  displayID,
			"name":       strings.TrimRight(name, "\x00"),
			"iconName":   iconName,
			"size":       size,
			"data":       data,
			"aiName":     aiName,
			"scriptName": scriptName,
		})
	}

	return upsertDocs(ctx, coll, docs)
}

func importGameObjectSpawn(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		`SELECT guid, id, map, spawnMask, phaseMask,
		        position_x, position_y, position_z, orientation,
		        rotation0, rotation1, rotation2, rotation3,
		        spawntimesecs, animprogress, state
		 FROM gameobject`)
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			guid          uint32
			entry         uint32
			mapID         uint16
			spawnMask     uint8
			phaseMask     uint32
			posX, posY    float32
			posZ, orient  float32
			rot0, rot1    float32
			rot2, rot3    float32
			spawnTimeSecs int32
			animProgress  uint8
			state         uint8
		)

		if err := rows.Scan(
			&guid, &entry, &mapID, &spawnMask, &phaseMask,
			&posX, &posY, &posZ, &orient,
			&rot0, &rot1, &rot2, &rot3,
			&spawnTimeSecs, &animProgress, &state,
		); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":           guid,
			"guid":          guid,
			"entry":         entry,
			"map":           mapID,
			"spawnMask":     spawnMask,
			"phaseMask":     phaseMask,
			"positionX":     posX,
			"positionY":     posY,
			"positionZ":     posZ,
			"orientation":   orient,
			"rotation":      [4]float32{rot0, rot1, rot2, rot3},
			"spawnTimeSecs": spawnTimeSecs,
			"animProgress":  animProgress,
			"state":         state,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importGameObjectLootTemplate(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		`SELECT entry, item, ChanceOrQuestChance, lootmode, groupid, mincountOrRef, maxcount
		 FROM gameobject_loot_template`)
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			entry        uint32
			item         uint32
			chance       float32
			lootMode     uint16
			groupID      uint8
			mincountOrRef int32
			maxCount     uint8
		)

		if err := rows.Scan(
			&entry, &item, &chance, &lootMode, &groupID, &mincountOrRef, &maxCount,
		); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":           fmt.Sprintf("%d_%d", entry, item),
			"entry":         entry,
			"item":          item,
			"challenge":     chance,
			"lootMode":      lootMode,
			"groupId":       groupID,
			"mincountOrRef": mincountOrRef,
			"maxCount":      maxCount,
		})
	}

	return insertDocs(ctx, coll, docs)
}

// --- Helpers ---

// insertDocs inserts documents using ordered bulk write with upsert.
// upsertDocs replaces documents by _id (inserting missing ones), so a
// re-run refreshes rows the schema grew new fields for.
func upsertDocs(ctx context.Context, coll *mongo.Collection, docs []interface{}) error {
	if len(docs) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(docs))

	for _, doc := range docs {
		m, ok := doc.(bson.M)
		if !ok {
			return fmt.Errorf("upsertDocs: unexpected document type %T", doc)
		}

		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.M{"_id": m["_id"]}).
			SetReplacement(m).
			SetUpsert(true))
	}

	_, err := coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	if err != nil {
		return fmt.Errorf("bulk upsert: %w", err)
	}

	return nil
}

func insertDocs(ctx context.Context, coll *mongo.Collection, docs []interface{}) error {
	if len(docs) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(docs))

	for _, doc := range docs {
		model := mongo.NewInsertOneModel().SetDocument(doc)
		models = append(models, model)
	}

	_, err := coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))

	// Ignore duplicate key errors (idempotent import)
	if err != nil && !isDuplicateKeyError(err) {
		return fmt.Errorf("bulk insert: %w", err)
	}

	return nil
}

func isDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "E11000 duplicate key")
}
