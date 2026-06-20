.PHONY: help up down restart rebuild ps logs logs-all health run test tidy keys migrate-up migrate-down docker-up docker-down docker-reset build install

.DEFAULT_GOAL := help

COMPOSE := docker compose
API_URL := http://localhost:8080

help: ## Lista os comandos disponíveis
	@echo "Reinoplus Backend"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: docker-up ## Sobe backend localmente (Postgres + migrations + API)

docker-up: ## Build e sobe todos os containers em background
	$(COMPOSE) up --build -d
	@echo ""
	@echo "Backend no ar:"
	@echo "  Health: $(API_URL)/health"
	@echo "  API:    $(API_URL)/api"
	@echo ""
	@echo "Login admin: admin@reinoplus.local / admin123"

down: docker-down ## Para os containers

docker-down:
	$(COMPOSE) down

restart: ## Reinicia apenas o container da API
	$(COMPOSE) restart api

rebuild: ## Rebuild da API e sobe de novo (aplica código + migrations)
	$(COMPOSE) up -d --build

ps: ## Status dos containers
	$(COMPOSE) ps

logs: docker-logs ## Logs da API (follow)

docker-logs:
	$(COMPOSE) logs -f api

logs-all: ## Logs de todos os serviços (follow)
	$(COMPOSE) logs -f

health: ## Verifica se a API está respondendo
	@curl -sf $(API_URL)/health | python3 -m json.tool 2>/dev/null || (echo "API indisponível em $(API_URL)" && exit 1)

docker-reset: ## Para containers e apaga o volume do Postgres (reset total)
	$(COMPOSE) down -v

run: ## Roda a API localmente com go run (requer Postgres e .env)
	go run ./cmd/api

build: ## Compila o binário em bin/api
	go build -o bin/api ./cmd/api

test: ## Executa os testes
	go test ./...

tidy: ## Ajusta dependências Go
	go mod tidy

keys: ## Gera par de chaves JWT em keys/
	bash scripts/generate-keys.sh

migrate-up: ## Roda migrations (requer goose e DATABASE_URL no .env)
	goose -dir migrations postgres "$${DATABASE_URL}" up

migrate-down: ## Reverte a última migration
	goose -dir migrations postgres "$${DATABASE_URL}" down

install: keys ## Prepara ambiente local (chaves JWT)
	@test -f .env || cp .env.example .env
	@echo "Copie .env.example para .env se ainda não existir."
