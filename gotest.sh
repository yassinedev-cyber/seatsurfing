#!/bin/sh
# Run the Go test suite in containers, because Go is not installed on the dev
# machine. Starts a throwaway Postgres on a private network, runs the suite
# against it, then tears it down.
# Usage: ./gotest.sh [go test args...]   default: ./...
set -e
cd "$(dirname "$0")"
NET=seatsurfing-test-net
DB=seatsurfing-test-db
cleanup() {
  docker rm -f "$DB" >/dev/null 2>&1 || true
  docker network rm "$NET" >/dev/null 2>&1 || true
}
trap cleanup EXIT
cleanup
docker network create "$NET" >/dev/null
docker run -d --name "$DB" --network "$NET" \
  -e POSTGRES_PASSWORD=root -e POSTGRES_USER=postgres -e POSTGRES_DB=seatsurfing_test \
  postgres:17-alpine >/dev/null
until docker exec "$DB" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
# -p 1: every package drops and recreates the schema in the one database, so
# they must not run concurrently. The project's own test.sh is sequential for
# the same reason.
MSYS_NO_PATHCONV=1 docker run --rm --network "$NET" --dns 8.8.8.8 --dns 1.1.1.1 \
  -v "$(pwd)/server:/src" \
  -v seatsurfing-gomod:/go/pkg/mod \
  -v seatsurfing-gobuild:/root/.cache/go-build \
  -w /src \
  -e POSTGRES_URL="postgres://postgres:root@$DB/seatsurfing_test?sslmode=disable" \
  -e CRYPT_KEY=rC8REJftxMcdhzTvu9Tk6RqgygBRctZC \
  -e MOCK_SENDMAIL=1 \
  docker.io/library/golang:1.27-bookworm \
  go test -p 1 -count=1 "$@" ./...
