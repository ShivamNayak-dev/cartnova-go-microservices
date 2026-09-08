package user

import (
	"context"
	"strconv"

	"github.com/ShivamNayak-dev/cartnova/internal/grpcapi"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCServer struct {
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) GetUser(ctx context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	idValue, ok := request.Fields["id"]
	if !ok {
		return nil, ErrNotFound
	}

	id := int64(idValue.GetNumberValue())
	if id <= 0 {
		parsed, err := strconv.ParseInt(idValue.GetStringValue(), 10, 64)
		if err != nil {
			return nil, ErrNotFound
		}
		id = parsed
	}

	user, err := s.service.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return structpb.NewStruct(map[string]any{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}
