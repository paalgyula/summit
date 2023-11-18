package auth

import (
	"context"
	"fmt"
	"time"

	authv1 "github.com/paalgyula/summit/pkg/pb/proto/auth/v1"
	"github.com/paalgyula/summit/pkg/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ManagementClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthManagementClient

	// RequestTimeout timeout for requests. Default is 5 seconds
	RequestTimeout time.Duration
}

// NewManagementClient initializes new management client with gRPC protocol
// to interact with the auth server.
func NewManagementClient(addr string) (*ManagementClient, error) {
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("auth.NewClient: %w", err)
	}

	client := new(ManagementClient)
	client.RequestTimeout = time.Second * 5
	client.client = authv1.NewAuthManagementClient(conn)
	client.conn = conn

	return client, nil
}

// Close closes the management client connection.
//
//nolint:wrapcheck
func (mc *ManagementClient) Close() error {
	return mc.conn.Close()
}

// Register registers a new user in auth server.
func (mc *ManagementClient) Register(user, pass, email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), mc.RequestTimeout)
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

// FindAccount finds an account in the store.
func (mc *ManagementClient) FindAccount(user string) *store.Account {
	panic("not implemented") // TODO: Implement
}

// GetSession returns the auth session if any.
func (mc *ManagementClient) GetSession(user string) *Session {
	ctx, cancel := context.WithTimeout(context.Background(), mc.RequestTimeout)
	defer cancel()

	// TODO: is this a good idea to ignore the error?
	res, _ := mc.client.GetSession(ctx, &authv1.GetSessionRequest{
		Username: user,
	})

	if res.GetFound() {
		return &Session{
			AccountName: user,
			SessionKey:  res.GetSessionKey(),
		}
	}

	return nil
}

// AddSession adds session to the auth session store.
func (mc *ManagementClient) AddSession(session *Session) {
	panic("not implemented") // TODO: Implement
}
