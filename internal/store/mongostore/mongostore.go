// Package mongostore implements store.AccountRepo and store.CharacterRepo
// using MongoDB as the backing database.
package mongostore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/store/model"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store implements store.AccountRepo and store.CharacterRepo.
type Store struct {
	client     *mongo.Client
	db         *mongo.Database
	accounts   *mongo.Collection
	characters *mongo.Collection
	counters   *mongo.Collection
}

// New creates a new MongoDB-backed store.
func New(client *mongo.Client, database string) *Store {
	db := client.Database(database)

	return &Store{
		client:     client,
		db:         db,
		accounts:   db.Collection("accounts"),
		characters: db.Collection("characters"),
		counters:   db.Collection("counters"),
	}
}

// Connect opens a MongoDB connection and returns a Store.
func Connect(ctx context.Context, uri, database string) (*Store, error) {
	clientOpts := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongostore.Connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongostore.Connect ping: %w", err)
	}

	// Ensure indexes
	s := New(client, database)

	if err := s.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("mongostore.Connect indexes: %w", err)
	}

	return s, nil
}

// ensureIndexes creates the required unique indexes.
func (s *Store) ensureIndexes(ctx context.Context) error {
	// accounts: unique name
	_, err := s.accounts.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("accounts name index: %w", err)
	}

	// characters: unique guid
	_, err = s.characters.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "guid", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("characters guid index: %w", err)
	}

	// characters: account lookup
	_, err = s.characters.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "account", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("characters account index: %w", err)
	}

	// characters: unique name
	_, err = s.characters.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("characters name index: %w", err)
	}

	return nil
}

// --- AccountRepo ---

// FindAccount retrieves an account by name (case-insensitive).
func (s *Store) FindAccount(name string) *store.Account {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var entity model.AccountEntity

	err := s.accounts.FindOne(ctx, bson.M{"name": strings.ToUpper(name)}).Decode(&entity)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			log.Error().Err(err).Str("name", name).Msg("FindAccount query failed")
		}

		return nil
	}

	acc, err := model.EntityToAccount(entity)
	if err != nil {
		log.Error().Err(err).Str("name", name).Msg("FindAccount entity conversion failed")

		return nil
	}

	return acc
}

// CreateAccount persists a new account.
func (s *Store) CreateAccount(acc *store.Account) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entity := model.AccountToEntity(acc)
	entity.Name = strings.ToUpper(entity.Name)

	result, err := s.accounts.InsertOne(ctx, entity)
	if err != nil {
		return fmt.Errorf("CreateAccount: %w", err)
	}

	if oid, ok := result.InsertedID.(interface{ String() string }); ok {
		acc.ID = oid.String()
	}

	return nil
}

// --- CharacterRepo ---

// GetCharacters retrieves all characters for an account.
func (s *Store) GetCharacters(account string) (player.Players, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.characters.Find(ctx, bson.M{"account": strings.ToUpper(account)})
	if err != nil {
		return nil, fmt.Errorf("GetCharacters: %w", err)
	}

	defer cursor.Close(ctx) //nolint:errcheck

	var entities []model.CharacterEntity

	if err := cursor.All(ctx, &entities); err != nil {
		return nil, fmt.Errorf("GetCharacters decode: %w", err)
	}

	players := make(player.Players, 0, len(entities))

	for _, e := range entities {
		p := model.EntityToPlayer(e)
		players = append(players, p)
	}

	return players, nil
}

// GetCharacter retrieves a single character by GUID.
func (s *Store) GetCharacter(guid uint32) (*player.Player, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var entity model.CharacterEntity

	err := s.characters.FindOne(ctx, bson.M{"guid": guid}).Decode(&entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, fmt.Errorf("GetCharacter: %w", err)
	}

	return model.EntityToPlayer(entity), nil
}

// CreateCharacter persists a new character and assigns it a GUID.
func (s *Store) CreateCharacter(account string, character *player.Player) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Allocate GUID
	guid, err := s.nextGUID(ctx)
	if err != nil {
		return fmt.Errorf("CreateCharacter guid allocation: %w", err)
	}

	character.ID = guid

	entity := model.PlayerToEntity(character, strings.ToUpper(account))
	entity.GUID = guid

	_, err = s.characters.InsertOne(ctx, entity)
	if err != nil {
		return fmt.Errorf("CreateCharacter: %w", err)
	}

	return nil
}

// UpdateCharacter updates an existing character by GUID.
func (s *Store) UpdateCharacter(character *player.Player) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// We need the account name — look it up first
	var entity model.CharacterEntity

	err := s.characters.FindOne(ctx, bson.M{"guid": character.ID}).Decode(&entity)
	if err != nil {
		return fmt.Errorf("UpdateCharacter lookup: %w", err)
	}

	updated := model.PlayerToEntity(character, entity.Account)
	updated.ID = entity.ID // preserve ObjectID
	updated.GUID = character.ID
	updated.UpdatedAt = time.Now()

	_, err = s.characters.ReplaceOne(ctx, bson.M{"guid": character.ID}, updated)
	if err != nil {
		return fmt.Errorf("UpdateCharacter: %w", err)
	}

	return nil
}

// UpdateCharacterQuests persists only the character's quest progress (active
// quests and rewarded quest ids) using a single atomic $set. Quests are embedded
// in the character document, so this avoids a separate collection and does not
// rewrite the rest of the document.
func (s *Store) UpdateCharacterQuests(character *player.Player) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quests, rewarded := model.QuestProgressToEntity(character)

	update := bson.M{
		"$set": bson.M{
			"quests":         quests,
			"rewardedQuests": rewarded,
			"updatedAt":      time.Now(),
		},
	}

	result, err := s.characters.UpdateOne(ctx, bson.M{"guid": character.ID}, update)
	if err != nil {
		return fmt.Errorf("UpdateCharacterQuests: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("character %d not found", character.ID)
	}

	return nil
}

// DeleteCharacter removes a character by GUID.
func (s *Store) DeleteCharacter(characterID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.characters.DeleteOne(ctx, bson.M{"guid": uint32(characterID)})
	if err != nil {
		return fmt.Errorf("DeleteCharacter: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("character %d not found", characterID)
	}

	return nil
}

// --- GUID allocation ---

// nextGUID atomically increments and returns the next character GUID.
func (s *Store) nextGUID(ctx context.Context) (uint32, error) {
	filter := bson.M{"_id": "character_guid"}
	update := bson.M{"$inc": bson.M{"seq": int64(1)}}
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After).
		SetUpsert(true)

	var counter model.CounterEntity

	err := s.counters.FindOneAndUpdate(ctx, filter, update, opts).Decode(&counter)
	if err != nil {
		return 0, fmt.Errorf("nextGUID: %w", err)
	}

	return uint32(counter.Seq), nil
}

// Client returns the underlying MongoDB client (for advanced operations).
func (s *Store) Client() *mongo.Client {
	return s.client
}

// Database returns the MongoDB database name.
func (s *Store) Database() string {
	return s.db.Name()
}

// DB returns the underlying MongoDB database (for creating sub-stores like WorldStore).
func (s *Store) DB() *mongo.Database {
	return s.db
}

// Close disconnects from MongoDB.
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
