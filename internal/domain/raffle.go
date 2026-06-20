package domain

import "time"

type RaffleStatus string

const (
	RaffleStatusActive    RaffleStatus = "active"
	RaffleStatusFinished  RaffleStatus = "finished"
	RaffleStatusCancelled RaffleStatus = "cancelled"
)

type RaffleNumberStatus string

const (
	RaffleNumberAvailable RaffleNumberStatus = "available"
	RaffleNumberSold      RaffleNumberStatus = "sold"
)

type Raffle struct {
	ID           string
	Name         string
	GoalAmount   float64
	PointValue   float64
	TotalNumbers int
	DrawDate     time.Time
	Status       RaffleStatus
	CreatedAt    time.Time
}

type RafflePrize struct {
	ID          string
	RaffleID    string
	Description string
	Position    int
}

type RaffleNumber struct {
	ID            string
	RaffleID      string
	Number        int
	Status        RaffleNumberStatus
	BuyerName     *string
	BuyerPhone    *string
	MemberID      *string
	SoldByUserID  *string
	SoldByName    *string
	SoldAt        *time.Time
	PaymentMethod *PaymentMethod
}

type RaffleDraw struct {
	ID            string
	RaffleID      string
	WinningNumber int
	DrawnAt       time.Time
}

type RaffleSummary struct {
	Raffle
	Prizes       []RafflePrize
	SoldNumbers  int
	WinnerName   *string
	WinningNumber *int
}
