#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

printf 'Checking gateway...\n'
curl -fsS "$BASE_URL/health"
printf '\n'

printf 'Registering test customer...\n'
REGISTER_RESPONSE=$(curl -fsS -X POST "$BASE_URL/api/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{"name":"CartNova Customer","email":"customer@example.com","password":"Customer@123"}') || true
printf '%s\n' "$REGISTER_RESPONSE"

printf 'Logging in admin...\n'
LOGIN_RESPONSE=$(curl -fsS -X POST "$BASE_URL/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@cartnova.local","password":"password"}')
ADMIN_TOKEN=$(printf '%s' "$LOGIN_RESPONSE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')

printf 'Creating category...\n'
CATEGORY=$(curl -fsS -X POST "$BASE_URL/api/v1/categories" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Electronics"}') || true
printf '%s\n' "$CATEGORY"
CATEGORY_ID=$(printf '%s' "$CATEGORY" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("id", ""))')

printf 'Creating product...\n'
PRODUCT=$(curl -fsS -X POST "$BASE_URL/api/v1/products" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Wireless Keyboard\",\"description\":\"Mechanical keyboard\",\"price\":2499.00,\"category_id\":$CATEGORY_ID}")
printf '%s\n' "$PRODUCT"
PRODUCT_ID=$(printf '%s' "$PRODUCT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')

printf 'Adding inventory...\n'
curl -fsS -X POST "$BASE_URL/api/v1/inventory" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"product_id\":$PRODUCT_ID,\"quantity\":10}"
printf '\n'

printf 'CartNova smoke test reached the core services successfully.\n'
