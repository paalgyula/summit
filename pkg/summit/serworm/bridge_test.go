package serworm_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/paalgyula/summit/pkg/store/mock_store"
)

func TestConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := mock_store.NewMockAccountRepo(ctrl)

	ams := auth.NewManagementService(ms)
	as, err := auth.NewServer("127.0.0.1:5000", ams)
	assert.NoError(t, err)

	defer as.Close()

	br := serworm.NewWorldBridge(5001, "localhost:5000", "Test Realm", nil, "TEST", "abc123")

	assert.NotNil(t, br)
}
