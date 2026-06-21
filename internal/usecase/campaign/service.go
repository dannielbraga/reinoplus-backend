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
	Update(ctx context.Context, id string, campaign domain.Campaign) (domain.Campaign, error)
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, status domain.CampaignStatus) (domain.Campaign, error)
	CountContributions(ctx context.Context, campaignID string) (int, error)
	ListContributions(ctx context.Context, campaignID string, search string) ([]domain.Contribution, error)
	GetCampaignTotals(ctx context.Context, campaignID string) (paid float64, promised float64, err error)
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
	Name               string
	Description        string
	GoalAmount         float64
	StartDate          time.Time
	EndDate            time.Time
	IsRecurring        bool
	RecurrenceInterval *domain.RecurrenceInterval
	DurationMonths     *int
}

type UpdateInput = CreateInput

type CreateContributionInput struct {
	CampaignID        string
	ContributorName   string
	ContributorPhone  string
	Amount            float64
	PaymentMethod     domain.PaymentMethod
	ContributedAt     time.Time
	IsPaid            bool
	InstallmentNumber *int
	CreatedByUserID   string
}

type CampaignDetail struct {
	Campaign domain.Campaign
	Raised   float64
	Promised float64
}

func (s *Service) List(ctx context.Context) ([]domain.Campaign, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (CampaignDetail, error) {
	campaign, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}

	paid, promised, err := s.repo.GetCampaignTotals(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}

	return CampaignDetail{Campaign: campaign, Raised: paid, Promised: promised}, nil
}

func (s *Service) ListContributions(ctx context.Context, campaignID string, search string) ([]domain.Contribution, error) {
	if campaignID == "" {
		return nil, domain.ErrValidation
	}
	return s.repo.ListContributions(ctx, campaignID, strings.TrimSpace(search))
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (domain.Campaign, error) {
	if id == "" {
		return domain.Campaign{}, domain.ErrValidation
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if existing.Status == domain.CampaignStatusCompleted {
		return domain.Campaign{}, domain.ErrValidation
	}

	campaign, err := buildCampaignFromInput(input)
	if err != nil {
		return domain.Campaign{}, err
	}
	campaign.Status = existing.Status
	campaign.CreatedAt = existing.CreatedAt

	return s.repo.Update(ctx, id, campaign)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return domain.ErrValidation
	}

	count, err := s.repo.CountContributions(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrCampaignHasContributions
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) SetStatus(ctx context.Context, id string, active bool) (domain.Campaign, error) {
	if id == "" {
		return domain.Campaign{}, domain.ErrValidation
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if existing.Status == domain.CampaignStatusCompleted {
		return domain.Campaign{}, domain.ErrValidation
	}

	status := domain.CampaignStatusCancelled
	if active {
		status = domain.CampaignStatusActive
	}

	return s.repo.SetStatus(ctx, id, status)
}

func buildCampaignFromInput(input CreateInput) (domain.Campaign, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" {
		return domain.Campaign{}, domain.ErrValidation
	}
	if input.GoalAmount <= 0 {
		return domain.Campaign{}, domain.ErrValidation
	}
	if input.StartDate.IsZero() {
		return domain.Campaign{}, domain.ErrValidation
	}

	endDate := input.EndDate
	var recurrenceInterval *domain.RecurrenceInterval
	var durationMonths *int

	if input.IsRecurring {
		if input.DurationMonths == nil || *input.DurationMonths <= 0 {
			return domain.Campaign{}, domain.ErrValidation
		}
		if input.RecurrenceInterval == nil || !domain.IsValidRecurrenceInterval(*input.RecurrenceInterval) {
			return domain.Campaign{}, domain.ErrValidation
		}
		endDate = input.StartDate.AddDate(0, *input.DurationMonths, 0)
		recurrenceInterval = input.RecurrenceInterval
		durationMonths = input.DurationMonths
	} else if endDate.IsZero() || endDate.Before(input.StartDate) {
		return domain.Campaign{}, domain.ErrValidation
	}

	return domain.Campaign{
		Name:               strings.TrimSpace(input.Name),
		Description:        strings.TrimSpace(input.Description),
		GoalAmount:         input.GoalAmount,
		StartDate:          input.StartDate,
		EndDate:            endDate,
		IsRecurring:        input.IsRecurring,
		RecurrenceInterval: recurrenceInterval,
		DurationMonths:     durationMonths,
	}, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Campaign, error) {
	campaign, err := buildCampaignFromInput(input)
	if err != nil {
		return domain.Campaign{}, err
	}
	campaign.Status = domain.CampaignStatusActive

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

	if campaign.IsRecurring {
		if input.InstallmentNumber == nil || *input.InstallmentNumber <= 0 {
			return domain.Contribution{}, domain.ErrValidation
		}
		if campaign.RecurrenceInterval == nil || campaign.DurationMonths == nil {
			return domain.Contribution{}, domain.ErrValidation
		}
		totalInstallments := domain.TotalInstallments(*campaign.DurationMonths, *campaign.RecurrenceInterval)
		if *input.InstallmentNumber > totalInstallments {
			return domain.Contribution{}, domain.ErrValidation
		}
	} else if input.InstallmentNumber != nil {
		return domain.Contribution{}, domain.ErrValidation
	}

	contribution := domain.Contribution{
		CampaignID:        input.CampaignID,
		ContributorName:   name,
		ContributorPhone:  phone,
		Amount:            input.Amount,
		PaymentMethod:     input.PaymentMethod,
		ContributedAt:     input.ContributedAt,
		IsPaid:            input.IsPaid,
		InstallmentNumber: input.InstallmentNumber,
		CreatedByUserID:   &input.CreatedByUserID,
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
