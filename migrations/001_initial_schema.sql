-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    birth_date DATE NOT NULL,
    address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_members_name ON members (name);
CREATE INDEX idx_members_phone ON members (phone);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'member')),
    member_id UUID REFERENCES members(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    goal_amount NUMERIC(12, 2) NOT NULL CHECK (goal_amount > 0),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'completed', 'cancelled')) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_date >= start_date)
);

CREATE TABLE contributions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    member_id UUID NOT NULL REFERENCES members(id),
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    payment_method TEXT NOT NULL CHECK (payment_method IN ('cash', 'pix', 'card', 'transfer')),
    contributed_at DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contributions_campaign_id ON contributions (campaign_id);

CREATE TABLE raffles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    goal_amount NUMERIC(12, 2) NOT NULL CHECK (goal_amount > 0),
    point_value NUMERIC(12, 2) NOT NULL CHECK (point_value > 0),
    total_numbers INT NOT NULL CHECK (total_numbers > 0),
    draw_date DATE NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'finished', 'cancelled')) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE raffle_prizes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raffle_id UUID NOT NULL REFERENCES raffles(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    position INT NOT NULL CHECK (position > 0),
    UNIQUE (raffle_id, position)
);

CREATE TABLE raffle_numbers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raffle_id UUID NOT NULL REFERENCES raffles(id) ON DELETE CASCADE,
    number INT NOT NULL CHECK (number > 0),
    status TEXT NOT NULL CHECK (status IN ('available', 'sold')) DEFAULT 'available',
    buyer_name TEXT,
    buyer_phone TEXT,
    member_id UUID REFERENCES members(id),
    sold_by_user_id UUID REFERENCES users(id),
    sold_at TIMESTAMPTZ,
    payment_method TEXT CHECK (payment_method IN ('cash', 'pix', 'card', 'transfer')),
    UNIQUE (raffle_id, number)
);

CREATE INDEX idx_raffle_numbers_raffle_id ON raffle_numbers (raffle_id);
CREATE INDEX idx_raffle_numbers_status ON raffle_numbers (raffle_id, status);

CREATE TABLE raffle_draws (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raffle_id UUID NOT NULL UNIQUE REFERENCES raffles(id),
    winning_number INT NOT NULL,
    drawn_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS raffle_draws;
DROP TABLE IF EXISTS raffle_numbers;
DROP TABLE IF EXISTS raffle_prizes;
DROP TABLE IF EXISTS raffles;
DROP TABLE IF EXISTS contributions;
DROP TABLE IF EXISTS campaigns;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS members;
