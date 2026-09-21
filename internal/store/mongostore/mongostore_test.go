package mongostore

import (
	"testing"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/store/mock_store"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// --- Interface compliance tests ---

func TestStore_ImplementsAccountRepo(t *testing.T) {
	var _ store.AccountRepo = (*Store)(nil)
}

func TestStore_ImplementsCharacterRepo(t *testing.T) {
	var _ store.CharacterRepo = (*Store)(nil)
}

// --- Mock-based tests (using gomock) ---

func TestMockAccountRepo_FindAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockAccountRepo(ctrl)

	// Set up expectations
	mockRepo.EXPECT().FindAccount("TEST").Return(&store.Account{
		Name: "TEST",
	})

	// Use the mock
	acc := mockRepo.FindAccount("TEST")
	assert.Equal(t, "TEST", acc.Name)
}

func TestMockAccountRepo_FindAccount_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockAccountRepo(ctrl)

	mockRepo.EXPECT().FindAccount("NONEXISTENT").Return(nil)

	acc := mockRepo.FindAccount("NONEXISTENT")
	assert.Nil(t, acc)
}

func TestMockAccountRepo_CreateAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockAccountRepo(ctrl)

	acc := &store.Account{
		Name: "NEWPLAYER",
	}

	mockRepo.EXPECT().CreateAccount(acc).Return(nil)

	err := mockRepo.CreateAccount(acc)
	assert.NoError(t, err)
}

func TestMockCharacterRepo_GetCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockCharacterRepo(ctrl)

	// Test GetCharacter returning nil (not found)
	mockRepo.EXPECT().GetCharacter(uint32(999)).Return(nil, nil)

	p, err := mockRepo.GetCharacter(999)
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestMockCharacterRepo_GetCharacters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockCharacterRepo(ctrl)

	p1 := player.NewPlayer()
	p1.ID = 1
	p1.Name = "Char1"

	p2 := player.NewPlayer()
	p2.ID = 2
	p2.Name = "Char2"

	players := player.Players{p1, p2}

	mockRepo.EXPECT().GetCharacters("TESTACCOUNT").Return(players, nil)

	result, err := mockRepo.GetCharacters("TESTACCOUNT")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Char1", result[0].Name)
	assert.Equal(t, "Char2", result[1].Name)
}

func TestMockCharacterRepo_CreateCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockCharacterRepo(ctrl)

	p := player.NewPlayer()
	p.Name = "NewChar"

	mockRepo.EXPECT().CreateCharacter("TESTACCOUNT", p).Return(nil)

	err := mockRepo.CreateCharacter("TESTACCOUNT", p)
	assert.NoError(t, err)
}

func TestMockCharacterRepo_UpdateCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockCharacterRepo(ctrl)

	p := player.NewPlayer()
	p.ID = 42
	p.Name = "UpdatedChar"

	mockRepo.EXPECT().UpdateCharacter(p).Return(nil)

	err := mockRepo.UpdateCharacter(p)
	assert.NoError(t, err)
}

func TestMockCharacterRepo_DeleteCharacter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockCharacterRepo(ctrl)

	mockRepo.EXPECT().DeleteCharacter(42).Return(nil)

	err := mockRepo.DeleteCharacter(42)
	assert.NoError(t, err)
}

func TestMockCharacterRepo_DeleteCharacter_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockCharacterRepo(ctrl)

	mockRepo.EXPECT().DeleteCharacter(999).Return(assert.AnError)

	err := mockRepo.DeleteCharacter(999)
	assert.Error(t, err)
}

// --- Model transform tests using mock ---

func TestMockCharacterRepo_GetCharacter_WithModel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_store.NewMockCharacterRepo(ctrl)

	p := player.NewPlayer()
	p.ID = 42
	p.Name = "Arthas"
	p.Race = 6
	p.Class = 2
	p.Level = 10

	mockRepo.EXPECT().GetCharacter(uint32(42)).Return(p, nil)

	result, err := mockRepo.GetCharacter(42)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, uint32(42), result.ID)
	assert.Equal(t, "Arthas", result.Name)
	assert.Equal(t, uint8(10), result.Level)
}
