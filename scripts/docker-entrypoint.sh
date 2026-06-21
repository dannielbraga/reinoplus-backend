#!/bin/sh
set -eu

KEYS_DIR=/app/keys
mkdir -p "$KEYS_DIR"

write_pem() {
  # Suporta PEM multilinha e \n escapado (comum em variáveis do Railway)
  printf '%b' "$1" > "$2"
}

if [ -n "${JWT_PRIVATE_KEY:-}" ] && [ -n "${JWT_PUBLIC_KEY:-}" ]; then
  write_pem "$JWT_PRIVATE_KEY" "$KEYS_DIR/private.pem"
  write_pem "$JWT_PUBLIC_KEY" "$KEYS_DIR/public.pem"
elif [ ! -f "$KEYS_DIR/private.pem" ]; then
  echo "Generating JWT keys..."
  openssl genrsa -out "$KEYS_DIR/private.pem" 2048
  openssl rsa -in "$KEYS_DIR/private.pem" -pubout -out "$KEYS_DIR/public.pem"
fi

export JWT_PRIVATE_KEY_PATH="$KEYS_DIR/private.pem"
export JWT_PUBLIC_KEY_PATH="$KEYS_DIR/public.pem"

# Railway injeta PORT; garante que a API escute na porta correta
if [ -n "${PORT:-}" ]; then
  export SERVER_PORT="$PORT"
fi

if [ -z "${DATABASE_URL:-}" ]; then
  echo "ERROR: DATABASE_URL is not set. Link the Postgres service to the API on Railway." >&2
  exit 1
fi

# Garante SSL em conexões remotas públicas (Railway/Neon)
case "$DATABASE_URL" in
  *sslmode=*|*railway.internal*|*@postgres:*|*localhost*|*127.0.0.1* ) ;;
  * )
    if printf '%s' "$DATABASE_URL" | grep -q '?'; then
      export DATABASE_URL="${DATABASE_URL}&sslmode=require"
    else
      export DATABASE_URL="${DATABASE_URL}?sslmode=require"
    fi
    ;;
esac

echo "DATABASE_URL configured (host hidden)"
echo "Running database migrations..."

attempt=1
max_attempts=30
while [ "$attempt" -le "$max_attempts" ]; do
  if goose -dir /app/migrations postgres "$DATABASE_URL" up; then
    echo "Migrations applied successfully."
    break
  fi

  echo "Migration attempt ${attempt}/${max_attempts} failed — database may still be starting..."
  if [ "$attempt" -eq "$max_attempts" ]; then
    echo "ERROR: migrations failed after ${max_attempts} attempts. Check DATABASE_URL and Postgres logs." >&2
    exit 1
  fi

  attempt=$((attempt + 1))
  sleep 3
done

echo "Starting API on port ${SERVER_PORT:-8080}..."
exec /app/api
