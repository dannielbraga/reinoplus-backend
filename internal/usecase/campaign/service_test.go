package campaign_test

import (
	"context"
	"testing"
	"time"

	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/usecase/campaign"
)

type mockCampaignRepo struct {
	campaign domain.Campaign
}

func (m *mockCampaignRepo) List(ctx context.Context) ([]domain.Campaign, error) {
	return nil, nil
}

func (m *mockCampaignRepo) GetByID(ctx context.Context, id string) (domain.Campaign, error) {
	return m.campaign, nil
}

func (m *mockCampaignRepo) Create(ctx context.Context, campaignEntity domain.Campaign) (domain.Campaign, error) {
	return domain.Campaign{}, nil
}

func (m *mockCampaignRepo) ListContributions(ctx context.Context, campaignID string) ([]domain.Contribution, error) {
	return nil, nil
}

func (m *mockCampaignRepo) GetRaisedAmount(ctx context.Context, campaignID string) (float64, error) {
	return 0, nil
}

type mockContributionWriter struct{}

func (m *mockContributionWriter) CreateContribution(ctx context.Context, contribution domain.Contribution) (domain.Contribution, error) {
	return contribution, nil
}

func TestCreateContributionRejectsInactiveCampaign(t *testing.T) {
	service := campaign.NewService(&mockCampaignRepo{
		campaign: domain.Campaign{ID: "camp-1", Status: domain.CampaignStatusCompleted},
	}, &mockContributionWriter{}, nil)

	_, err := service.CreateContribution(context.Background(), campaign.CreateContributionInput{
		CampaignID: "camp-1", ContributorName: "João", ContributorPhone: "11999999999",
		Amount: 100, PaymentMethod: domain.PaymentMethodPix, ContributedAt: time.Now(),
		CreatedByUserID: "user-1",
	})
	if err != domain.ErrCampaignNotActive {
		t.Fatalf("expected ErrCampaignNotActive, got %v", err)
	}
}
