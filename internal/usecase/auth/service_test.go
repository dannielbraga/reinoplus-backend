package auth_test

import (
	"context"
	"testing"

	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/usecase/auth"
)

type mockUserRepo struct {
	user domain.User
	err  error
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if m.err != nil {
		return domain.User{}, m.err
	}
	return m.user, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (domain.User, error) {
	if m.err != nil {
		return domain.User{}, m.err
	}
	return m.user, nil
}

type mockTokenService struct{}

func (m *mockTokenService) GenerateAccessToken(user domain.User) (string, error) {
	return "access-token", nil
}

func (m *mockTokenService) GenerateRefreshToken(user domain.User) (string, error) {
	return "refresh-token", nil
}

func (m *mockTokenService) ParseRefreshToken(token string) (string, error) {
	return "user-1", nil
}

type mockRegisterRepo struct{}

func (m *mockRegisterRepo) RegisterUserWithMember(ctx context.Context, input domain.RegisterUserInput) (domain.User, domain.Member, error) {
	return domain.User{}, domain.Member{}, nil
}

func (m *mockRegisterRepo) UpdateUserProfile(ctx context.Context, user domain.User, member domain.Member, passwordHash *string) (domain.User, domain.Member, error) {
	return user, member, nil
}

func (m *mockRegisterRepo) GetMemberByID(ctx context.Context, id string) (domain.Member, error) {
	return domain.Member{}, nil
}

func TestLoginInvalidCredentials(t *testing.T) {
	service := auth.NewService(&mockUserRepo{
		user: domain.User{ID: "user-1", Email: "admin@reinoplus.local", PasswordHash: "hash"},
	}, &mockRegisterRepo{}, &mockTokenService{})

	_, err := service.Login(context.Background(), auth.LoginInput{
		Email: "admin@reinoplus.local", Password: "wrong",
	}, func(hash, password string) bool { return false })
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginSuccess(t *testing.T) {
	service := auth.NewService(&mockUserRepo{
		user: domain.User{ID: "user-1", Email: "admin@reinoplus.local", PasswordHash: "hash", Name: "Admin"},
	}, &mockRegisterRepo{}, &mockTokenService{})

	output, err := service.Login(context.Background(), auth.LoginInput{
		Email: "admin@reinoplus.local", Password: "admin123",
	}, func(hash, password string) bool { return true })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.AccessToken != "access-token" || output.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected tokens: %+v", output)
	}
}
