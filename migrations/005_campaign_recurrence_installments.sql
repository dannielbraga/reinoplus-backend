-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN is_recurring BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN recurrence_interval TEXT CHECK (
        recurrence_interval IS NULL OR recurrence_interval IN ('biweekly', 'monthly', 'bimonthly', 'quarterly')
    ),
    ADD COLUMN duration_months INT CHECK (duration_months IS NULL OR duration_months > 0);

ALTER TABLE contributions
    ADD COLUMN installment_number INT CHECK (installment_number IS NULL OR installment_number > 0);

-- +goose Down
ALTER TABLE contributions DROP COLUMN installment_number;

ALTER TABLE campaigns
    DROP COLUMN is_recurring,
    DROP COLUMN recurrence_interval,
    DROP COLUMN duration_months;
