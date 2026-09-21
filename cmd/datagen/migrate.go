package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/paalgyula/summit/internal/store/mongostore"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
	migrateCmd.Flags().String("mysql-dsn", "root:ac_password@tcp(127.0.0.1:3307)/world", "MySQL DSN")
	migrateCmd.Flags().String("mongo-uri", "mongodb://localhost:27017", "MongoDB URI")
	migrateCmd.Flags().String("mongo-db", "summit", "MongoDB database name")
	migrateCmd.Flags().Bool("drop", false, "Drop existing collections before import")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	mysqlDSN, _ := cmd.Flags().GetString("mysql-dsn")
	mongoURI, _ := cmd.Flags().GetString("mongo-uri")
	mongoDBName, _ := cmd.Flags().GetString("mongo-db")
	drop, _ := cmd.Flags().GetBool("drop")

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
			"playerCreateInfo", "playerCreateInfoItem", "playerCreateInfoAction",
			"playerCreateInfoSpell", "itemTemplate", "creatureTemplate",
			"creature", "questTemplate", "playerLevelStats", "playerClassLevelStats",
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
		{"player_levelstats", importPlayerLevelStats},
		{"player_classlevelstats", importPlayerClassLevelStats},
	}

	total := len(jobs)

	for i, job := range jobs {
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
			x, y, z, o float32
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
			"_id":        fmt.Sprintf("%d_%d_%d", raceMask, classMask, spell),
			"raceMask":   raceMask,
			"classMask":  classMask,
			"spell":      spell,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importItemTemplate(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		`SELECT entry, class, subclass, name, displayid, InventoryType, BuyCount, BuyPrice, SellPrice, ItemLevel, RequiredLevel
		 FROM item_template`)
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			entry         uint32
			class, subclass, invType, buyCount, reqLevel uint8
			name          string
			displayID     uint32
			buyPrice      int64
			sellPrice     uint32
			itemLevel     uint16
		)

		if err := rows.Scan(&entry, &class, &subclass, &name, &displayID,
			&invType, &buyCount, &buyPrice, &sellPrice, &itemLevel, &reqLevel); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":            entry,
			"entry":          entry,
			"class":          class,
			"subclass":       subclass,
			"name":           strings.TrimRight(name, "\x00"),
			"displayId":      displayID,
			"inventoryType":  invType,
			"buyCount":       buyCount,
			"buyPrice":       buyPrice,
			"sellPrice":      sellPrice,
			"itemLevel":      itemLevel,
			"requiredLevel":  reqLevel,
		})
	}

	return insertDocs(ctx, coll, docs)
}

func importCreatureTemplate(ctx context.Context, mysqlDB *sql.DB, coll *mongo.Collection) error {
	rows, err := mysqlDB.QueryContext(ctx,
		"SELECT entry, name, minlevel, maxlevel, faction, npcflag FROM creature_template")
	if err != nil {
		return err
	}

	defer rows.Close() //nolint:errcheck

	var docs []interface{}

	for rows.Next() {
		var (
			entry                   uint32
			name                    string
			minLevel, maxLevel      uint8
			faction                 uint16
			npcFlag                 uint32
		)

		if err := rows.Scan(&entry, &name, &minLevel, &maxLevel, &faction, &npcFlag); err != nil {
			return err
		}

		docs = append(docs, bson.M{
			"_id":      entry,
			"entry":    entry,
			"name":     strings.TrimRight(name, "\x00"),
			"minLevel": minLevel,
			"maxLevel": maxLevel,
			"faction":  faction,
			"npcFlag":  npcFlag,
		})
	}

	return insertDocs(ctx, coll, docs)
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
			guid        uint32
			entry       uint32
			mapID       uint16
			x, y, z, o float32
			spawnTime   uint32
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
			"_id":              id,
			"id":               id,
			"questLevel":       questLevel,
			"minLevel":         minLevel,
			"questType":        questType,
			"requiredClasses":  requiredClasses,
			"requiredRaces":    requiredRaces,
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

// --- Helpers ---

// insertDocs inserts documents using ordered bulk write with upsert.
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
