package raffle

import (
	"context"
	"strings"
	"time"

	"github.com/reinoplus/reinoplus/internal/domain"
)

type Repository interface {
	List(ctx context.Context) ([]domain.RaffleSummary, error)
	GetActive(ctx context.Context) (*domain.RaffleSummary, error)
	GetByID(ctx context.Context, id string) (domain.RaffleSummary, error)
	Create(ctx context.Context, raffle domain.Raffle, prizes []domain.RafflePrize, numbers []domain.RaffleNumber) (domain.RaffleSummary, error)
	Update(ctx context.Context, id string, raffle domain.Raffle, prizes []domain.RafflePrize) (domain.RaffleSummary, error)
	Delete(ctx context.Context, id string) error
	ListNumbers(ctx context.Context, raffleID string) ([]domain.RaffleNumber, error)
	CountSoldNumbers(ctx context.Context, raffleID string) (int, error)
}

type TransactionalRepository interface {
	SellNumber(ctx context.Context, raffleID string, number int, sale SellNumberInput) error
	SellNumbers(ctx context.Context, raffleID string, numbers []int, sale SellNumberInput) error
	Draw(ctx context.Context, raffleID string, winningNumber int) (domain.RaffleSummary, error)
}

type MemberLookup interface {
	GetByPhone(ctx context.Context, phone string) (*domain.Member, error)
}

type Service struct {
	repo         Repository
	tx           TransactionalRepository
	memberLookup MemberLookup
}

func NewService(repo Repository, tx TransactionalRepository, memberLookup MemberLookup) *Service {
	return &Service{repo: repo, tx: tx, memberLookup: memberLookup}
}

type CreateInput struct {
	Name         string
	GoalAmount   float64
	PointValue   float64
	TotalNumbers int
	DrawDate     time.Time
	Prizes       []string
}

type UpdateInput struct {
	Name       string
	GoalAmount float64
	PointValue float64
	DrawDate   time.Time
	Prizes     []string
}

type SellNumberInput struct {
	BuyerName     string
	BuyerPhone    string
	MemberID      *string
	PaymentMethod domain.PaymentMethod
	SoldByUserID  string
}

type DrawInput struct {
	RaffleID      string
	WinningNumber int
}

