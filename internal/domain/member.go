package domain

import "time"

type Member struct {
	ID        string
	Name      string
	Phone     string
	BirthDate time.Time
	Address   string
	CreatedAt time.Time
}
