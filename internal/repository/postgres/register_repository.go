package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reinoplus/reinoplus/internal/domain"
)

type RegisterRepository struct {
	pool *pgxpool.Pool
}

func NewRegisterRepository(pool *pgxpool.Pool) *RegisterRepository {
	return &RegisterRepository{pool: pool}
}

type RegisterInput struct {
	Name         string
	Email        string
	PasswordHash string
	Phone        string
	BirthDate    time.Time
	Address      string
}

func (r *RegisterRepository) RegisterUserWithMember(ctx context.Context, input domain.RegisterUserInput) (domain.User, domain.Member, error) {
	return r.registerUserWithMember(ctx, RegisterInput{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: input.PasswordHash,
		Phone:        input.Phone,
		BirthDate:    input.BirthDate,
		Address:      input.Address,
	})
}

func (r *RegisterRepository) registerUserWithMember(ctx context.Context, input RegisterInput) (domain.User, domain.Member, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, domain.Member{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var member domain.Member
	const memberQuery = `
		INSERT INTO members (name, phone, birth_date, address)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, phone, birth_date, address, created_at
	`
	err = tx.QueryRow(ctx, memberQuery, input.Name, input.Phone, input.BirthDate, input.Address).Scan(
		&member.ID, &member.Name, &member.Phone, &member.BirthDate, &member.Address, &member.CreatedAt,
	)
	if err != nil {
		return domain.User{}, domain.Member{}, mapRegisterError(err)
	}

	var user domain.User
	memberID := member.ID
	const userQuery = `
		INSERT INTO users (name, email, password_hash, role, member_id)
		VALUES ($1, $2, $3, 'member', $4)
		RETURNING id, name, email, password_hash, role, member_id, created_at
	`
	err = tx.QueryRow(ctx, userQuery, input.Name, input.Email, input.PasswordHash, memberID).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role, &user.MemberID, &user.CreatedAt,
	)
	if err != nil {
		return domain.User{}, domain.Member{}, mapRegisterError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, domain.Member{}, fmt.Errorf("commit tx: %w", err)
	}

	return user, member, nil
}

func mapRegisterError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return fmt.Errorf("register user: %w", err)
}

func (r *RegisterRepository) UpdateUserProfile(ctx context.Context, user domain.User, member domain.Member, passwordHash *string) (domain.User, domain.Member, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, domain.Member{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if user.MemberID != nil {
		const memberQuery = `
			UPDATE members SET name = $2, phone = $3, birth_date = $4, address = $5
			WHERE id = $1
			RETURNING id, name, phone, birth_date, address, created_at
		`
		err = tx.QueryRow(ctx, memberQuery, *user.MemberID, member.Name, member.Phone, member.BirthDate, member.Address).Scan(
			&member.ID, &member.Name, &member.Phone, &member.BirthDate, &member.Address, &member.CreatedAt,
		)
		if err != nil {
			return domain.User{}, domain.Member{}, mapRegisterError(err)
		}
	}

	var updated domain.User
	if passwordHash != nil {
		const userQuery = `
			UPDATE users SET name = $2, email = $3, password_hash = $4
			WHERE id = $1
			RETURNING id, name, email, password_hash, role, member_id, created_at
		`
		err = tx.QueryRow(ctx, userQuery, user.ID, user.Name, user.Email, *passwordHash).Scan(
			&updated.ID, &updated.Name, &updated.Email, &updated.PasswordHash, &updated.Role, &updated.MemberID, &updated.CreatedAt,
		)
	} else {
		const userQuery = `
			UPDATE users SET name = $2, email = $3
			WHERE id = $1
			RETURNING id, name, email, password_hash, role, member_id, created_at
		`
		err = tx.QueryRow(ctx, userQuery, user.ID, user.Name, user.Email).Scan(
			&updated.ID, &updated.Name, &updated.Email, &updated.PasswordHash, &updated.Role, &updated.MemberID, &updated.CreatedAt,
		)
	}
	if err != nil {
		return domain.User{}, domain.Member{}, mapRegisterError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, domain.Member{}, fmt.Errorf("commit tx: %w", err)
	}

	return updated, member, nil
}

func (r *RegisterRepository) GetMemberByID(ctx context.Context, id string) (domain.Member, error) {
	const query = `
		SELECT id, name, phone, birth_date, address, created_at
		FROM members WHERE id = $1
	`
	var member domain.Member
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&member.ID, &member.Name, &member.Phone, &member.BirthDate, &member.Address, &member.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Member{}, fmt.Errorf("get member: %w", err)
	}
	return member, nil
}
