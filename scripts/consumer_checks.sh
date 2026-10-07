#!/usr/bin/env bash
# Copy the portable SDK consumer into a disposable directory: never edit checked-in pins.
set -euo pipefail
mode=${1:-published}
root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/evaly-consumer.XXXXXX")
trap 'rm -rf "$work"' EXIT
cp -R "$root/integrations/recipes/." "$work/"
export GOWORK=off
export GOPATH=${GOPATH:-/tmp/evaly-gopath}
export GOCACHE=${GOCACHE:-/tmp/evaly-go-build}
export GOLANGCI_LINT_CACHE=${GOLANGCI_LINT_CACHE:-/tmp/evaly-consumer-lint}
export GOMODCACHE=${CONSUMER_MODCACHE:-$GOPATH/pkg/mod}
case "$mode" in
 published) ;;
 source)
  sources=${2:?source mode requires directory containing SDK checkouts}
  sources=$(cd "$sources" && pwd)
  printf "evaly source: "
  git -C "$root" rev-parse HEAD
  git -C "$root" status --short
  go -C "$work" mod edit -replace "github.com/skosovsky/evaly=$root"
  for sdk in prompty metry; do
   test -f "$sources/$sdk/go.mod"
   go -C "$work" mod edit -replace "github.com/skosovsky/$sdk=$sources/$sdk"
   printf '%s source: ' "$sdk"
   git -C "$sources/$sdk" rev-parse HEAD
   # Dirty source is allowed locally but must be reported, never called a pinned clean set.
   git -C "$sources/$sdk" status --short
  done
  ;;
 *) printf 'unsupported mode: %s\n' "$mode" >&2; exit 2 ;;
esac
printf 'consumer mode: %s\n' "$mode"
go -C "$work" mod tidy
# Report complete module identities and replacements actually used.
go -C "$work" list -m all
if [ "$mode" = published ]; then
 replacements=$(go -C "$work" list -m -f '{{if .Replace}}{{.Path}} => {{.Replace.Path}}{{end}}' all)
 if [[ -n "${replacements//[[:space:]]/}" ]]; then
  printf 'published matrix must not use replacements\n' >&2; exit 1
 fi
fi
(cd "$work" && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 config verify)
(cd "$work" && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 fmt --diff)
(cd "$work" && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run --allow-serial-runners --max-issues-per-linter=0 --max-same-issues=0)
go -C "$work" vet ./...
go -C "$work" test -race -count=1 -v ./...
