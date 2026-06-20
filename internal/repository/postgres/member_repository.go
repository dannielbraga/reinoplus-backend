package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reinoplus/reinoplus/internal/domain"
)

type MemberRepository struct {
	pool *pgxpool.Pool
}

func NewMemberRepository(pool *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{pool: pool}
}

func (r *MemberRepository) List(ctx context.Context, search string, limit, offset int) ([]domain.Member, error) {
	query := `
		SELECT id, name, phone, birth_date, address, created_at
		FROM members
	`
	args := []any{}
	if search != "" {
		query += ` WHERE name ILIKE $1 OR phone ILIKE $1`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY name ASC LIMIT `
	if len(args) == 0 {
		query += fmt.Sprintf("%d OFFSET %d", limit, offset)
	} else {
		query += fmt.Sprintf("$2 OFFSET $3")
		args = append(args, limit, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	members := []domain.Member{}
	for rows.Next() {
		var member domain.Member
		if err := rows.Scan(&member.ID, &member.Name, &member.Phone, &member.BirthDate, &member.Address, &member.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *MemberRepository) GetByID(ctx context.Context, id string) (domain.Member, error) {
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

func (r *MemberRepository) GetByPhone(ctx context.Context, phone string) (*domain.Member, error) {
	const query = `
		SELECT id, name, phone, birth_date, address, created_at
		FROM members WHERE phone = $1
	`
	var member domain.Member
	err := r.pool.QueryRow(ctx, query, phone).Scan(
		&member.ID, &member.Name, &member.Phone, &member.BirthDate, &member.Address, &member.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get member by phone: %w", err)
	}
	return &member, nil
}

func (r *MemberRepository) Create(ctx context.Context, member domain.Member) (domain.Member, error) {
	const query = `
		INSERT INTO members (name, phone, birth_date, address)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, phone, birth_date, address, created_at
	`
	var created domain.Member
	err := r.pool.QueryRow(ctx, query, member.Name, member.Phone, member.BirthDate, member.Address).Scan(
		&created.ID, &created.Name, &created.Phone, &created.BirthDate, &created.Address, &created.CreatedAt,
	)
	if err != nil {
		return domain.Member{}, fmt.Errorf("create member: %w", err)
	}
	return created, nil
}

func (r *MemberRepository) Update(ctx context.Context, member domain.Member) (domain.Member, error) {
	const query = `
		UPDATE members
		SET name = $2, phone = $3, birth_date = $4, address = $5
		WHERE id = $1
		RETURNING id, name, phone, birth_date, address, created_at
	`
	var updated domain.Member
	err := r.pool.QueryRow(ctx, query, member.ID, member.Name, member.Phone, member.BirthDate, member.Address).Scan(
		&updated.ID, &updated.Name, &updated.Phone, &updated.BirthDate, &updated.Address, &updated.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Member{}, fmt.Errorf("update member: %w", err)
	}
	return updated, nil
}
