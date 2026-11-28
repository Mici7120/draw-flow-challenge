#!/usr/bin/env bash
#
# verify.sh — quality and coverage gate for the draw-flow-challenge Go backend.
#
# Usage:
#   cd backend
#   ./scripts/verify.sh
#
# Environment:
#   MIN_COVERAGE (default: 80.0) — minimum statement coverage percentage per core package.
#
# Enforces: gofmt, go vet, go build, go test -race, and >= MIN_COVERAGE% on core packages.
# Optional tools (staticcheck, golangci-lint, govulncheck) run when installed and emit [skip] otherwise.

set -euo pipefail

MIN_COVERAGE="${MIN_COVERAGE:-80.0}"
CORE_PACKAGES=(
  "./internal/program"
  "./internal/codegen"
  "./internal/sandbox"
  "./internal/store"
  "./internal/service"
  "./internal/httpapi"
)

if [ -t 1 ]; then
  YELLOW=$'\033[33m'; RED=$'\033[31m'; GREEN=$'\033[32m'; RESET=$'\033[0m'
else
  YELLOW=''; RED=''; GREEN=''; RESET=''
fi

failed=0
have() { command -v "$1" >/dev/null 2>&1; }
warn() { printf '%s[skip]%s %s\n' "$YELLOW" "$RESET" "$*"; }
fail() { printf '%s[fail]%s %s\n' "$RED" "$RESET" "$*"; failed=1; }
ok()   { printf '%s[ ok ]%s %s\n' "$GREEN" "$RESET" "$*"; }
info() { printf -- '----- %s\n' "$*"; }

if [ ! -f go.mod ]; then
  printf '%serror:%s no go.mod in %s — run from backend/ module root.\n' "$RED" "$RESET" "$(pwd)" >&2
  exit 2
fi

info "gofmt"
fmt_out="$(gofmt -l .)"
if [ -n "$fmt_out" ]; then
  fail "unformatted Go files (run: gofmt -w .):"
  printf '%s\n' "$fmt_out"
else
  ok "gofmt clean"
fi

info "go vet"
if go vet ./...; then
  ok "go vet clean"
else
  fail "go vet reported issues"
fi

info "go build"
if go build ./...; then
  ok "go build clean"
else
  fail "go build failed"
fi

info "staticcheck"
if have staticcheck; then
  if staticcheck ./...; then ok "staticcheck clean"; else fail "staticcheck reported issues"; fi
else
  warn "staticcheck not installed"
fi

info "golangci-lint"
if have golangci-lint; then
  if golangci-lint run; then ok "golangci-lint clean"; else fail "golangci-lint reported issues"; fi
else
  warn "golangci-lint not installed"
fi

info "go test -race + coverage gate (>= ${MIN_COVERAGE}%)"
cover_file="$(mktemp)"
trap 'rm -f "$cover_file"' EXIT

test_flags=(-race -covermode=atomic "-coverprofile=$cover_file")
cgo_enabled="$(go env CGO_ENABLED 2>/dev/null || echo 0)"
cgo_cc="$(go env CC 2>/dev/null || echo cc)"
if [ "$cgo_enabled" != "1" ] || ! have "$cgo_cc"; then
  warn "cgo or C compiler unavailable; running without -race"
  test_flags=(-covermode=atomic "-coverprofile=$cover_file")
fi

test_out="$(go test "${test_flags[@]}" ./... 2>&1)" && test_status=0 || test_status=$?
printf '%s\n' "$test_out"
if [ "$test_status" -ne 0 ]; then
  fail "go test failed"
else
  ok "go test passed"
  # Verify coverage threshold per core package that exists on disk.
  checked_pkgs=0
  for pkg in "${CORE_PACKAGES[@]}"; do
    pkg_dir="${pkg#./}"
    if [ ! -d "$pkg_dir" ]; then
      continue
    fi
    checked_pkgs=$((checked_pkgs + 1))
    pkg_out="$(go test -cover "$pkg" 2>&1)" || { fail "coverage run failed for $pkg"; continue; }
    pct="$(printf '%s\n' "$pkg_out" | awk '/coverage:/ { for (i=1; i<=NF; i++) if ($i == "coverage:") { gsub(/%/, "", $(i+1)); print $(i+1); exit } }')"
    if [ -z "$pct" ]; then
      fail "could not parse coverage percentage for $pkg"
      continue
    fi
    if awk -v p="$pct" -v min="$MIN_COVERAGE" 'BEGIN { exit !(p + 0 >= min + 0) }'; then
      ok "$pkg coverage ${pct}% >= ${MIN_COVERAGE}%"
    else
      fail "$pkg coverage ${pct}% < ${MIN_COVERAGE}% floor"
    fi
  done
  if [ "$checked_pkgs" -eq 0 ]; then
    warn "no core packages exist yet; coverage floor will enforce once packages are added"
  fi
fi

info "govulncheck"
if have govulncheck; then
  if govulncheck ./...; then ok "govulncheck clean"; else fail "govulncheck found vulnerabilities"; fi
else
  warn "govulncheck not installed"
fi

echo
if [ "$failed" -ne 0 ]; then
  printf '%sFAIL:%s one or more checks failed.\n' "$RED" "$RESET"
  exit 1
fi
printf '%sPASS:%s all active checks and coverage thresholds passed.\n' "$GREEN" "$RESET"
