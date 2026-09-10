#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
project="wishlist-e2e-$(date +%s)-$$"
compose=${COMPOSE:-docker compose}
export PORT=0 DB_PORT=0
export JWT_SECRET=
export JWT_EXPIRY_MINUTES=60

cleanup() {
    $compose --env-file /dev/null -p "$project" down -v --remove-orphans
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

$compose --env-file /dev/null -p "$project" up -d --build --wait --wait-timeout 90
secret_digest=$($compose --env-file /dev/null -p "$project" exec -T app sha256sum /app/state/jwt-secret)
$compose --env-file /dev/null -p "$project" up -d --force-recreate --no-deps --wait --wait-timeout 90 app
recreated_digest=$($compose --env-file /dev/null -p "$project" exec -T app sha256sum /app/state/jwt-secret)
if [ "$secret_digest" != "$recreated_digest" ]; then
    echo "JWT secret changed after container recreation" >&2
    exit 1
fi
address=$($compose --env-file /dev/null -p "$project" port app 8080)
db_address=$($compose --env-file /dev/null -p "$project" port db 5432)
WISHLIST_TEST_DATABASE_URL="postgres://postgres:postgres@$db_address/wishlist?sslmode=disable" \
    go test -race -count=1 -v ./internal/repository
WISHLIST_E2E_URL="http://$address" go test -race -count=1 -v ./tests/e2e
