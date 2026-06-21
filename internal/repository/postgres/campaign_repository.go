package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

const campaignSelectColumns = `
	id, name, description, goal_amount, start_date, end_date, status,
	is_recurring, recurrence_interval, duration_months, created_at
`

func (r *CampaignRepository) List(ctx context.Context) ([]domain.Campaign, error) {
	query := `
		SELECT ` + campaignSelectColumns + `
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
		campaign, err := scanCampaign(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan campaign: %w", err)
		}
		campaigns = append(campaigns, campaign)
	}
	return campaigns, rows.Err()
}

func (r *CampaignRepository) GetByID(ctx context.Context, id string) (domain.Campaign, error) {
	query := `SELECT ` + campaignSelectColumns + ` FROM campaigns WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	campaign, err := scanCampaign(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Campaign{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("get campaign: %w", err)
	}
	return campaign, nil
}

func (r *CampaignRepository) Create(ctx context.Context, campaign domain.Campaign) (domain.Campaign, error) {
	query := `
		INSERT INTO campaigns (
			name, description, goal_amount, start_date, end_date, status,
			is_recurring, recurrence_interval, duration_months
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + campaignSelectColumns + `
	`
	row := r.pool.QueryRow(ctx, query,
		campaign.Name, campaign.Description, campaign.GoalAmount,
		campaign.StartDate, campaign.EndDate, campaign.Status,
		campaign.IsRecurring, recurrenceIntervalToDB(campaign.RecurrenceInterval), campaign.DurationMonths,
	)
	created, err := scanCampaign(row.Scan)
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("create campaign: %w", err)
	}
	return created, nil
}

func (r *CampaignRepository) Update(ctx context.Context, id string, campaign domain.Campaign) (domain.Campaign, error) {
	query := `
		UPDATE campaigns
		SET name = $2, description = $3, goal_amount = $4, start_date = $5, end_date = $6,
		    is_recurring = $7, recurrence_interval = $8, duration_months = $9
		WHERE id = $1
		RETURNING ` + campaignSelectColumns + `
	`
	row := r.pool.QueryRow(ctx, query, id,
		campaign.Name, campaign.Description, campaign.GoalAmount,
		campaign.StartDate, campaign.EndDate,
		campaign.IsRecurring, recurrenceIntervalToDB(campaign.RecurrenceInterval), campaign.DurationMonths,
	)
	updated, err := scanCampaign(row.Scan)
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("update campaign: %w", err)
	}
	return updated, nil
}

func (r *CampaignRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM campaigns WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CampaignRepository) SetStatus(ctx context.Context, id string, status domain.CampaignStatus) (domain.Campaign, error) {
	query := `
		UPDATE campaigns SET status = $2 WHERE id = $1
		RETURNING ` + campaignSelectColumns + `
	`
	row := r.pool.QueryRow(ctx, query, id, status)
	updated, err := scanCampaign(row.Scan)
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("set campaign status: %w", err)
	}
	return updated, nil
}

func (r *CampaignRepository) CountContributions(ctx context.Context, campaignID string) (int, error) {
	const query = `SELECT COUNT(*) FROM contributions WHERE campaign_id = $1`
	var count int
	if err := r.pool.QueryRow(ctx, query, campaignID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count contributions: %w", err)
	}
	return count, nil
}

func (r *CampaignRepository) ListContributions(ctx context.Context, campaignID string, search string) ([]domain.Contribution, error) {
	query := `
		SELECT c.id, c.campaign_id, c.member_id,
		       COALESCE(m.name, c.contributor_name) AS display_name,
		       c.contributor_name, c.contributor_phone,
		       c.amount, c.payment_method, c.contributed_at,
		       c.is_recurring, c.recurrence_interval, c.is_paid, c.installment_number,
		       c.created_at, c.created_by_user_id, COALESCE(u.name, '') AS created_by_name
		FROM contributions c
		LEFT JOIN members m ON m.id = c.member_id
		LEFT JOIN users u ON u.id = c.created_by_user_id
		WHERE c.campaign_id = $1
	`
	args := []any{campaignID}
	if search != "" {
		digits := normalizePhoneDigits(search)
		query += `
			AND (
				c.contributor_name ILIKE $2
				OR COALESCE(m.name, '') ILIKE $2
				OR c.contributor_phone ILIKE $2
		`
		args = append(args, "%"+search+"%")
		if digits != "" {
			query += `
				OR c.contributor_phone LIKE $3
			`
			args = append(args, "%"+digits+"%")
		}
		query += `
			)
		`
	}
	query += `
		ORDER BY c.installment_number ASC NULLS LAST, c.contributed_at DESC, c.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list contributions: %w", err)
	}
	defer rows.Close()

	contributions := []domain.Contribution{}
	for rows.Next() {
		contribution, err := scanContribution(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan contribution: %w", err)
		}
		contributions = append(contributions, contribution)
	}
	return contributions, rows.Err()
}

