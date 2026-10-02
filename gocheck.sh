#!/bin/sh
# Compile (and optionally vet/test) the Go server in a container, because Go is
# not installed on the dev machine. Build and module caches live in named
# volumes so repeat runs are fast.
# Usage: ./gocheck.sh [build|vet|test] ...   default: build
set -e
cd "$(dirname "$0")"
# MSYS_NO_PATHCONV stops Git Bash rewriting the container-side paths.
MSYS_NO_PATHCONV=1 exec docker run --rm --dns 8.8.8.8 --dns 1.1.1.1 \
  -v "$(pwd)/server:/src" \
  -v seatsurfing-gomod:/go/pkg/mod \
  -v seatsurfing-gobuild:/root/.cache/go-build \
  -w /src \
  docker.io/library/golang:1.27-bookworm \
  go "${@:-build}" ./...
