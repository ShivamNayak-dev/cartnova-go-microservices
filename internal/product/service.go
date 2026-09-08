package product

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrInvalidName = errors.New("product name is required")
var ErrInvalidPrice = errors.New("price must be zero or greater")

type Service struct {
	repository *Repository
	redis      *redis.Client
}

func NewService(repository *Repository, redisClient *redis.Client) *Service {
	return &Service{repository: repository, redis: redisClient}
}

func (s *Service) Create(ctx context.Context, request CreateProductRequest) (*Product, error) {
	if strings.TrimSpace(request.Name) == "" {
		return nil, ErrInvalidName
	}
	if request.Price < 0 {
		return nil, ErrInvalidPrice
	}

	product := &Product{
		Name:        strings.TrimSpace(request.Name),
		Description: strings.TrimSpace(request.Description),
		Price:       request.Price,
		CategoryID:  request.CategoryID,
		Status:      "ACTIVE",
	}

	if err := s.repository.Create(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Product, error) {
	key := fmt.Sprintf("product:%d", id)
	if s.redis != nil {
		if data, err := s.redis.Get(ctx, key).Bytes(); err == nil {
			var product Product
			if err := jsonUnmarshal(data, &product); err == nil {
				return &product, nil
			}
		}
	}

	product, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.redis != nil {
		if data, err := jsonMarshal(product); err == nil {
			_ = s.redis.Set(ctx, key, data, 5*time.Minute).Err()
		}
	}

	return product, nil
}

func (s *Service) GetAll(ctx context.Context, categoryID *int64, search string) ([]Product, error) {
	return s.repository.FindAll(ctx, categoryID, search)
}

func (s *Service) Update(ctx context.Context, id int64, request UpdateProductRequest) (*Product, error) {
	if strings.TrimSpace(request.Name) == "" {
		return nil, ErrInvalidName
	}
	if request.Price < 0 {
		return nil, ErrInvalidPrice
	}
	if request.Status == "" {
		request.Status = "ACTIVE"
	}

	product := &Product{
		Name:        strings.TrimSpace(request.Name),
		Description: strings.TrimSpace(request.Description),
		Price:       request.Price,
		CategoryID:  request.CategoryID,
		Status:      strings.ToUpper(strings.TrimSpace(request.Status)),
	}

	if err := s.repository.Update(ctx, id, product); err != nil {
		return nil, err
	}

	s.invalidateCache(ctx, id)
	return s.repository.FindByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateCache(ctx, id)
	return nil
}

func (s *Service) invalidateCache(ctx context.Context, id int64) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, "product:"+strconv.FormatInt(id, 10)).Err()
	}
}
