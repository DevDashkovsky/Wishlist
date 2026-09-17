package service

import (
	"context"
	"errors"
	"wishlist-api/internal/domain"
	"wishlist-api/internal/jwt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = domain.ErrEmailTaken
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// dummyPasswordHash makes login timing independent of whether an email exists.
// It is a bcrypt hash of a value that is never accepted as a user password.
const dummyPasswordHash = "$2y$10$LvCAjez3FrIL63Ji3A0oouHLz1k0pPPaHHSu3d9os.YM2dhSytAEa"

type UserRepository interface {
	CreateUser(ctx context.Context, u *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
}

type AuthService struct {
	users UserRepository
	jwt   *jwt.Manager
}

func NewAuthService(users UserRepository, jwt *jwt.Manager) *AuthService {
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

	passwordHash := dummyPasswordHash
	if user != nil {
		passwordHash = user.PasswordHash
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); user == nil || err != nil {
		return "", ErrInvalidCredentials
	}

	return s.jwt.Generate(user.ID)
}
