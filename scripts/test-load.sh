#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
task_results=$(mktemp -d "${TMPDIR:-/tmp}/wishlist-load.XXXXXX")
task_project=$(basename "$task_results" | tr '[:upper:].' '[:lower:]-')
task_compose=${COMPOSE:-docker compose}
export PORT=0 DB_PORT=0 JWT_SECRET=
export JWT_EXPIRY_MINUTES=60 REQUEST_TIMEOUT_SECONDS=10 MIGRATION_TIMEOUT_SECONDS=60

compose() {
    $task_compose --env-file /dev/null -p "$task_project" \
        -f docker-compose.yml -f docker-compose.load.yml "$@"
}

cleanup() {
    task_status=$?
    trap - EXIT
    compose logs --no-color app db > "$task_results/services.log" 2>&1 || true
    if ! compose down -v --remove-orphans; then
        task_status=1
    fi
    printf '\nLoad test artifacts: %s\n' "$task_results"
    exit "$task_status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

{
    date -u '+UTC: %Y-%m-%dT%H:%M:%SZ'
    git rev-parse HEAD
    git status --short
    docker version
    docker info --format 'CPUs={{.NCPU}} Memory={{.MemTotal}} OS={{.OperatingSystem}} Arch={{.Architecture}}'
    printf 'LOAD_RATE=%s LOAD_SECONDS=%s LOAD_CONFLICT_SECONDS=%s\n' \
        "${LOAD_RATE:-100}" "${LOAD_SECONDS:-60}" "${LOAD_CONFLICT_SECONDS:-20}"
} > "$task_results/environment.txt"
# Save the actual inputs, including changes not yet committed.
cp scripts/load-test.js scripts/test-load.sh docker-compose.yml docker-compose.load.yml Dockerfile "$task_results/"
compose up --build -d --wait --wait-timeout 90 app
compose images > "$task_results/images.txt"
task_status=0
compose run --name "$task_project-k6" -T --no-deps k6 > "$task_results/k6.log" 2>&1 || task_status=$?
cat "$task_results/k6.log"
if ! docker cp "$task_project-k6:/tmp/summary.json" "$task_results/summary.json"; then
    echo 'k6 did not produce a summary' >&2
    task_status=1
fi
exit "$task_status"
