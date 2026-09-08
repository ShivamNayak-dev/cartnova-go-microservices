package user

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidName = errors.New("name is required")
var ErrInvalidEmail = errors.New("valid email is required")
var ErrInvalidPassword = errors.New("password must be at least 8 characters")
var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Register(ctx context.Context, request RegisterRequest) (*User, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))

	if request.Name == "" {
		return nil, ErrInvalidName
	}
	if !strings.Contains(request.Email, "@") {
		return nil, ErrInvalidEmail
	}
	if len(request.Password) < 8 {
		return nil, ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		Name:         request.Name,
		Email:        request.Email,
		PasswordHash: string(hash),
		Role:         "CUSTOMER",
	}

	if err := s.repository.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, request LoginRequest) (*User, error) {
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	user, err := s.repository.FindByEmail(ctx, request.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, id int64, request UpdateProfileRequest) (*User, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	if err := s.repository.UpdateName(ctx, id, name); err != nil {
		return nil, err
	}

	return s.repository.FindByID(ctx, id)
}
