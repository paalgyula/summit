package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	authv1 "github.com/paalgyula/summit/pkg/pb/proto/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// managementTokenHeader carries the shared secret world servers present.
const managementTokenHeader = "authorization"

type managementRPCServer struct {
	// Must embed it because of the grpc generated interface.
	authv1.UnimplementedAuthManagementServer

	srv ManagementService
}

func (ms *managementRPCServer) Regiester(_ context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	err := ms.srv.Register(req.Username, req.Password, req.Email)

	resp := &authv1.RegisterResponse{Status: authv1.RegistrationStatus_SUCCESS}
	switch {
	case err == nil:
	case errors.Is(err, ErrAccountAlreadyExists):
		resp.Status = authv1.RegistrationStatus_EMAIL_ALREADY_EXISTS
		resp.ErrorMessage = err.Error()
	default:
		resp.Status = authv1.RegistrationStatus_OTHER_ERROR
		resp.ErrorMessage = err.Error()
	}

	return resp, nil
}

func (ms *managementRPCServer) GetSession(_ context.Context, req *authv1.GetSessionRequest) (*authv1.GetSessionResponse, error) {
	sess := ms.srv.GetSession(req.GetUsername())
	if sess == nil {
		return &authv1.GetSessionResponse{Found: false}, nil
	}

	return &authv1.GetSessionResponse{Found: true, SessionKey: sess.SessionKey}, nil
}

func (ms *managementRPCServer) UpdateRealm(_ context.Context, req *authv1.UpdateRealmRequest) (*authv1.UpdateRealmResponse, error) {
	st := req.GetStatus()
	if st == nil || st.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "realm name is required")
	}

	ttl, err := ms.srv.UpdateRealm(realmFromStatus(st))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &authv1.UpdateRealmResponse{HeartbeatSeconds: uint32(ttl.Seconds())}, nil
}

// realmFromStatus converts a reported status into a realm list entry.
func realmFromStatus(st *authv1.RealmStatus) *Realm {
	r := &Realm{
		Icon:          uint8(st.GetIcon()),
		Name:          st.GetName(),
		Slug:          st.GetSlug(),
		Address:       st.GetAddress(),
		OnlinePlayers: st.GetOnlinePlayers(),
		Timezone:      uint8(st.GetTimezone()),
	}
	if st.GetLocked() {
		r.Lock = 1
	}
	if st.GetMaxPlayers() > 0 {
		r.Population = float32(st.GetOnlinePlayers()) / float32(st.GetMaxPlayers())
	}
	return r
}

// managementAuthInterceptor rejects calls without the configured shared token.
// An empty token disables the check (local development only).
func managementAuthInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if token == "" {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		presented := ""
		if vals := md.Get(managementTokenHeader); len(vals) > 0 {
			presented = strings.TrimPrefix(vals[0], "Bearer ")
		}
		if subtle.ConstantTimeCompare([]byte(presented), []byte(token)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid management token")
		}
		return handler(ctx, req)
	}
}
