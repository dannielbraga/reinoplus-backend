package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reinoplus/reinoplus/internal/domain"
)

type CampaignRepository struct {
	pool *pgxpool.Pool
}

func NewCampaignRepository(pool *pgxpool.Pool) *CampaignRepository {
	return &CampaignRepository{pool: pool}
}

func (r *CampaignRepository) List(ctx context.Context) ([]domain.Campaign, error) {
	const query = `
		SELECT id, name, description, goal_amount, start_date, end_date, status, created_at
		FROM campaigns
		ORDER BY CASE status WHEN 'active' THEN 0 ELSE 1 END, created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := []domain.Campaign{}
	for rows.Next() {
		var campaign domain.Campaign
		if err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Description, &campaign.GoalAmount,
			&campaign.StartDate, &campaign.EndDate, &campaign.Status, &campaign.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan campaign: %w", err)
		}
		campaigns = append(campaigns, campaign)
	}
	return campaigns, rows.Err()
}

func (r *CampaignRepository) GetByID(ctx context.Context, id string) (domain.Campaign, error) {
	const query = `
		SELECT id, name, description, goal_amount, start_date, end_date, status, created_at
		FROM campaigns WHERE id = $1
	`
	var campaign domain.Campaign
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&campaign.ID, &campaign.Name, &campaign.Description, &campaign.GoalAmount,
		&campaign.StartDate, &campaign.EndDate, &campaign.Status, &campaign.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Campaign{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("get campaign: %w", err)
	}
	return campaign, nil
}

func (r *CampaignRepository) Create(ctx context.Context, campaign domain.Campaign) (domain.Campaign, error) {
	const query = `
		INSERT INTO campaigns (name, description, goal_amount, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, goal_amount, start_date, end_date, status, created_at
	`
	var created domain.Campaign
	err := r.pool.QueryRow(ctx, query,
		campaign.Name, campaign.Description, campaign.GoalAmount,
		campaign.StartDate, campaign.EndDate, campaign.Status,
	).Scan(
		&created.ID, &created.Name, &created.Description, &created.GoalAmount,
		&created.StartDate, &created.EndDate, &created.Status, &created.CreatedAt,
	)
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("create campaign: %w", err)
	}
	return created, nil
}

func (r *CampaignRepository) ListContributions(ctx context.Context, campaignID string) ([]domain.Contribution, error) {
	const query = `
		SELECT c.id, c.campaign_id, c.member_id,
		       COALESCE(m.name, c.contributor_name) AS display_name,
		       c.contributor_name, c.contributor_phone,
		       c.amount, c.payment_method, c.contributed_at, c.created_at,
		       c.created_by_user_id, COALESCE(u.name, '') AS created_by_name
		FROM contributions c
		LEFT JOIN members m ON m.id = c.member_id
		LEFT JOIN users u ON u.id = c.created_by_user_id
		WHERE c.campaign_id = $1
		ORDER BY c.contributed_at DESC, c.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list contributions: %w", err)
	}
	defer rows.Close()

	contributions := []domain.Contribution{}
	for rows.Next() {
		var contribution domain.Contribution
		if err := rows.Scan(
			&contribution.ID, &contribution.CampaignID, &contribution.MemberID,
			&contribution.MemberName, &contribution.ContributorName, &contribution.ContributorPhone,
			&contribution.Amount, &contribution.PaymentMethod, &contribution.ContributedAt, &contribution.CreatedAt,
			&contribution.CreatedByUserID, &contribution.CreatedByName,
		); err != nil {
			return nil, fmt.Errorf("scan contribution: %w", err)
		}
		contributions = append(contributions, contribution)
	}
	return contributions, rows.Err()
}

func (r *CampaignRepository) GetRaisedAmount(ctx context.Context, campaignID string) (float64, error) {
	const query = `SELECT COALESCE(SUM(amount), 0) FROM contributions WHERE campaign_id = $1`
	var raised float64
	if err := r.pool.QueryRow(ctx, query, campaignID).Scan(&raised); err != nil {
		return 0, fmt.Errorf("sum contributions: %w", err)
	}
	return raised, nil
}

func (r *CampaignRepository) CreateContribution(ctx context.Context, contribution domain.Contribution) (domain.Contribution, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Contribution{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertQuery = `
		INSERT INTO contributions (campaign_id, member_id, contributor_name, contributor_phone, amount, payment_method, contributed_at, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, campaign_id, member_id, contributor_name, contributor_phone, amount, payment_method, contributed_at, created_at, created_by_user_id
	`

	var created domain.Contribution
	err = tx.QueryRow(ctx, insertQuery,
		contribution.CampaignID, contribution.MemberID, contribution.ContributorName,
		contribution.ContributorPhone, contribution.Amount,
		contribution.PaymentMethod, contribution.ContributedAt, contribution.CreatedByUserID,
	).Scan(
		&created.ID, &created.CampaignID, &created.MemberID,
		&created.ContributorName, &created.ContributorPhone,
		&created.Amount, &created.PaymentMethod, &created.ContributedAt, &created.CreatedAt,
		&created.CreatedByUserID,
	)
	if err != nil {
		return domain.Contribution{}, fmt.Errorf("insert contribution: %w", err)
	}

	created.MemberName = created.ContributorName
	if created.MemberID != nil {
		const memberQuery = `SELECT name FROM members WHERE id = $1`
		if err := tx.QueryRow(ctx, memberQuery, *created.MemberID).Scan(&created.MemberName); err != nil {
			return domain.Contribution{}, fmt.Errorf("load member name: %w", err)
		}
	}

	if created.CreatedByUserID != nil {
		const userQuery = `SELECT name FROM users WHERE id = $1`
		_ = tx.QueryRow(ctx, userQuery, *created.CreatedByUserID).Scan(&created.CreatedByName)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Contribution{}, fmt.Errorf("commit contribution tx: %w", err)
	}

	return created, nil
}
