package auth

import (
	"context"

	authv1 "github.com/paalgyula/summit/pkg/pb/proto/auth/v1"
)

type managementRPCServer struct {
	// Must embed it because of the grpc generated interface.
	authv1.UnimplementedAuthManagementServer

	srv ManagementService
}

func (ms *managementRPCServer) Regiester(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	err := ms.srv.Register(req.Username, req.Password, req.Email)

	status := authv1.RegistrationStatus_SUCCESS
	errorMsg := ""

	switch err {
	case ErrAccountAlreadyExists:
		status = authv1.RegistrationStatus_EMAIL_ALREADY_EXISTS
	}

	return &authv1.RegisterResponse{
		Status:       status,
		ErrorMessage: errorMsg,
	}, nil
}

func (ms *managementRPCServer) GetSession(ctx context.Context, req *authv1.GetSessionRequest) (*authv1.GetSessionResponse, error) {
	// ms.srv.

	return nil, nil
}
