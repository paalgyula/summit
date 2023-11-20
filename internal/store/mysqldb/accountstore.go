package mysqldb

import "github.com/paalgyula/summit/pkg/store"

type AccountStore struct{}

func (store *AccountStore) FindAccount(name string) *store.Account {
	panic("not implemented") // TODO: Implement
}
func (store *AccountStore) CreateAccount(name string, password string) (*store.Account, error) {
	panic("not implemented") // TODO: Implement
}

