package service

import (
	"context"
	"errors"
	"wishlist-api/internal/domain"
	"wishlist-api/internal/jwt"
	"wishlist-api/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type AuthService struct {
	users *repository.UserRepository
	jwt   *jwt.Manager
}

func NewAuthService(users *repository.UserRepository, jwt *jwt.Manager) *AuthService {
	return &AuthService{users: users, jwt: jwt}
}

func (s *AuthService) Register(ctx context.Context, input domain.RegisterInput) (string, error) {
	existing, err := s.users.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return "", ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user := &domain.User{
		Email:        input.Email,
		PasswordHash: string(hash),
	}
	if err := s.users.CreateUser(ctx, user); err != nil {
		return "", err
	}

	return s.jwt.Generate(user.ID)
}

func (s *AuthService) Login(ctx context.Context, input domain.LoginInput) (string, error) {
	user, err := s.users.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return s.jwt.Generate(user.ID)
}
