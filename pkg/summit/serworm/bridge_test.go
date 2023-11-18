package serworm_test

import (
	"os"
	"testing"

	"github.com/paalgyula/summit/pkg/store/localdb"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	store := localdb.InitYamlDatabase("test.yaml")
	defer os.Remove("test.yaml")

	store.Accounts = append(store.Accounts,
		&localdb.Account{
			Name:     "TEST",
			Salt:     "9398c11e0e7128c7a56e3fde45b418744ffe9c7f41aaed48ac27e62d3700e223",
			Verifier: "3e3f49a5a14a43b870f8de5534e318c63394738c364a71f205a8ba277bb56ff6",
		})

	ms := auth.NewManagementService(store)
	as, err := auth.NewServer("127.0.0.1:5000", ms)
	assert.NoError(t, err)

	defer as.Close()

	br := serworm.NewWorldBridge(5001, "localhost:5000", "Test Realm", nil)

	assert.NotNil(t, br)
}
