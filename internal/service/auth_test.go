package service

import (
	"context"
	"errors"
	"testing"
	"time"
	"wishlist-api/internal/domain"
	"wishlist-api/internal/jwt"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	getUserByEmail func(ctx context.Context, email string) (*domain.User, error)
	createUser     func(ctx context.Context, u *domain.User) error
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.getUserByEmail(ctx, email)
}

func (m *mockUserRepo) CreateUser(ctx context.Context, u *domain.User) error {
	return m.createUser(ctx, u)
}

func TestRegister_Success(t *testing.T) {
	repo := &mockUserRepo{getUserByEmail: func(ctx context.Context, email string) (*domain.User, error) {
		return nil, nil
	},
		createUser: func(ctx context.Context, u *domain.User) error {
			u.ID = 1
			return nil
		},
	}

	svc := NewAuthService(repo, testJWTManager())

	token, err := svc.Register(context.Background(), domain.RegisterInput{
		Email:    "test@test.com",
		Password: "test1234",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestRegister_EmailTaken(t *testing.T) {
	repo := &mockUserRepo{
		getUserByEmail: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email}, nil
		},
	}

	svc := NewAuthService(repo, testJWTManager())

	_, err := svc.Register(context.Background(), domain.RegisterInput{
		Email:    "taken@taken.com",
		Password: "test1234",
	})

	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("got %v, want ErrEmailTaken", err)
	}
}

func TestLogin_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)

	repo := &mockUserRepo{
		getUserByEmail: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, PasswordHash: string(hash)}, nil
		},
	}

	svc := NewAuthService(repo, testJWTManager())

	token, err := svc.Login(context.Background(), domain.LoginInput{
		Email:    "test@test.com",
		Password: "secret123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)

	repo := &mockUserRepo{
		getUserByEmail: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: 1, Email: email, PasswordHash: string(hash)}, nil
		},
	}

	svc := NewAuthService(repo, testJWTManager())

	_, err := svc.Login(context.Background(), domain.LoginInput{
		Email:    "test@test.com",
		Password: "wrongpass",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{
		getUserByEmail: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil
		},
	}

	svc := NewAuthService(repo, testJWTManager())

	_, err := svc.Login(context.Background(), domain.LoginInput{
		Email:    "noone@test.com",
		Password: "secret123",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func testJWTManager() *jwt.Manager {
	return jwt.NewManager("test-secret", time.Hour)
}
