package model

import (
	"math/big"
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountToEntity(t *testing.T) {
	now := time.Now()

	acc := &store.Account{
		ID:        "test-id-123",
		Name:      "TESTPLAYER",
		Email:     "test@example.com",
		Salt:      big.NewInt(0xABC),
		Verifier:  big.NewInt(0xDEF),
		CreatedAt: now,
		Activated: true,
	}

	e := AccountToEntity(acc)

	assert.Equal(t, "TESTPLAYER", e.Name)
	assert.Equal(t, "test@example.com", e.Email)
	assert.Equal(t, "abc", e.Salt)
	assert.Equal(t, "def", e.Verifier)
	assert.Equal(t, now, e.CreatedAt)
	assert.True(t, e.Activated)
}

func TestEntityToAccount(t *testing.T) {
	now := time.Now()

	e := AccountEntity{
		Name:      "TESTPLAYER",
		Email:     "test@example.com",
		Salt:      "abc",
		Verifier:  "def",
		CreatedAt: now,
		Activated: true,
	}

	acc, err := EntityToAccount(e)
	require.NoError(t, err)

	assert.Equal(t, "TESTPLAYER", acc.Name)
	assert.Equal(t, "test@example.com", acc.Email)
	assert.Equal(t, big.NewInt(0xabc), acc.Salt)
	assert.Equal(t, big.NewInt(0xdef), acc.Verifier)
	assert.True(t, acc.Activated)
}

func TestAccountRoundTrip(t *testing.T) {
	now := time.Now()
	banExpiry := now.Add(24 * time.Hour)

	acc := &store.Account{
		ID:        "test-id",
		Name:      "PLAYERONE",
		Email:     "player@example.com",
		Salt:      big.NewInt(12345),
		Verifier:  big.NewInt(67890),
		CreatedAt: now,
		Activated: true,
		Ban: &store.AccountBan{
			BanReason: "test ban",
			BannedAt:  now.Format(time.RFC3339),
			BannedBy:  "admin",
			Expires:   &banExpiry,
		},
	}

	e := AccountToEntity(acc)

	roundTripped, err := EntityToAccount(e)
	require.NoError(t, err)

	assert.Equal(t, acc.Name, roundTripped.Name)
	assert.Equal(t, acc.Email, roundTripped.Email)
	assert.Equal(t, acc.Verifier, roundTripped.Verifier)
	assert.Equal(t, acc.Salt, roundTripped.Salt)
	assert.NotNil(t, roundTripped.Ban)
	assert.Equal(t, "test ban", roundTripped.Ban.BanReason)
	assert.Equal(t, "admin", roundTripped.Ban.BannedBy)
}

func TestPlayerToEntity(t *testing.T) {
	p := player.NewPlayer()
	p.ID = 42
	p.Name = "Arthas"
	p.Race = 6
	p.Class = 2
	p.Gender = 0
	p.Skin = 1
	p.Face = 2
	p.HairStyle = 3
	p.HairColor = 4
	p.FacialHair = 5
	p.Level = 10
	p.Money = 500
	p.Health = 200
	p.MaxHealth = 300
	p.GuildID = 100

	p.Location = player.WorldLocation{
		Map:  0,
		Zone: 12,
		X:    -8949.87,
		Y:    -132.45,
		Z:    83.53,
		O:    3.66,
	}

	p.KnownSpells = []uint32{20154, 21084}
	p.Actions[0] = 6603
	p.Actions[1] = 21084

	e := PlayerToEntity(p, "TESTACCOUNT")

	assert.Equal(t, uint32(42), e.GUID)
	assert.Equal(t, "TESTACCOUNT", e.Account)
	assert.Equal(t, "Arthas", e.Name)
	assert.Equal(t, uint8(6), e.Race)
	assert.Equal(t, uint8(2), e.Class)
	assert.Equal(t, uint8(10), e.Level)
	assert.Equal(t, uint32(500), e.Money)
	assert.Equal(t, float32(-8949.87), e.Location.X)
	assert.Equal(t, []uint32{20154, 21084}, e.KnownSpells)
	assert.Equal(t, uint32(6603), e.Actions[0])
}

func TestEntityToPlayer(t *testing.T) {
	e := CharacterEntity{
		GUID:      42,
		Name:      "Arthas",
		Race:      6,
		Class:     2,
		Gender:    0,
		Skin:      1,
		Face:      2,
		HairStyle: 3,
		HairColor: 4,
		FacialHair: 5,
		Level:     10,
		Money:     500,
		Health:    200,
		MaxHealth: 300,
		Power:     []uint32{0, 0, 0, 0, 0},
		MaxPower:  []uint32{0, 0, 0, 0, 0},
		Location: LocationEntity{
			Map:  0,
			Zone: 12,
			X:    -8949.87,
			Y:    -132.45,
			Z:    83.53,
			O:    3.66,
		},
		KnownSpells: []uint32{20154, 21084},
		Actions:     make([]uint32, 48),
		Inventory: []ItemEntity{
			{Slot: 0, Entry: 23322, Count: 1, Enchant: 0},
			{Slot: 15, Entry: 23344, Count: 1, Enchant: 0},
		},
	}

	e.Actions[0] = 6603
	e.Actions[1] = 21084

	p := EntityToPlayer(e)

	assert.Equal(t, uint32(42), p.ID)
	assert.Equal(t, "Arthas", p.Name)
	assert.Equal(t, uint8(10), p.Level)
	assert.Equal(t, uint32(500), p.Money)
	assert.Equal(t, float32(-8949.87), p.Location.X)
	assert.Equal(t, []uint32{20154, 21084}, p.KnownSpells)
	assert.Equal(t, uint32(6603), p.Actions[0])
	assert.Equal(t, uint32(21084), p.Actions[1])

	// Inventory should have items at slots 0 and 15
	require.NotNil(t, p.Inventory)
	item0 := p.Inventory.GetItem(0)
	require.NotNil(t, item0)
	assert.Equal(t, uint32(23322), item0.ItemEntry)

	item15 := p.Inventory.GetItem(15)
	require.NotNil(t, item15)
	assert.Equal(t, uint32(23344), item15.ItemEntry)
}

func TestPlayerRoundTrip(t *testing.T) {
	p := player.NewPlayer()
	p.ID = 99
	p.Name = "Thrall"
	p.Race = 2
	p.Class = 7
	p.Gender = 0
	p.Skin = 5
	p.Level = 60
	p.Money = 10000
	p.KnownSpells = []uint32{100, 200, 300}
	p.Actions[5] = 999

	e := PlayerToEntity(p, "HORDE")
	roundTripped := EntityToPlayer(e)

	assert.Equal(t, p.ID, roundTripped.ID)
	assert.Equal(t, p.Name, roundTripped.Name)
	assert.Equal(t, p.Race, roundTripped.Race)
	assert.Equal(t, p.Class, roundTripped.Class)
	assert.Equal(t, p.Level, roundTripped.Level)
	assert.Equal(t, p.Money, roundTripped.Money)
	assert.Equal(t, p.KnownSpells, roundTripped.KnownSpells)
	assert.Equal(t, p.Actions[5], roundTripped.Actions[5])
}

func TestItemToEntity(t *testing.T) {
	item := player.NewItem(23322, 100)
	item.StackCount = 5
	item.Enchantments[0] = 1234

	e := ItemToEntity(item, 15)

	assert.Equal(t, uint16(15), e.Slot)
	assert.Equal(t, uint32(23322), e.Entry)
	assert.Equal(t, uint32(5), e.Count)
	assert.Equal(t, uint32(1234), e.Enchant)
}

func TestEntityToItem(t *testing.T) {
	e := ItemEntity{
		Slot:       0,
		Entry:      23322,
		Count:      3,
		Enchant:    5678,
		Flags:      0,
		Durability: 100,
	}

	item := EntityToItem(e)

	assert.Equal(t, uint32(23322), item.ItemEntry)
	assert.Equal(t, uint32(3), item.StackCount)
	assert.Equal(t, uint32(100), item.Durability)
	assert.True(t, item.HasEnchant())
}

func TestItemRoundTrip(t *testing.T) {
	item := player.NewItem(4540, 50)
	item.StackCount = 10
	item.Durability = 50
	item.Enchantments[0] = 999

	e := ItemToEntity(item, 23)
	roundTripped := EntityToItem(e)

	assert.Equal(t, item.ItemEntry, roundTripped.ItemEntry)
	assert.Equal(t, item.StackCount, roundTripped.StackCount)
	assert.Equal(t, item.Durability, roundTripped.Durability)
	assert.True(t, roundTripped.HasEnchant())
}

func TestCollectionNames(t *testing.T) {
	assert.Equal(t, "accounts", AccountEntity{}.CollectionName())
	assert.Equal(t, "characters", CharacterEntity{}.CollectionName())
	assert.Equal(t, "counters", CounterEntity{}.CollectionName())
}
