package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/usecase/raffle"
)

type RaffleRepository struct {
	pool *pgxpool.Pool
}

func NewRaffleRepository(pool *pgxpool.Pool) *RaffleRepository {
	return &RaffleRepository{pool: pool}
}

func (r *RaffleRepository) List(ctx context.Context) ([]domain.RaffleSummary, error) {
	const query = `
		SELECT r.id, r.name, r.goal_amount, r.point_value, r.total_numbers, r.draw_date, r.status, r.created_at
		FROM raffles r
		ORDER BY CASE r.status WHEN 'active' THEN 0 ELSE 1 END, r.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list raffles: %w", err)
	}
	defer rows.Close()

	summaries := []domain.RaffleSummary{}
	for rows.Next() {
		summary, err := scanRaffleSummaryRow(rows)
		if err != nil {
			return nil, err
		}
		enriched, err := r.enrichSummary(ctx, summary)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, enriched)
	}
	return summaries, rows.Err()
}

func (r *RaffleRepository) GetActive(ctx context.Context) (*domain.RaffleSummary, error) {
	const query = `
		SELECT r.id, r.name, r.goal_amount, r.point_value, r.total_numbers, r.draw_date, r.status, r.created_at
		FROM raffles r
		WHERE r.status = 'active'
		ORDER BY r.created_at DESC
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query)
	summary, err := scanRaffleSummaryRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	enriched, err := r.enrichSummary(ctx, summary)
	if err != nil {
		return nil, err
	}
	return &enriched, nil
}

