package campaign

import (
	"context"
	"strings"
	"time"

	"github.com/reinoplus/reinoplus/internal/domain"
)

type Repository interface {
	List(ctx context.Context) ([]domain.Campaign, error)
	GetByID(ctx context.Context, id string) (domain.Campaign, error)
	Create(ctx context.Context, campaign domain.Campaign) (domain.Campaign, error)
	ListContributions(ctx context.Context, campaignID string) ([]domain.Contribution, error)
	GetRaisedAmount(ctx context.Context, campaignID string) (float64, error)
}

type ContributionWriter interface {
	CreateContribution(ctx context.Context, contribution domain.Contribution) (domain.Contribution, error)
}

type MemberLookup interface {
	GetByPhone(ctx context.Context, phone string) (*domain.Member, error)
}

type Service struct {
	repo         Repository
	writer       ContributionWriter
	memberLookup MemberLookup
}

func NewService(repo Repository, writer ContributionWriter, memberLookup MemberLookup) *Service {
	return &Service{repo: repo, writer: writer, memberLookup: memberLookup}
}

type CreateInput struct {
	Name        string
	Description string
	GoalAmount  float64
	StartDate   time.Time
	EndDate     time.Time
}

type CreateContributionInput struct {
	CampaignID       string
	ContributorName  string
	ContributorPhone string
	Amount           float64
	PaymentMethod    domain.PaymentMethod
	ContributedAt    time.Time
	CreatedByUserID  string
}

type CampaignDetail struct {
	Campaign domain.Campaign
	Raised   float64
}

func (s *Service) List(ctx context.Context) ([]domain.Campaign, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (CampaignDetail, error) {
	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}

	raised, err := s.repo.GetRaisedAmount(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}

	return CampaignDetail{Campaign: campaign, Raised: raised}, nil
}

func (s *Service) ListContributions(ctx context.Context, campaignID string) ([]domain.Contribution, error) {
	if campaignID == "" {
		return nil, domain.ErrValidation
	}
	return s.repo.ListContributions(ctx, campaignID)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Campaign, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" {
		return domain.Campaign{}, domain.ErrValidation
	}
	if input.GoalAmount <= 0 {
		return domain.Campaign{}, domain.ErrValidation
	}
	if input.EndDate.Before(input.StartDate) {
		return domain.Campaign{}, domain.ErrValidation
	}

	campaign := domain.Campaign{
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		GoalAmount:  input.GoalAmount,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		Status:      domain.CampaignStatusActive,
	}

	return s.repo.Create(ctx, campaign)
}

func (s *Service) CreateContribution(ctx context.Context, input CreateContributionInput) (domain.Contribution, error) {
	phone := normalizePhone(input.ContributorPhone)
	name := strings.TrimSpace(input.ContributorName)

	if input.CampaignID == "" || phone == "" {
		return domain.Contribution{}, domain.ErrValidation
	}
	if name == "" {
		return domain.Contribution{}, domain.ErrValidation
	}
	if input.Amount <= 0 {
		return domain.Contribution{}, domain.ErrValidation
	}
	if input.ContributedAt.IsZero() {
		return domain.Contribution{}, domain.ErrValidation
	}
	if input.CreatedByUserID == "" {
		return domain.Contribution{}, domain.ErrValidation
	}
	if !isValidContributionPaymentMethod(input.PaymentMethod) {
		return domain.Contribution{}, domain.ErrValidation
	}

	campaign, err := s.repo.GetByID(ctx, input.CampaignID)
	if err != nil {
		return domain.Contribution{}, err
	}
	if campaign.Status != domain.CampaignStatusActive {
		return domain.Contribution{}, domain.ErrCampaignNotActive
	}

	contribution := domain.Contribution{
		CampaignID:       input.CampaignID,
		ContributorName:  name,
		ContributorPhone: phone,
		Amount:           input.Amount,
		PaymentMethod:    input.PaymentMethod,
		ContributedAt:    input.ContributedAt,
		CreatedByUserID:  &input.CreatedByUserID,
	}

	if s.memberLookup != nil {
		if member, err := s.memberLookup.GetByPhone(ctx, phone); err == nil && member != nil {
			contribution.MemberID = &member.ID
			contribution.ContributorName = member.Name
		}
	}

	return s.writer.CreateContribution(ctx, contribution)
}

func isValidContributionPaymentMethod(method domain.PaymentMethod) bool {
	switch method {
	case domain.PaymentMethodCash, domain.PaymentMethodPix:
		return true
	default:
		return false
	}
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
