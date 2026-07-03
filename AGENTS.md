# Reinoplus Backend

Backend MVP em Go (Clean Architecture) para o app Reinoplus — sistema de gestão de campanhas, rifas e membros para igrejas.

## Stack

- **Go 1.25** — linguagem
- **chi v5** — HTTP router (leve, idiomatico)
- **pgx/v5** — driver PostgreSQL nativo com pool de conexões
- **goose** — migrações SQL versionadas
- **JWT RS256** (golang-jwt/v5) — autenticação com par RSA 2048
- **zap** — logging estruturado
- **bcrypt** (golang.org/x/crypto) — hash de senhas
- **godotenv** — carregamento de .env

## Arquitetura: Clean Architecture (Ports & Adapters)

```
cmd/api/main.go         → bootstrap (logger, config, pool, repos, services, router, server)
internal/
  config/               → config tipada via environment
  domain/               → entidades + erros de domínio (não dependem de nada)
  usecase/              → regras de negócio (dependem de interfaces)
    auth/ service.go
    campaign/ service.go
    member/ service.go
    raffle/ service.go
  repository/postgres/  → implementações concretas (pgx)
  handler/              → HTTP handlers + DTOs + router setup
  middleware/           → JWT auth, CORS, logging, recovery
  auth/jwt/             → geração/validação de tokens RS256
  httputil/             → WriteJSON, WriteError, DecodeJSON, MapDomainError
  textutil/             → TitleCaseName (formatação de nomes)
migrations/             → SQL migrations (goose)
```

## Fluxo de dados

```
Handler → UseCase (via interface) → Repository (PostgreSQL) → Domain
```

## Entidades de domínio (internal/domain/)

- `User` — id, name, email, password_hash, role (admin|member), member_id
- `Member` — id, name, phone, address, birth_date
- `Campaign` — id, name, description, goal_amount, dates, status, is_recurring
- `Contribution` — id, campaign_id, contributor info, amount, payment_method, is_recurring
- `Raffle` — id, name, goal_amount, point_value, total_numbers, draw_date, status
- `RafflePrize` — id, raffle_id, description, position
- `RaffleNumber` — id, raffle_id, number, status (available|sold), buyer info
- `RaffleDraw` — id, raffle_id, winning_number, drawn_at

## Rotas da API

| Grupo | Endpoints |
|-------|-----------|
| Health | `GET /health` |
| Auth | `POST /api/auth/login` `POST /api/auth/register` `POST /api/auth/refresh` `POST /api/auth/logout` `GET /api/auth/me` `GET/PUT /api/auth/profile` |
| Members | `GET /api/members` `GET /api/members?search=` `GET /api/members/:id` `POST /api/members` `PUT /api/members/:id` |
| Campaigns | `GET /api/campaigns` `POST /api/campaigns` `GET /api/campaigns/:id` `PUT /api/campaigns/:id` `DELETE /api/campaigns/:id` `PATCH /api/campaigns/:id/status` `GET /api/campaigns/:id/contributions` `POST /api/campaigns/:id/contributions` |
| Raffles | `GET /api/raffles` `POST /api/raffles` `GET /api/raffles/active` `GET /api/raffles/:id` `PUT /api/raffles/:id` `DELETE /api/raffles/:id` `GET /api/raffles/:id/tickets` `POST /api/raffles/:id/sales` `POST /api/raffles/:id/draw` |
| Notices | `GET /api/notices` (stub vazio) |

**Aliases para o frontend:** `GET /api/raffles/active`, `GET /api/raffles/:id/tickets` (alias de `/numbers`), `POST /api/raffles/:id/sales` (venda em lote).

## Regras de negócio implementadas

- Regra no usecase; handler só valida/parsing
- Repositórios como interfaces no usecase
- Escritas financeiras transacionais (pgx.Tx)
- Totais calculados via agregação SQL (sem contadores desnormalizados)
- Erros de domínio mapeados para HTTP no handler
- 22 erros de domínio: ErrNotFound, ErrUnauthorized, ErrInvalidCredentials, ErrConflict, ErrValidation, ErrRaffleNumberAlreadySold, ErrCampaignHasContributions, etc.

## Como rodar

```bash
# Docker (recomendado)
make up          # docker compose up --build -d

# Local
make install     # gera chaves JWT, copia .env
make run         # go run ./cmd/api

# Testes
make test        # go test ./...

# Migrations
make migrate-up  # goose up
make migrate-down # goose down
```

**Seed:** admin@reinoplus.local / admin123

## Banco de dados

- PostgreSQL 16
- Porta host: 5433 (container: 5432)
- 5 migrations sequenciais (001 a 005)
- Tabelas: members, users, campaigns, contributions, raffles, raffle_prizes, raffle_numbers, raffle_draws

## Testes

```bash
go test ./...
```

Framework: standard library `testing`. Mocks manuais (structs implementando interfaces).
Arquivos de teste: `internal/textutil/names_test.go`, `internal/usecase/auth/service_test.go`, `internal/usecase/campaign/service_test.go`, `internal/usecase/raffle/service_test.go`

## Deploy

Railway via Dockerfile. `railway.toml` configura health check e restart policy.
