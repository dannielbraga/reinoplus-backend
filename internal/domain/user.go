package domain

import "time"

type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
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
