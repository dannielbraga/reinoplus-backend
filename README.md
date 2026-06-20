# Reinoplus — Backend

Backend MVP em Go (Clean Architecture) para o app Reinoplus.

## Stack

- Go + Clean Architecture (`domain` / `usecase` / `repository` / `handler`)
- PostgreSQL + pgx/v5
- chi (HTTP router)
- goose (migrations)
- JWT RS256 (`golang-jwt`)
- zap (logging estruturado)

## Setup

### Docker (recomendado — um comando)

```bash
cd reinoplus-backend
docker compose up --build -d
# ou: make docker-up
```

Isso sobe:
1. **Postgres** (porta host `5433` → container `5432`)
2. **Migrations** (goose) aplicadas automaticamente
3. **API** na porta `8080`

Verificar:

```bash
curl http://localhost:8080/health
docker compose logs -f api
```

Parar:

```bash
docker compose down
# remover volume do banco: docker compose down -v
```

No frontend Expo, use `EXPO_PUBLIC_API_URL=http://localhost:8080/api` (ou o IP da máquina na rede local, se testar no celular).

### Setup local (sem Docker)

```bash
cd reinoplus-backend
cp .env.example .env
bash scripts/generate-keys.sh
go mod tidy
```

### Banco de dados

```bash
# com Postgres rodando localmente
createdb reinoplus
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir migrations postgres "$DATABASE_URL" up
```

### Rodar API

```bash
go run ./cmd/api
```

Health check: `GET http://localhost:8080/health`

Base da API consumida pelo frontend: `http://localhost:8080/api`

## Usuário seed

- **E-mail:** `admin@reinoplus.local`
- **Senha:** `admin123`

## Estrutura

```
cmd/api/                 # bootstrap
internal/
  config/                # config tipada via .env
  domain/                # entidades + erros de domínio
  usecase/               # regras de negócio
  repository/postgres/   # pgx
  handler/               # HTTP + DTOs
  middleware/            # JWT, CORS, logging, recovery
  auth/jwt/              # tokens RS256
migrations/              # goose
```

## Endpoints MVP

### Auth (público)
- `POST /api/auth/login`
- `POST /api/auth/refresh`

### Auth (protegido)
- `GET /api/auth/me`
- `POST /api/auth/logout`

### Demais rotas (JWT obrigatório)
- Members, Campaigns, Raffles conforme prompt
- Aliases para o frontend Expo:
  - `GET /api/raffles/active`
  - `GET /api/raffles/:id/tickets` (alias de `/numbers`)
  - `POST /api/raffles/:id/sales` (venda em lote)
  - `GET /api/notices` (stub vazio no MVP)

## Testes

```bash
go test ./...
```

## Regras implementadas

- Regra de negócio no usecase; handler só valida/parsing
- Repositórios como interfaces no usecase
- Escritas financeiras transacionais (`pgx.Tx`)
- Totais calculados via agregação SQL (sem contadores desnormalizados)
- Erros de domínio mapeados para HTTP no handler
