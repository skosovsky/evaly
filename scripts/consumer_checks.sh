#!/usr/bin/env bash
# Copy the portable SDK consumer into a disposable directory: never edit checked-in pins.
set -euo pipefail
mode=${1:-published}
root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/evaly-consumer.XXXXXX")
trap 'rm -rf "$work"' EXIT
cp -R "$root/integrations/recipes/." "$work/"
export GOWORK=off GOFLAGS= GOENV=off
if [[ "$mode" != candidate ]]; then
 export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GONOSUMDB=none GONOPROXY=none GOPRIVATE=
fi
export GOTOOLCHAIN="go$(python3 -c 'import json; print(json.load(open("checks/registry.json"))["go"])')"
linter=$(python3 -c 'import json; print(json.load(open("checks/registry.json"))["linter"])')
export GOPATH=${GOPATH:-/tmp/evaly-gopath}
export GOCACHE=${GOCACHE:-/tmp/evaly-go-build}
export GOLANGCI_LINT_CACHE=${GOLANGCI_LINT_CACHE:-/tmp/evaly-consumer-lint}
export GOMODCACHE=${CONSUMER_MODCACHE:-$GOPATH/pkg/mod}
case "$mode" in
 published|public) ;;
 candidate)
  sources=${2:?candidate requires pinned peers}
  sources=$(cd "$sources" && pwd)
  ;;
 source)
  sources=${2:?source mode requires directory containing SDK checkouts}
  sources=$(cd "$sources" && pwd)
  printf "evaly source: "
  git -C "$root" rev-parse HEAD
  git -C "$root" status --short
  go -C "$work" mod edit -replace "github.com/skosovsky/evaly=$root"
  ;;
 *) printf 'unsupported mode: %s\n' "$mode" >&2; exit 2 ;;
esac
if [[ "$mode" = source || "$mode" = candidate ]]; then
  for sdk in prompty metry; do
   test -f "$sources/$sdk/go.mod"
   go -C "$work" mod edit -replace "github.com/skosovsky/$sdk=$sources/$sdk"
   printf '%s source: ' "$sdk"
   git -C "$sources/$sdk" rev-parse HEAD
   # Dirty source is allowed locally but must be reported, never called a pinned clean set.
   git -C "$sources/$sdk" status --short
  done
fi
if [[ "$mode" = candidate || "$mode" = public ]]; then
 go -C "$work" mod edit -require "github.com/skosovsky/evaly@${CONSUMER_VERSION:?exact version required}"
fi
printf 'consumer mode: %s\n' "$mode"
go -C "$work" mod tidy
# Report complete module identities and replacements actually used.
go -C "$work" list -m all
if [[ "$mode" = published || "$mode" = public || "$mode" = candidate ]]; then
 replacements=$(go -C "$work" list -m -f '{{if .Replace}}{{.Path}} => {{.Replace.Path}}{{end}}' all)
 if [[ "$mode" = candidate ]]; then
  replacements=$(printf '%s\n' "$replacements" | sed '/^github.com\/skosovsky\/prompty => /d; /^github.com\/skosovsky\/metry => /d')
 fi
 if [[ -n "${replacements//[[:space:]]/}" ]]; then
  printf 'published matrix must not use replacements\n' >&2; exit 1
 fi
fi
if [[ "$mode" = candidate || "$mode" = public ]]; then
 identity=$(go -C "$work" list -m -f '{{.Version}}' github.com/skosovsky/evaly)
 test "$identity" = "$CONSUMER_VERSION"
 go -C "$work" mod download -json "github.com/skosovsky/evaly@$CONSUMER_VERSION" > "$work/exact.json"
 python3 - "$work/exact.json" "${CONSUMER_EXPECTED_SUM:-}" <<'PYJSON'
import json,sys
record=json.load(open(sys.argv[1]))
if record.get('Error') or not record.get('Sum') or not record.get('GoModSum') or (sys.argv[2] and record['Sum'] != sys.argv[2]):
 raise SystemExit('exact consumer checksum mismatch')
print(json.dumps(record,indent=2))
PYJSON
 go -C "$work" mod verify
fi
# Missing required SDK protocols are explicitly unsupported, never a semantic PASS.
for capability in github.com/skosovsky/prompty.Stream github.com/skosovsky/prompty.CaptureReport github.com/skosovsky/metry/genai.EvaluationRecorder github.com/skosovsky/metry/genai.EvaluationSkipped; do
 package=${capability%.*}
 go -C "$work" list "$package" >/dev/null # resolution/infrastructure failure stays an error
 if ! go -C "$work" doc "$capability" >"$work/capability.log" 2>&1; then
  if ! grep -F "no symbol ${capability##*.} in package $package" "$work/capability.log" >/dev/null; then
   cat "$work/capability.log" >&2; exit 1
  fi
  printf 'UNSUPPORTED dependency API: %s; semantic fixtures were not executed\n' "$capability" >&2
  exit 2
 fi
done
(cd "$work" && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"$linter" config verify)
(cd "$work" && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"$linter" fmt --diff)
(cd "$work" && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"$linter" run --allow-serial-runners --max-issues-per-linter=0 --max-same-issues=0)
go -C "$work" vet ./...
go -C "$work" test -race -count=1 -timeout=30m -v ./...
