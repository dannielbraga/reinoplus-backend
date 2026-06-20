package domain

import "time"

type CampaignStatus string

const (
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusCancelled CampaignStatus = "cancelled"
)

type Campaign struct {
	ID                 string
	Name               string
	Description        string
	GoalAmount         float64
	StartDate          time.Time
	EndDate            time.Time
	Status             CampaignStatus
	IsRecurring        bool
	RecurrenceInterval *RecurrenceInterval
	DurationMonths     *int
	CreatedAt          time.Time
}

type PaymentMethod string

const (
	PaymentMethodCash     PaymentMethod = "cash"
	PaymentMethodPix      PaymentMethod = "pix"
	PaymentMethodCard     PaymentMethod = "card"
	PaymentMethodTransfer PaymentMethod = "transfer"
)

type RecurrenceInterval string

const (
	RecurrenceIntervalBiweekly   RecurrenceInterval = "biweekly"
	RecurrenceIntervalMonthly    RecurrenceInterval = "monthly"
	RecurrenceIntervalBimonthly  RecurrenceInterval = "bimonthly"
	RecurrenceIntervalQuarterly  RecurrenceInterval = "quarterly"
)

func IsValidRecurrenceInterval(interval RecurrenceInterval) bool {
	switch interval {
	case RecurrenceIntervalBiweekly, RecurrenceIntervalMonthly,
		RecurrenceIntervalBimonthly, RecurrenceIntervalQuarterly:
		return true
	default:
		return false
	}
}

func TotalInstallments(durationMonths int, interval RecurrenceInterval) int {
	if durationMonths <= 0 {
		return 0
	}

	switch interval {
	case RecurrenceIntervalBiweekly:
		return durationMonths * 2
	case RecurrenceIntervalMonthly:
		return durationMonths
	case RecurrenceIntervalBimonthly:
		return (durationMonths + 1) / 2
	case RecurrenceIntervalQuarterly:
		return (durationMonths + 2) / 3
	default:
		return durationMonths
	}
}

type Contribution struct {
	ID                 string
	CampaignID         string
	MemberID           *string
	ContributorName    string
	ContributorPhone   string
	MemberName         string
	Amount             float64
	PaymentMethod      PaymentMethod
	ContributedAt      time.Time
	IsRecurring        bool
	RecurrenceInterval *RecurrenceInterval
	IsPaid             bool
	InstallmentNumber  *int
	CreatedAt          time.Time
	CreatedByUserID    *string
	CreatedByName      string
}