func (s *Service) List(ctx context.Context) ([]domain.RaffleSummary, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetActive(ctx context.Context) (*domain.RaffleSummary, error) {
	return s.repo.GetActive(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.RaffleSummary, error) {
	if id == "" {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListNumbers(ctx context.Context, raffleID string) ([]domain.RaffleNumber, error) {
	if raffleID == "" {
		return nil, domain.ErrValidation
	}
	return s.repo.ListNumbers(ctx, raffleID)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.RaffleSummary, error) {
	if strings.TrimSpace(input.Name) == "" {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	if input.GoalAmount <= 0 || input.PointValue <= 0 || input.TotalNumbers <= 0 {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	if input.DrawDate.IsZero() {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	if len(input.Prizes) == 0 {
		return domain.RaffleSummary{}, domain.ErrValidation
	}

	raffle := domain.Raffle{
		Name:         strings.TrimSpace(input.Name),
		GoalAmount:   input.GoalAmount,
		PointValue:   input.PointValue,
		TotalNumbers: input.TotalNumbers,
		DrawDate:     input.DrawDate,
		Status:       domain.RaffleStatusActive,
	}

	prizes := make([]domain.RafflePrize, len(input.Prizes))
	for i, description := range input.Prizes {
		prizes[i] = domain.RafflePrize{
			Description: strings.TrimSpace(description),
			Position:    i + 1,
		}
	}

	numbers := make([]domain.RaffleNumber, input.TotalNumbers)
	for i := 0; i < input.TotalNumbers; i++ {
		numbers[i] = domain.RaffleNumber{
			Number: i + 1,
			Status: domain.RaffleNumberAvailable,
		}
	}

	return s.repo.Create(ctx, raffle, prizes, numbers)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (domain.RaffleSummary, error) {
	if id == "" {
		return domain.RaffleSummary{}, domain.ErrValidation
	}

	summary, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.RaffleSummary{}, err
	}
	if summary.Status != domain.RaffleStatusActive {
		return domain.RaffleSummary{}, domain.ErrRaffleNotActive
	}

	if strings.TrimSpace(input.Name) == "" {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	if input.GoalAmount <= 0 || input.PointValue <= 0 {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	if input.DrawDate.IsZero() {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	if len(input.Prizes) == 0 {
		return domain.RaffleSummary{}, domain.ErrValidation
	}

	raffle := domain.Raffle{
		Name:       strings.TrimSpace(input.Name),
		GoalAmount: input.GoalAmount,
		PointValue: input.PointValue,
		DrawDate:   input.DrawDate,
	}

	prizes := make([]domain.RafflePrize, len(input.Prizes))
	for i, description := range input.Prizes {
		prizes[i] = domain.RafflePrize{
			Description: strings.TrimSpace(description),
			Position:    i + 1,
		}
	}

	return s.repo.Update(ctx, id, raffle, prizes)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return domain.ErrValidation
	}

	summary, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if summary.SoldNumbers > 0 {
		return domain.ErrRaffleHasSales
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) SellNumber(ctx context.Context, raffleID string, number int, input SellNumberInput) error {
	return s.validateAndSell(ctx, raffleID, []int{number}, input)
}

func (s *Service) SellNumbers(ctx context.Context, raffleID string, numbers []int, input SellNumberInput) error {
	if len(numbers) == 0 {
		return domain.ErrValidation
	}
	return s.validateAndSell(ctx, raffleID, numbers, input)
}

func (s *Service) Draw(ctx context.Context, input DrawInput) (domain.RaffleSummary, error) {
	if input.RaffleID == "" || input.WinningNumber <= 0 {
		return domain.RaffleSummary{}, domain.ErrValidation
	}
	return s.tx.Draw(ctx, input.RaffleID, input.WinningNumber)
}

func (s *Service) validateAndSell(ctx context.Context, raffleID string, numbers []int, input SellNumberInput) error {
	if raffleID == "" {
		return domain.ErrValidation
	}
	if strings.TrimSpace(input.BuyerName) == "" || strings.TrimSpace(input.BuyerPhone) == "" {
		return domain.ErrValidation
	}
	if input.SoldByUserID == "" {
		return domain.ErrValidation
	}
	if !isValidPaymentMethod(input.PaymentMethod) {
		return domain.ErrValidation
	}

	summary, err := s.repo.GetByID(ctx, raffleID)
	if err != nil {
		return err
	}
	if summary.Status != domain.RaffleStatusActive {
		return domain.ErrRaffleNotActive
	}

	for _, number := range numbers {
		if number <= 0 || number > summary.TotalNumbers {
			return domain.ErrInvalidRaffleNumber
		}
	}

	sale := SellNumberInput{
		BuyerName:     strings.TrimSpace(input.BuyerName),
		BuyerPhone:    normalizePhone(input.BuyerPhone),
		MemberID:      input.MemberID,
		PaymentMethod: input.PaymentMethod,
		SoldByUserID:  input.SoldByUserID,
	}

	if sale.MemberID == nil && s.memberLookup != nil {
		if member, err := s.memberLookup.GetByPhone(ctx, sale.BuyerPhone); err == nil && member != nil {
			sale.MemberID = &member.ID
		}
	}

	if len(numbers) == 1 {
		return s.tx.SellNumber(ctx, raffleID, numbers[0], sale)
	}
	return s.tx.SellNumbers(ctx, raffleID, numbers, sale)
}

func isValidPaymentMethod(method domain.PaymentMethod) bool {
	switch method {
	case domain.PaymentMethodCash, domain.PaymentMethodPix, domain.PaymentMethodCard, domain.PaymentMethodTransfer:
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
