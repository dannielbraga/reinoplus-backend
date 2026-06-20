package raffle_test

import (
	"context"
	"testing"

	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/usecase/raffle"
)

type mockRaffleRepo struct {
	summary domain.RaffleSummary
}

func (m *mockRaffleRepo) List(ctx context.Context) ([]domain.RaffleSummary, error) {
	return nil, nil
}

func (m *mockRaffleRepo) GetActive(ctx context.Context) (*domain.RaffleSummary, error) {
	return nil, nil
}

func (m *mockRaffleRepo) GetByID(ctx context.Context, id string) (domain.RaffleSummary, error) {
	return m.summary, nil
}

func (m *mockRaffleRepo) Create(ctx context.Context, raffleEntity domain.Raffle, prizes []domain.RafflePrize, numbers []domain.RaffleNumber) (domain.RaffleSummary, error) {
	return domain.RaffleSummary{}, nil
}

func (m *mockRaffleRepo) Update(ctx context.Context, id string, raffleEntity domain.Raffle, prizes []domain.RafflePrize) (domain.RaffleSummary, error) {
	return m.summary, nil
}

func (m *mockRaffleRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockRaffleRepo) ListNumbers(ctx context.Context, raffleID string) ([]domain.RaffleNumber, error) {
	return nil, nil
}

func (m *mockRaffleRepo) CountSoldNumbers(ctx context.Context, raffleID string) (int, error) {
	return 0, nil
}

type mockTxRepo struct {
	sellErr error
}

func (m *mockTxRepo) SellNumber(ctx context.Context, raffleID string, number int, sale raffle.SellNumberInput) error {
	return m.sellErr
}

func (m *mockTxRepo) SellNumbers(ctx context.Context, raffleID string, numbers []int, sale raffle.SellNumberInput) error {
	return m.sellErr
}

func (m *mockTxRepo) Draw(ctx context.Context, raffleID string, winningNumber int) (domain.RaffleSummary, error) {
	return domain.RaffleSummary{}, nil
}

func TestSellNumberRejectsInactiveRaffle(t *testing.T) {
	service := raffle.NewService(&mockRaffleRepo{
		summary: domain.RaffleSummary{
			Raffle: domain.Raffle{ID: "raffle-1", Status: domain.RaffleStatusFinished, TotalNumbers: 100},
		},
	}, &mockTxRepo{}, nil)

	err := service.SellNumber(context.Background(), "raffle-1", 10, raffle.SellNumberInput{
		BuyerName: "João", BuyerPhone: "11999999999", PaymentMethod: domain.PaymentMethodPix, SoldByUserID: "user-1",
	})
	if err != domain.ErrRaffleNotActive {
		t.Fatalf("expected ErrRaffleNotActive, got %v", err)
	}
}

func TestSellNumberRejectsInvalidNumber(t *testing.T) {
	service := raffle.NewService(&mockRaffleRepo{
		summary: domain.RaffleSummary{
			Raffle: domain.Raffle{ID: "raffle-1", Status: domain.RaffleStatusActive, TotalNumbers: 50},
		},
	}, &mockTxRepo{}, nil)

	err := service.SellNumber(context.Background(), "raffle-1", 999, raffle.SellNumberInput{
		BuyerName: "João", BuyerPhone: "11999999999", PaymentMethod: domain.PaymentMethodPix, SoldByUserID: "user-1",
	})
	if err != domain.ErrInvalidRaffleNumber {
		t.Fatalf("expected ErrInvalidRaffleNumber, got %v", err)
	}
}
