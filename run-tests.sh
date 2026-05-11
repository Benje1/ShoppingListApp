#!/usr/bin/env bash
# run-tests.sh — run the test suite locally, with or without a Dockerised DB.
#
# Usage:
#   ./run-tests.sh           # unit tests only (no database required)
#   ./run-tests.sh --db      # spin up Postgres in Docker, run tests, tear it down

set -euo pipefail

DB_URL="postgres://shopping:shopping@localhost:5432/shopping_test?sslmode=disable"

run_unit() {
  echo "▶ Running unit tests (no database)…"
  go test ./...
  echo "✓ All tests passed."
}

run_with_db() {
  echo "▶ Starting ephemeral Postgres via Docker Compose…"
  docker compose up -d --wait

  # Ensure the DB is torn down even if the tests fail.
  trap 'echo "▶ Stopping containers…"; docker compose down' EXIT

  echo "▶ Running tests with DATABASE_URL set…"
  DATABASE_URL="$DB_URL" go test ./...
  echo "✓ All tests passed."
}

case "${1:-}" in
  --db)
    run_with_db
    ;;
  "")
    run_unit
    ;;
  *)
    echo "Unknown option: $1"
    echo "Usage: $0 [--db]"
    exit 1
    ;;
esac
