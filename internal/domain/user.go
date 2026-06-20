package domain

import "time"

type UserRole string

const (
	// UserRoleAdmin has full access to all write operations.
	UserRoleAdmin UserRole = "admin"
	// UserRoleMember is assigned on self-service registration (read-only + raffle sales).
	UserRoleMember UserRole = "member"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         UserRole
	MemberID     *string
	CreatedAt    time.Time
}

type RegisterUserInput struct {
	Name         string
	Email        string
	PasswordHash string
	Phone        string
	BirthDate    time.Time
	Address      string
}
