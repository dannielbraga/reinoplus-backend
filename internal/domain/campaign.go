package domain

import "time"

type CampaignStatus string

const (
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusCancelled CampaignStatus = "cancelled"
)

type Campaign struct {
	ID          string
	Name        string
	Description string
	GoalAmount  float64
	StartDate   time.Time
	EndDate     time.Time
	Status      CampaignStatus
	CreatedAt   time.Time
}

type PaymentMethod string

const (
	PaymentMethodCash     PaymentMethod = "cash"
	PaymentMethodPix      PaymentMethod = "pix"
	PaymentMethodCard     PaymentMethod = "card"
	PaymentMethodTransfer PaymentMethod = "transfer"
)

type Contribution struct {
	ID               string
	CampaignID       string
	MemberID         *string
	ContributorName  string
	ContributorPhone string
	MemberName       string
	Amount           float64
	PaymentMethod    PaymentMethod
	ContributedAt    time.Time
	CreatedAt        time.Time
	CreatedByUserID  *string
	CreatedByName    string
}
