package product

import (
	"context"

	"github.com/ShivamNayak-dev/cartnova/internal/grpcapi"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCServer struct {
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) GetProduct(ctx context.Context, request *structpb.Struct) (*structpb.Struct, error) {
	idValue, ok := request.Fields["id"]
	if !ok {
		return nil, ErrNotFound
	}

	id := int64(idValue.GetNumberValue())
	if id <= 0 {
		return nil, ErrNotFound
	}

	product, err := s.service.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return structpb.NewStruct(map[string]any{
		"id":          product.ID,
		"name":        product.Name,
		"description": product.Description,
		"price":       product.Price,
		"status":      product.Status,
	})
}

var _ grpcapi.ProductServiceServer = (*GRPCServer)(nil)