func (r *CampaignRepository) GetCampaignTotals(ctx context.Context, campaignID string) (float64, float64, error) {
	const query = `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE is_paid = true), 0),
			COALESCE(SUM(amount) FILTER (WHERE is_paid = false), 0)
		FROM contributions
		WHERE campaign_id = $1
	`
	var paid, promised float64
	if err := r.pool.QueryRow(ctx, query, campaignID).Scan(&paid, &promised); err != nil {
		return 0, 0, fmt.Errorf("sum contribution totals: %w", err)
	}
	return paid, promised, nil
}

func (r *CampaignRepository) CreateContribution(ctx context.Context, contribution domain.Contribution) (domain.Contribution, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Contribution{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertQuery = `
		INSERT INTO contributions (
			campaign_id, member_id, contributor_name, contributor_phone,
			amount, payment_method, contributed_at,
			is_recurring, recurrence_interval, is_paid, installment_number, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, campaign_id, member_id, contributor_name, contributor_phone,
		          amount, payment_method, contributed_at,
		          is_recurring, recurrence_interval, is_paid, installment_number,
		          created_at, created_by_user_id
	`

	row := tx.QueryRow(ctx, insertQuery,
		contribution.CampaignID, contribution.MemberID, contribution.ContributorName,
		contribution.ContributorPhone, contribution.Amount,
		contribution.PaymentMethod, contribution.ContributedAt,
		false, nil, contribution.IsPaid,
		contribution.InstallmentNumber, contribution.CreatedByUserID,
	)
	created, err := scanCreatedContribution(row.Scan)
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

type scanFunc func(dest ...any) error

func scanCampaign(scan scanFunc) (domain.Campaign, error) {
	var campaign domain.Campaign
	var recurrenceInterval *string
	err := scan(
		&campaign.ID, &campaign.Name, &campaign.Description, &campaign.GoalAmount,
		&campaign.StartDate, &campaign.EndDate, &campaign.Status,
		&campaign.IsRecurring, &recurrenceInterval, &campaign.DurationMonths, &campaign.CreatedAt,
	)
	if err != nil {
		return domain.Campaign{}, err
	}
	campaign.RecurrenceInterval = scanRecurrenceInterval(recurrenceInterval)
	return campaign, nil
}

func scanCreatedContribution(scan scanFunc) (domain.Contribution, error) {
	var contribution domain.Contribution
	var recurrenceInterval *string
	err := scan(
		&contribution.ID, &contribution.CampaignID, &contribution.MemberID,
		&contribution.ContributorName, &contribution.ContributorPhone,
		&contribution.Amount, &contribution.PaymentMethod, &contribution.ContributedAt,
		&contribution.IsRecurring, &recurrenceInterval, &contribution.IsPaid, &contribution.InstallmentNumber,
		&contribution.CreatedAt, &contribution.CreatedByUserID,
	)
	if err != nil {
		return domain.Contribution{}, err
	}
	contribution.RecurrenceInterval = scanRecurrenceInterval(recurrenceInterval)
	contribution.MemberName = contribution.ContributorName
	return contribution, nil
}

func scanContribution(scan scanFunc) (domain.Contribution, error) {
	var contribution domain.Contribution
	var recurrenceInterval *string
	err := scan(
		&contribution.ID, &contribution.CampaignID, &contribution.MemberID,
		&contribution.MemberName, &contribution.ContributorName, &contribution.ContributorPhone,
		&contribution.Amount, &contribution.PaymentMethod, &contribution.ContributedAt,
		&contribution.IsRecurring, &recurrenceInterval, &contribution.IsPaid, &contribution.InstallmentNumber,
		&contribution.CreatedAt, &contribution.CreatedByUserID, &contribution.CreatedByName,
	)
	if err != nil {
		return domain.Contribution{}, err
	}
	contribution.RecurrenceInterval = scanRecurrenceInterval(recurrenceInterval)
	return contribution, nil
}

func recurrenceIntervalToDB(value *domain.RecurrenceInterval) *string {
	if value == nil {
		return nil
	}
	formatted := string(*value)
	return &formatted
}

func scanRecurrenceInterval(value *string) *domain.RecurrenceInterval {
	if value == nil || *value == "" {
		return nil
	}
	interval := domain.RecurrenceInterval(*value)
	return &interval
}

func normalizePhoneDigits(phone string) string {
	var digits strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}
