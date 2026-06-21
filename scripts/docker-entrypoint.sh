#!/bin/sh
set -e

KEYS_DIR=/app/keys
mkdir -p "$KEYS_DIR"

if [ -n "$JWT_PRIVATE_KEY" ] && [ -n "$JWT_PUBLIC_KEY" ]; then
  printf '%s\n' "$JWT_PRIVATE_KEY" > "$KEYS_DIR/private.pem"
  printf '%s\n' "$JWT_PUBLIC_KEY" > "$KEYS_DIR/public.pem"
elif [ ! -f "$KEYS_DIR/private.pem" ]; then
  openssl genrsa -out "$KEYS_DIR/private.pem" 2048
  openssl rsa -in "$KEYS_DIR/private.pem" -pubout -out "$KEYS_DIR/public.pem"
fi

export JWT_PRIVATE_KEY_PATH="$KEYS_DIR/private.pem"
export JWT_PUBLIC_KEY_PATH="$KEYS_DIR/public.pem"

if [ -n "$PORT" ] && [ -z "$SERVER_PORT" ]; then
  export SERVER_PORT="$PORT"
fi

if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

echo "Running database migrations..."
goose -dir /app/migrations postgres "$DATABASE_URL" up

echo "Starting API on port ${SERVER_PORT:-8080}..."
exec /app/api
