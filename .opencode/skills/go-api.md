---
name: go-api
description: Desenvolvimento da API Go com chi, pgx e Clean Architecture
---

# Go API — Reinoplus Backend

## Arquitetura

Clean Architecture (Ports & Adapters):
- `internal/domain/` — entidades e erros (camada mais interna, zero dependências)
- `internal/usecase/` — regras de negócio, depende de interfaces de repositório
- `internal/repository/postgres/` — implementações concretas com pgx
- `internal/handler/` — HTTP handlers + DTOs + router
- `internal/middleware/` — JWT, CORS, logging, recovery

## Convenções

### Handlers
- Handler recebe `http.ResponseWriter, *http.Request` e um service interface
- Parsing de body com `httputil.DecodeJSON`
- Respostas com `httputil.WriteJSON` / `httputil.WriteError`
- Erros de domínio mapeados via `httputil.MapDomainError`

### Use Cases
- Service struct recebe repository interfaces no constructor
- Métodos retornam `(result, error)` — nunca lidam com HTTP
- Regras de negócio VALIDADAS antes de chamar repositório
- Erros de domínio em `internal/domain/errors.go`

### Repositories
- Interface definida no usecase, implementação em `repository/postgres/`
- Operações financeiras usam `pgx.Tx` (transação)
- Pool compartilhado via `pgxpool.Pool` (inicializado em `pool.go`)
- Totais calculados via SQL agregado (sem contadores desnormalizados)

### Testes
- `go test ./...` — framework standard library `testing`
- Mocks manuais (structs que implementam interfaces)
- Testes focados em use cases com injeção de mock repositories

## Fluxo para adicionar novo endpoint

1. Definir entidade + erros em `internal/domain/`
2. Definir interface do repositório no usecase
3. Implementar repositório em `internal/repository/postgres/`
4. Implementar use case em `internal/usecase/`
5. Criar handler em `internal/handler/`
6. Registrar rota em `internal/handler/router.go`
7. Injetar dependências em `cmd/api/main.go`
8. Adicionar migration SQL em `migrations/`
9. Escrever testes

## Comandos frequentes

```bash
make run              # go run ./cmd/api
make test             # go test ./...
make up               # docker compose up --build -d
make migrate-up       # goose up
make migrate-down     # goose down
make keys             # gera par JWT
```

## Seed

- Admin: admin@reinoplus.local / admin123
- Criado na migration `002_seed_admin.sql`
