---
name: database
description: Migrações e banco de dados PostgreSQL do Reinoplus
---

# Banco de Dados — Reinoplus Backend

## Stack

- PostgreSQL 16
- Driver: pgx/v5 com pgxpool
- Migrations: goose (SQL versionado)

## Pool de conexões

`internal/repository/postgres/pool.go` — `NewPool(ctx, databaseURL)`:
- Usa `pgxpool.ParseConfig` + `pgxpool.NewWithConfig`
- Verifica conectividade com `Ping`

## Migrations

5 migrations sequenciais em `migrations/`:

| # | Arquivo | Descrição |
|---|---------|-----------|
| 001 | `001_initial_schema.sql` | Schema inicial: members, users, campaigns, contributions, raffles, raffle_prizes, raffle_numbers, raffle_draws |
| 002 | `002_seed_admin.sql` | Seed: admin user + member |
| 003 | `003_contributions_free_text.sql` | contributor_name, contributor_phone text livre; member_id opcional |
| 004 | `004_contribution_recurring_payment.sql` | is_recurring, recurrence_interval, is_paid em contributions |
| 005 | `005_campaign_recurrence_installments.sql` | is_recurring em campaigns; installment_number em contributions |

```bash
make migrate-up       # goose up
make migrate-down     # goose down
```

## Tabelas principais

- **members** — id (UUID PK), name, phone, birth_date, address, created_at
- **users** — id (UUID PK), name, email (UNIQUE), password_hash, role (admin|member), member_id (FK), created_at
- **campaigns** — id (UUID PK), name, description, goal_amount, start_date, end_date, status, is_recurring, recurrence_interval, duration_months, created_at
- **contributions** — id (UUID PK), campaign_id (FK), member_id (FK opcional), contributor_name, contributor_phone, amount, payment_method, contributed_at, is_recurring, recurrence_interval, is_paid, installment_number, created_by_user_id (FK), created_at
- **raffles** — id (UUID PK), name, goal_amount, point_value, total_numbers, draw_date, status, created_at
- **raffle_prizes** — id (UUID PK), raffle_id (FK CASCADE), description, position (UNIQUE per raffle)
- **raffle_numbers** — id (UUID PK), raffle_id (FK CASCADE), number, status (available|sold), buyer_name, buyer_phone, member_id, sold_by_user_id, sold_at, payment_method (UNIQUE raffle_id+number)
- **raffle_draws** — id (UUID PK), raffle_id (FK UNIQUE), winning_number, drawn_at

## Operações transacionais

- `RegisterUserWithMember` — cria member + user em uma tx
- `CreateContribution` — insere contribuição com busca de nomes
- `Create` (raffle) — cria raffle + prizes + numbers em tx
- `sellNumbers` — venda atômica de números (verifica disponibilidade)
- `Draw` — sorteio atômico (valida, insere draw, atualiza status)
- `UpdateUserProfile` — atualiza user + member em tx

## Convenções

- UUIDs gerados via `gen_random_uuid()` (Postgres native)
- Timestamps com `NOW()`
- Constraint UNIQUE em raffle_numbers (raffle_id + number)
- FK CASCADE em raffle_prizes (deleta prêmios ao deletar raffle)
- Todas as escritas financeiras são transacionais
