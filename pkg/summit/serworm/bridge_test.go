package serworm_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	// t.Skip("Still failing needs more attention")

	db.GetInstance().Accounts = append(db.GetInstance().Accounts,
		&db.Account{
			Name: "TEST",
			S:    "9398c11e0e7128c7a56e3fde45b418744ffe9c7f41aaed48ac27e62d3700e223",
			V:    "3e3f49a5a14a43b870f8de5534e318c63394738c364a71f205a8ba277bb56ff6",
		})

	as, err := auth.NewServer("localhost:5000", &auth.StaticRealmProvider{})
	assert.NoError(t, err)
	defer as.Close()

	br := serworm.NewBridge(5001, "localhost:5000", "Test Realm", nil)

	assert.NotNil(t, br)
}