func (r *RaffleRepository) GetByID(ctx context.Context, id string) (domain.RaffleSummary, error) {
	const query = `
		SELECT r.id, r.name, r.goal_amount, r.point_value, r.total_numbers, r.draw_date, r.status, r.created_at
		FROM raffles r WHERE r.id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	summary, err := scanRaffleSummaryRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RaffleSummary{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RaffleSummary{}, err
	}
	return r.enrichSummary(ctx, summary)
}

func (r *RaffleRepository) Create(ctx context.Context, raffleEntity domain.Raffle, prizes []domain.RafflePrize, numbers []domain.RaffleNumber) (domain.RaffleSummary, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertRaffle = `
		INSERT INTO raffles (name, goal_amount, point_value, total_numbers, draw_date, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, goal_amount, point_value, total_numbers, draw_date, status, created_at
	`

	var summary domain.RaffleSummary
	err = tx.QueryRow(ctx, insertRaffle,
		raffleEntity.Name, raffleEntity.GoalAmount, raffleEntity.PointValue,
		raffleEntity.TotalNumbers, raffleEntity.DrawDate, raffleEntity.Status,
	).Scan(
		&summary.ID, &summary.Name, &summary.GoalAmount, &summary.PointValue,
		&summary.TotalNumbers, &summary.DrawDate, &summary.Status, &summary.CreatedAt,
	)
	if err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("insert raffle: %w", err)
	}

	const insertPrize = `
		INSERT INTO raffle_prizes (raffle_id, description, position)
		VALUES ($1, $2, $3)
		RETURNING id, raffle_id, description, position
	`
	summary.Prizes = make([]domain.RafflePrize, 0, len(prizes))
	for _, prize := range prizes {
		var created domain.RafflePrize
		if err := tx.QueryRow(ctx, insertPrize, summary.ID, prize.Description, prize.Position).Scan(
			&created.ID, &created.RaffleID, &created.Description, &created.Position,
		); err != nil {
			return domain.RaffleSummary{}, fmt.Errorf("insert prize: %w", err)
		}
		summary.Prizes = append(summary.Prizes, created)
	}

	const insertNumber = `
		INSERT INTO raffle_numbers (raffle_id, number, status)
		VALUES ($1, $2, 'available')
	`
	for _, number := range numbers {
		if _, err := tx.Exec(ctx, insertNumber, summary.ID, number.Number); err != nil {
			return domain.RaffleSummary{}, fmt.Errorf("insert raffle number: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("commit raffle tx: %w", err)
	}

	summary.SoldNumbers = 0
	return summary, nil
}

func (r *RaffleRepository) Update(ctx context.Context, id string, raffleEntity domain.Raffle, prizes []domain.RafflePrize) (domain.RaffleSummary, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const statusQuery = `SELECT status FROM raffles WHERE id = $1`
	var status domain.RaffleStatus
	if err := tx.QueryRow(ctx, statusQuery, id).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RaffleSummary{}, domain.ErrNotFound
		}
		return domain.RaffleSummary{}, fmt.Errorf("load raffle status: %w", err)
	}
	if status != domain.RaffleStatusActive {
		return domain.RaffleSummary{}, domain.ErrRaffleNotActive
	}

	const updateRaffle = `
		UPDATE raffles
		SET name = $2, goal_amount = $3, point_value = $4, draw_date = $5
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, updateRaffle, id, raffleEntity.Name, raffleEntity.GoalAmount, raffleEntity.PointValue, raffleEntity.DrawDate); err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("update raffle: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM raffle_prizes WHERE raffle_id = $1`, id); err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("delete prizes: %w", err)
	}

	const insertPrize = `
		INSERT INTO raffle_prizes (raffle_id, description, position)
		VALUES ($1, $2, $3)
	`
	for _, prize := range prizes {
		if _, err := tx.Exec(ctx, insertPrize, id, prize.Description, prize.Position); err != nil {
			return domain.RaffleSummary{}, fmt.Errorf("insert prize: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("commit update tx: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *RaffleRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM raffles WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete raffle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *RaffleRepository) ListNumbers(ctx context.Context, raffleID string) ([]domain.RaffleNumber, error) {
	const query = `
		SELECT rn.id, rn.raffle_id, rn.number, rn.status, rn.buyer_name, rn.buyer_phone,
		       rn.member_id, rn.sold_by_user_id, u.name, rn.sold_at, rn.payment_method
		FROM raffle_numbers rn
		LEFT JOIN users u ON u.id = rn.sold_by_user_id
		WHERE rn.raffle_id = $1
		ORDER BY rn.number ASC
	`
	rows, err := r.pool.Query(ctx, query, raffleID)
	if err != nil {
		return nil, fmt.Errorf("list raffle numbers: %w", err)
	}
	defer rows.Close()

	numbers := []domain.RaffleNumber{}
	for rows.Next() {
		var number domain.RaffleNumber
		if err := rows.Scan(
			&number.ID, &number.RaffleID, &number.Number, &number.Status,
			&number.BuyerName, &number.BuyerPhone, &number.MemberID,
			&number.SoldByUserID, &number.SoldByName, &number.SoldAt, &number.PaymentMethod,
		); err != nil {
			return nil, fmt.Errorf("scan raffle number: %w", err)
		}
		numbers = append(numbers, number)
	}
	return numbers, rows.Err()
}

func (r *RaffleRepository) CountSoldNumbers(ctx context.Context, raffleID string) (int, error) {
	const query = `SELECT COUNT(*) FROM raffle_numbers WHERE raffle_id = $1 AND status = 'sold'`
	var count int
	if err := r.pool.QueryRow(ctx, query, raffleID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count sold numbers: %w", err)
	}
	return count, nil
}

func (r *RaffleRepository) SellNumber(ctx context.Context, raffleID string, number int, sale raffle.SellNumberInput) error {
	return r.sellNumbers(ctx, raffleID, []int{number}, sale)
}

func (r *RaffleRepository) SellNumbers(ctx context.Context, raffleID string, numbers []int, sale raffle.SellNumberInput) error {
	return r.sellNumbers(ctx, raffleID, numbers, sale)
}

func (r *RaffleRepository) sellNumbers(ctx context.Context, raffleID string, numbers []int, sale raffle.SellNumberInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const updateQuery = `
		UPDATE raffle_numbers
		SET status = 'sold',
		    buyer_name = $3,
		    buyer_phone = $4,
		    member_id = $5,
		    sold_by_user_id = $6,
		    sold_at = $7,
		    payment_method = $8
		WHERE raffle_id = $1 AND number = $2 AND status = 'available'
	`

	now := time.Now().UTC()
	for _, number := range numbers {
		commandTag, err := tx.Exec(ctx, updateQuery,
			raffleID, number, sale.BuyerName, sale.BuyerPhone,
			sale.MemberID, sale.SoldByUserID, now, sale.PaymentMethod,
		)
		if err != nil {
			return fmt.Errorf("sell number %d: %w", number, err)
		}
		if commandTag.RowsAffected() == 0 {
			return domain.ErrRaffleNumberAlreadySold
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sell tx: %w", err)
	}
	return nil
}

func (r *RaffleRepository) Draw(ctx context.Context, raffleID string, winningNumber int) (domain.RaffleSummary, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const numberQuery = `
		SELECT buyer_name, status
		FROM raffle_numbers
		WHERE raffle_id = $1 AND number = $2
	`
	var buyerName *string
	var status domain.RaffleNumberStatus
	if err := tx.QueryRow(ctx, numberQuery, raffleID, winningNumber).Scan(&buyerName, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RaffleSummary{}, domain.ErrInvalidRaffleNumber
		}
		return domain.RaffleSummary{}, fmt.Errorf("load winning number: %w", err)
	}
	if status != domain.RaffleNumberSold {
		return domain.RaffleSummary{}, domain.ErrWinningNumberNotSold
	}

	const insertDraw = `
		INSERT INTO raffle_draws (raffle_id, winning_number)
		VALUES ($1, $2)
	`
	if _, err := tx.Exec(ctx, insertDraw, raffleID, winningNumber); err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("insert draw: %w", err)
	}

	const updateRaffle = `UPDATE raffles SET status = 'finished' WHERE id = $1`
	if _, err := tx.Exec(ctx, updateRaffle, raffleID); err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("finish raffle: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RaffleSummary{}, fmt.Errorf("commit draw tx: %w", err)
	}

	summary, err := r.GetByID(ctx, raffleID)
	if err != nil {
		return domain.RaffleSummary{}, err
	}
	summary.WinningNumber = &winningNumber
	summary.WinnerName = buyerName
	return summary, nil
}

func (r *RaffleRepository) enrichSummary(ctx context.Context, summary domain.RaffleSummary) (domain.RaffleSummary, error) {
	prizes, err := r.loadPrizes(ctx, summary.ID)
	if err != nil {
		return domain.RaffleSummary{}, err
	}
	summary.Prizes = prizes

	sold, err := r.CountSoldNumbers(ctx, summary.ID)
	if err != nil {
		return domain.RaffleSummary{}, err
	}
	summary.SoldNumbers = sold

	draw, err := r.loadDraw(ctx, summary.ID)
	if err != nil {
		return domain.RaffleSummary{}, err
	}
	if draw != nil {
		summary.WinningNumber = &draw.WinningNumber
		winner, err := r.loadWinnerName(ctx, summary.ID, draw.WinningNumber)
		if err != nil {
			return domain.RaffleSummary{}, err
		}
		summary.WinnerName = winner
	}

	return summary, nil
}

func (r *RaffleRepository) loadPrizes(ctx context.Context, raffleID string) ([]domain.RafflePrize, error) {
	const query = `
		SELECT id, raffle_id, description, position
		FROM raffle_prizes WHERE raffle_id = $1 ORDER BY position ASC
	`
	rows, err := r.pool.Query(ctx, query, raffleID)
	if err != nil {
		return nil, fmt.Errorf("load prizes: %w", err)
	}
	defer rows.Close()

	prizes := []domain.RafflePrize{}
	for rows.Next() {
		var prize domain.RafflePrize
		if err := rows.Scan(&prize.ID, &prize.RaffleID, &prize.Description, &prize.Position); err != nil {
			return nil, fmt.Errorf("scan prize: %w", err)
		}
		prizes = append(prizes, prize)
	}
	return prizes, rows.Err()
}

func (r *RaffleRepository) loadDraw(ctx context.Context, raffleID string) (*domain.RaffleDraw, error) {
	const query = `SELECT id, raffle_id, winning_number, drawn_at FROM raffle_draws WHERE raffle_id = $1`
	var draw domain.RaffleDraw
	err := r.pool.QueryRow(ctx, query, raffleID).Scan(&draw.ID, &draw.RaffleID, &draw.WinningNumber, &draw.DrawnAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load draw: %w", err)
	}
	return &draw, nil
}

func (r *RaffleRepository) loadWinnerName(ctx context.Context, raffleID string, winningNumber int) (*string, error) {
	const query = `SELECT buyer_name FROM raffle_numbers WHERE raffle_id = $1 AND number = $2`
	var buyerName *string
	if err := r.pool.QueryRow(ctx, query, raffleID, winningNumber).Scan(&buyerName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load winner name: %w", err)
	}
	return buyerName, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanRaffleSummaryRow(row scannable) (domain.RaffleSummary, error) {
	var summary domain.RaffleSummary
	err := row.Scan(
		&summary.ID, &summary.Name, &summary.GoalAmount, &summary.PointValue,
		&summary.TotalNumbers, &summary.DrawDate, &summary.Status, &summary.CreatedAt,
	)
	if err != nil {
		return domain.RaffleSummary{}, err
	}
	return summary, nil
}
