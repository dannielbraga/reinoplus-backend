-- +goose Up
ALTER TABLE contributions
    ADD COLUMN contributor_name TEXT,
    ADD COLUMN contributor_phone TEXT,
    ADD COLUMN created_by_user_id UUID REFERENCES users(id);

UPDATE contributions c
SET contributor_name = m.name,
    contributor_phone = m.phone
FROM members m
WHERE c.member_id = m.id;

ALTER TABLE contributions ALTER COLUMN member_id DROP NOT NULL;

-- +goose Down
ALTER TABLE contributions ALTER COLUMN member_id SET NOT NULL;
ALTER TABLE contributions
    DROP COLUMN IF EXISTS created_by_user_id,
    DROP COLUMN IF EXISTS contributor_phone,
    DROP COLUMN IF EXISTS contributor_name;
