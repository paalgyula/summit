package auth

import (
	"errors"
	"fmt"
	"strings"
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
}

// NewManagementService initializes account manager.
func NewManagementService(store store.AccountRepo) *ManagementServiceImpl {
	return &ManagementServiceImpl{
		store:    store,
		sessions: make(map[string]*Session),

		log: log.With().Str("service", "management").Logger(),
	}
}

type ManagementServiceImpl struct {
	store    store.AccountRepo
	sessions map[string]*Session

	log zerolog.Logger
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
	verifier := pwcrypt.GenerateVerifier(strings.ToUpper(user), pass, salt)

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
	user = strings.ToLower(user)

	return ms.sessions[user]
}

// AddSession adds session to the auth session store.
func (ms *ManagementServiceImpl) AddSession(session *Session) {
	ms.sessions[strings.ToLower(session.AccountName)] = session
}
