#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KEYS_DIR="${ROOT_DIR}/keys"

mkdir -p "${KEYS_DIR}"
openssl genrsa -out "${KEYS_DIR}/private.pem" 2048
openssl rsa -in "${KEYS_DIR}/private.pem" -pubout -out "${KEYS_DIR}/public.pem"

echo "RSA keys generated in ${KEYS_DIR}"
