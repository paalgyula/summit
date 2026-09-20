package auth

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/wow/crypt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrAccountCreateError   = errors.New("can't create account")
)

// ManagementService service provided.
type ManagementService interface {
	// Registers the user. Returns error if user already exists,
	// or email is alreay used, or sg.
	Register(user, pass, email string) error

	// FindAccount finds an account in the store.
	FindAccount(user string) *store.Account

	// GetSession returns the auth session if any.
	GetSession(user string) *Session

	// AddSession adds session to the auth session store.
	AddSession(session *Session)

	// UpdateRealm registers or refreshes a world server in the realm list.
	// World servers call it periodically; the returned duration is the
	// heartbeat deadline.
	UpdateRealm(status *Realm) (time.Duration, error)
}

// NewManagementService initializes account manager.
func NewManagementService(store store.AccountRepo) *ManagementServiceImpl {
	return &ManagementServiceImpl{
		store:    store,
		sessions: make(map[string]*Session),
		realms:   NewRealmRegistry(nil, DefaultRealmTTL),

		log: log.With().Str("service", "management").Logger(),
	}
}

type ManagementServiceImpl struct {
	store    store.AccountRepo
	sessions map[string]*Session
	mu       sync.RWMutex
	realms   *RealmRegistry

	log zerolog.Logger
}

// SetRealmRegistry replaces the registry world servers report into (e.g. one
// seeded with statically configured realms).
func (ms *ManagementServiceImpl) SetRealmRegistry(rr *RealmRegistry) {
	ms.realms = rr
}

// RealmRegistry returns the registry, which doubles as the RealmProvider.
func (ms *ManagementServiceImpl) RealmRegistry() *RealmRegistry {
	return ms.realms
}

// UpdateRealm records a world server's status report.
func (ms *ManagementServiceImpl) UpdateRealm(status *Realm) (time.Duration, error) {
	ms.realms.Update(status)
	return ms.realms.TTL(), nil
}

// Register tries to register an account on the auth server if it does not exists already.
// In this case ErrAccountAlreadyExists error will be returned.
func (ms *ManagementServiceImpl) Register(user string, pass string, email string) error {
	if acc := ms.store.FindAccount(user); acc != nil {
		return ErrAccountAlreadyExists
	}

	// TODO: check username and email
	// TODO: check password strength

	pwcrypt := crypt.NewWoWSRP6()
	salt := pwcrypt.RandomSalt()
	// The client proves UPPER(user):UPPER(pass); derive the verifier the same way
	verifier := pwcrypt.GenerateVerifier(strings.ToUpper(user), strings.ToUpper(pass), salt)

	acc, err := store.AccountFromCreds(user, salt.Text(16), verifier.Text(16))
	if err != nil {
		return fmt.Errorf("unknown error: %w", err)
	}

	acc.Email = email
	acc.CreatedAt = time.Now()

	// TODO: implement email activation flow
	acc.Activated = true

	if err := ms.store.CreateAccount(acc); err != nil {
		return fmt.Errorf("account persist: %w", err)
	}

	log.Info().Str("acc", user).Msg("account [%s] has been registered")

	return nil
}

// FindAccount retrives account from the database.
func (ms *ManagementServiceImpl) FindAccount(user string) *store.Account {
	return ms.store.FindAccount(user)
}

// GetSession returns the auth session if any.
func (ms *ManagementServiceImpl) GetSession(user string) *Session {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return ms.sessions[strings.ToLower(user)]
}

// AddSession adds session to the auth session store.
func (ms *ManagementServiceImpl) AddSession(session *Session) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.sessions[strings.ToLower(session.AccountName)] = session
}
