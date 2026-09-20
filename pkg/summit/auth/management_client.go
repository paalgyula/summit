package auth

import (
	"context"
	"fmt"
	"time"

	authv1 "github.com/paalgyula/summit/pkg/pb/proto/auth/v1"
	"github.com/paalgyula/summit/pkg/store"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type ManagementClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthManagementClient
	token  string

	// RequestTimeout timeout for requests. Default is 5 seconds
	RequestTimeout time.Duration
}

// NewManagementClient initializes new management client with gRPC protocol
// to interact with the auth server. token is the shared management secret.
func NewManagementClient(addr, token string) (*ManagementClient, error) {
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("auth.NewClient: %w", err)
	}

	client := new(ManagementClient)
	client.RequestTimeout = time.Second * 5
	client.client = authv1.NewAuthManagementClient(conn)
	client.conn = conn
	client.token = token

	return client, nil
}

// context returns a request context carrying the management token.
func (mc *ManagementClient) context() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), mc.RequestTimeout)
	if mc.token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, managementTokenHeader, "Bearer "+mc.token)
	}
	return ctx, cancel
}

// Close closes the management client connection.
//
//nolint:wrapcheck
func (mc *ManagementClient) Close() error {
	return mc.conn.Close()
}

// Register registers a new user in auth server.
func (mc *ManagementClient) Register(user, pass, email string) error {
	ctx, cancel := mc.context()
	defer cancel()

	res, err := mc.client.Regiester(ctx, &authv1.RegisterRequest{
		Username: user,
		Password: pass,
		Email:    email,
	})
	if err != nil {
		return fmt.Errorf("management.Register: %w", err)
	}

	switch res.GetStatus() {
	case authv1.RegistrationStatus_SUCCESS:
		return nil
	case authv1.RegistrationStatus_EMAIL_ALREADY_EXISTS:
		return ErrAccountAlreadyExists
	case authv1.RegistrationStatus_INVALID_USERNAME,
		authv1.RegistrationStatus_WEAK_PASSWORD,
		authv1.RegistrationStatus_OTHER_ERROR:
		fallthrough
	default:
		return fmt.Errorf("%w: %s", ErrAccountCreateError, res.GetErrorMessage())
	}
}

// FindAccount is not available remotely: world servers read accounts from
// their own store, and the management API deliberately exposes no account data.
func (mc *ManagementClient) FindAccount(_ string) *store.Account {
	return nil
}

// GetSession returns the auth session if any. Errors (auth server down,
// bad token) yield no session, i.e. the login is refused.
func (mc *ManagementClient) GetSession(user string) *Session {
	ctx, cancel := mc.context()
	defer cancel()

	res, err := mc.client.GetSession(ctx, &authv1.GetSessionRequest{
		Username: user,
	})
	if err != nil {
		log.Warn().Err(err).Str("account", user).Msg("management: session lookup failed")
		return nil
	}

	if res.GetFound() {
		return &Session{
			AccountName: user,
			SessionKey:  res.GetSessionKey(),
		}
	}

	return nil
}

// AddSession is only meaningful on the auth server itself.
func (mc *ManagementClient) AddSession(_ *Session) {}

// UpdateRealm reports this world server's status to the auth server.
func (mc *ManagementClient) UpdateRealm(r *Realm) (time.Duration, error) {
	ctx, cancel := mc.context()
	defer cancel()

	res, err := mc.client.UpdateRealm(ctx, &authv1.UpdateRealmRequest{Status: &authv1.RealmStatus{
		Name:          r.Name,
		Slug:          r.URLSlug(),
		Address:       r.Address,
		OnlinePlayers: r.OnlinePlayers,
		MaxPlayers:    r.MaxPlayers,
		Locked:        r.Lock != 0,
		Icon:          uint32(r.Icon),
		Timezone:      uint32(r.Timezone),
	}})
	if err != nil {
		return 0, fmt.Errorf("management.UpdateRealm: %w", err)
	}

	return time.Duration(res.GetHeartbeatSeconds()) * time.Second, nil
}
