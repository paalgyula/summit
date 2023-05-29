package serworm_test

import (
	"testing"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/serworm"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	t.Skip("Still failing needs more attention")

	db.GetInstance().Accounts = append(db.GetInstance().Accounts, &db.Account{Name: "TEST"})

	as, err := auth.NewServer("localhost:5000")
	assert.NoError(t, err)
	defer as.Close()

	br := serworm.NewBridge("localhost:5000", "TEST", "test")

	assert.NotNil(t, br)
}
