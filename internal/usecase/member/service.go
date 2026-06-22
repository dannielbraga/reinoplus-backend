package member

import (
	"context"
	"strings"
	"time"

	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/textutil"
)

type Repository interface {
	List(ctx context.Context, search string, limit, offset int) ([]domain.Member, error)
	GetByID(ctx context.Context, id string) (domain.Member, error)
	GetByPhone(ctx context.Context, phone string) (*domain.Member, error)
	Create(ctx context.Context, member domain.Member) (domain.Member, error)
	Update(ctx context.Context, member domain.Member) (domain.Member, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	Name      string
	Phone     string
	BirthDate time.Time
	Address   string
}

type UpdateInput struct {
	ID        string
	Name      string
	Phone     string
	BirthDate time.Time
	Address   string
}

func (s *Service) List(ctx context.Context, search string, limit, offset int) ([]domain.Member, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.List(ctx, strings.TrimSpace(search), limit, offset)
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.Member, error) {
	if id == "" {
		return domain.Member{}, domain.ErrValidation
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) SearchByPhone(ctx context.Context, phone string) (*domain.Member, error) {
	normalized := normalizePhone(phone)
	if len(normalized) < 10 {
		return nil, domain.ErrValidation
	}
	return s.repo.GetByPhone(ctx, normalized)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Member, error) {
	if err := validateInput(input.Name, input.Phone, input.Address, input.BirthDate); err != nil {
		return domain.Member{}, err
	}

	member := domain.Member{
		Name:      textutil.TitleCaseName(input.Name),
		Phone:     normalizePhone(input.Phone),
		BirthDate: input.BirthDate,
		Address:   strings.TrimSpace(input.Address),
	}

	return s.repo.Create(ctx, member)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (domain.Member, error) {
	if input.ID == "" {
		return domain.Member{}, domain.ErrValidation
	}
	if err := validateInput(input.Name, input.Phone, input.Address, input.BirthDate); err != nil {
		return domain.Member{}, err
	}

	member := domain.Member{
		ID:        input.ID,
		Name:      textutil.TitleCaseName(input.Name),
		Phone:     normalizePhone(input.Phone),
		BirthDate: input.BirthDate,
		Address:   strings.TrimSpace(input.Address),
	}

	return s.repo.Update(ctx, member)
}

func validateInput(name, phone, address string, birthDate time.Time) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(address) == "" {
		return domain.ErrValidation
	}
	if len(normalizePhone(phone)) < 10 {
		return domain.ErrValidation
	}
	if birthDate.IsZero() {
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
