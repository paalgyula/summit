package auth

import (
	"context"
	"fmt"
	"time"

	authv1 "github.com/paalgyula/summit/pkg/pb/proto/auth/v1"
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

	switch res.Status {
	case authv1.RegistrationStatus_SUCCESS:
		return nil
	case authv1.RegistrationStatus_EMAIL_ALREADY_EXISTS:
		return ErrAccountAlreadyExists
	default:
		return fmt.Errorf("%w: %s", ErrAccountCreateError, res.ErrorMessage)
	}
}
