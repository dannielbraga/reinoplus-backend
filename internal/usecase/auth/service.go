package auth

import (
	"context"
	"strings"
	"time"

	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/textutil"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
}

type RegisterRepository interface {
	RegisterUserWithMember(ctx context.Context, input domain.RegisterUserInput) (domain.User, domain.Member, error)
	UpdateUserProfile(ctx context.Context, user domain.User, member domain.Member, passwordHash *string) (domain.User, domain.Member, error)
	GetMemberByID(ctx context.Context, id string) (domain.Member, error)
}

type TokenService interface {
	GenerateAccessToken(user domain.User) (string, error)
	GenerateRefreshToken(user domain.User) (string, error)
	ParseRefreshToken(token string) (userID string, err error)
}

type Service struct {
	users    UserRepository
	register RegisterRepository
	tokens   TokenService
}

func NewService(users UserRepository, register RegisterRepository, tokens TokenService) *Service {
	return &Service{users: users, register: register, tokens: tokens}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	User         domain.User
}

func (s *Service) Login(ctx context.Context, input LoginInput, verifyPassword func(hash, password string) bool) (LoginOutput, error) {
	if input.Email == "" || input.Password == "" {
		return LoginOutput{}, domain.ErrValidation
	}

	user, err := s.users.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(input.Email)))
	if err != nil {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	if !verifyPassword(user.PasswordHash, input.Password) {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	accessToken, err := s.tokens.GenerateAccessToken(user)
	if err != nil {
		return LoginOutput{}, err
	}

	refreshToken, err := s.tokens.GenerateRefreshToken(user)
	if err != nil {
		return LoginOutput{}, err
	}

	return LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

type RegisterInput struct {
	Name      string
	Email     string
	Password  string
	Phone     string
	BirthDate time.Time
	Address   string
}

func (s *Service) Register(ctx context.Context, input RegisterInput, hashPassword func(password string) (string, error)) (LoginOutput, error) {
	if err := validateRegisterInput(input); err != nil {
		return LoginOutput{}, err
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return LoginOutput{}, err
	}

	user, _, err := s.register.RegisterUserWithMember(ctx, domain.RegisterUserInput{
		Name:         textutil.TitleCaseName(input.Name),
		Email:        strings.TrimSpace(strings.ToLower(input.Email)),
		PasswordHash: passwordHash,
		Phone:        normalizePhone(input.Phone),
		BirthDate:    input.BirthDate,
		Address:      strings.TrimSpace(input.Address),
	})
	if err != nil {
		return LoginOutput{}, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(user)
	if err != nil {
		return LoginOutput{}, err
	}

	refreshToken, err := s.tokens.GenerateRefreshToken(user)
	if err != nil {
		return LoginOutput{}, err
	}

	return LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

type UpdateProfileInput struct {
	UserID   string
	Name     string
	Email    string
	Password string
	Phone    string
	BirthDate time.Time
	Address  string
}

type ProfileOutput struct {
	User   domain.User
	Member domain.Member
}

func (s *Service) UpdateProfile(ctx context.Context, input UpdateProfileInput, hashPassword func(password string) (string, error)) (ProfileOutput, error) {
	if input.UserID == "" {
		return ProfileOutput{}, domain.ErrValidation
	}
	if err := validateProfileInput(input); err != nil {
		return ProfileOutput{}, err
	}

	user, err := s.users.GetByID(ctx, input.UserID)
	if err != nil {
		return ProfileOutput{}, err
	}

	member := domain.Member{}
	if user.MemberID != nil {
		member, err = s.register.GetMemberByID(ctx, *user.MemberID)
		if err != nil {
			return ProfileOutput{}, err
		}
	}

	user.Name = textutil.TitleCaseName(input.Name)
	user.Email = strings.TrimSpace(strings.ToLower(input.Email))
	member.Name = user.Name
	member.Phone = normalizePhone(input.Phone)
	member.BirthDate = input.BirthDate
	member.Address = strings.TrimSpace(input.Address)

	var passwordHash *string
	if strings.TrimSpace(input.Password) != "" {
		if len(input.Password) < 6 {
			return ProfileOutput{}, domain.ErrValidation
		}
		hash, err := hashPassword(input.Password)
		if err != nil {
			return ProfileOutput{}, err
		}
		passwordHash = &hash
	}

	updatedUser, updatedMember, err := s.register.UpdateUserProfile(ctx, user, member, passwordHash)
	if err != nil {
		return ProfileOutput{}, err
	}

	return ProfileOutput{User: updatedUser, Member: updatedMember}, nil
}

func (s *Service) GetProfile(ctx context.Context, userID string) (ProfileOutput, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return ProfileOutput{}, err
	}

	output := ProfileOutput{User: user}
	if user.MemberID != nil {
		member, err := s.register.GetMemberByID(ctx, *user.MemberID)
		if err != nil {
			return ProfileOutput{}, err
		}
		output.Member = member
	}

	return output, nil
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (RefreshOutput, error) {
	if refreshToken == "" {
		return RefreshOutput{}, domain.ErrInvalidToken
	}

	userID, err := s.tokens.ParseRefreshToken(refreshToken)
	if err != nil {
		return RefreshOutput{}, domain.ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return RefreshOutput{}, domain.ErrInvalidToken
	}

	accessToken, err := s.tokens.GenerateAccessToken(user)
	if err != nil {
		return RefreshOutput{}, err
	}

	newRefreshToken, err := s.tokens.GenerateRefreshToken(user)
	if err != nil {
		return RefreshOutput{}, err
	}

	return RefreshOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *Service) Me(ctx context.Context, userID string) (domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

func validateRegisterInput(input RegisterInput) error {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Address) == "" {
		return domain.ErrValidation
	}
	if !strings.Contains(input.Email, "@") {
		return domain.ErrValidation
	}
	if len(input.Password) < 6 {
		return domain.ErrValidation
	}
	if len(normalizePhone(input.Phone)) < 10 {
		return domain.ErrValidation
	}
	if input.BirthDate.IsZero() {
		return domain.ErrValidation
	}
	return nil
}

func validateProfileInput(input UpdateProfileInput) error {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Address) == "" {
		return domain.ErrValidation
	}
	if !strings.Contains(input.Email, "@") {
		return domain.ErrValidation
	}
	if len(normalizePhone(input.Phone)) < 10 {
		return domain.ErrValidation
	}
	if input.BirthDate.IsZero() {
		return domain.ErrValidation
	}
	return nil
}

func normalizePhone(phone string) string {
	var digits strings.Builder
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}
	return digits.String()
}

func NowUTC() time.Time {
	return time.Now().UTC()
}
