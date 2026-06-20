-- +goose Up
ALTER TABLE contributions
    ADD COLUMN is_recurring BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN recurrence_interval TEXT CHECK (recurrence_interval IS NULL OR recurrence_interval IN ('monthly')),
    ADD COLUMN is_paid BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE contributions
    DROP COLUMN is_recurring,
    DROP COLUMN recurrence_interval,
    DROP COLUMN is_paid;
