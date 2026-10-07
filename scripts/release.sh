#!/bin/bash
# Root-module release: prepare in an isolated repository, publish one exact ref.
set -euo pipefail

release_type=${1:-}
modules=${2:-}
if [[ "$release_type" != patch && "$release_type" != break ]] || [[ "$modules" != . ]]; then
    echo 'Usage: release.sh {patch|break} . (only the root module is released)' >&2
    exit 1
fi
if [[ ! -f go.mod ]]; then
    echo 'Error: run from the repository root containing go.mod' >&2
    exit 1
fi
export GOWORK=off PYTHONDONTWRITEBYTECODE=1 GOENV=off GOFLAGS=
export GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GONOSUMDB=none GONOPROXY=none GOPRIVATE=
pin=$(python3 -c 'import json; print(json.load(open("checks/registry.json"))["go"])')
export GOTOOLCHAIN="go$pin"
source_root=$(git rev-parse --show-toplevel)
if [[ "$PWD" != "$source_root" ]]; then
    echo 'Error: run from the repository root' >&2
    exit 1
fi
if ! git symbolic-ref --quiet HEAD >/dev/null; then
    echo 'Error: detached source HEAD is unsupported; select a branch first' >&2
    exit 1
fi
if ! git diff --quiet HEAD -- || ! git diff --cached --quiet; then
    echo 'Error: commit or stash tracked changes before releasing' >&2
    exit 1
fi
source_head=$(git rev-parse HEAD)
source_fingerprint=$(python3 scripts/checks.py fingerprint)
remote=$(git remote get-url --push origin)
# Relative local remotes must retain their meaning after changing directory.
case "$remote" in
    /*|*:* ) ;;
    * ) remote="$source_root/$remote" ;;
esac
remote_tags=$(git ls-remote --refs "$remote" 'refs/tags/v*')
latest=$(printf '%s\n' "$remote_tags" | awk '{sub("refs/tags/v", "", $2); if ($2 ~ /^[0-9]+\.[0-9]+\.[0-9]+$/) print $2}' | sort -t . -k1,1n -k2,2n -k3,3n | tail -n 1)
latest=${latest:-0.0.0}
IFS=. read -r major minor patch <<< "$latest"
# Bound components before shell arithmetic, including its signed overflow.
for component in "$major" "$minor" "$patch"; do
    if [[ ${#component} -gt 9 ]]; then
        echo 'Error: version component exceeds supported 9-digit range' >&2
        exit 1
    fi
done
major=$((10#$major)); minor=$((10#$minor)); patch=$((10#$patch))
if [[ "$release_type" == patch ]]; then
    patch=$((patch + 1))
elif [[ "$major" == 0 ]]; then
    minor=$((minor + 1)); patch=0
else
    major=$((major + 1)); minor=0; patch=0
fi
if (( major > 999999999 || minor > 999999999 || patch > 999999999 )); then
    echo 'Error: next version exceeds supported 9-digit range' >&2
    exit 1
fi
version="v$major.$minor.$patch"
echo "Current remote version: v$latest"
echo "New version: $version ($release_type)"
read -r -p "Proceed with release $version? [y/N] " reply
if [[ "$reply" != y && "$reply" != Y ]]; then
    echo 'Aborted'
    exit 1
fi

release_dir=$(mktemp -d "${TMPDIR:-/tmp}/evaly-release.XXXXXXXX")
keep_checkout=false
cleanup() {
    if [[ "$keep_checkout" == false ]]; then
        rm -rf "$release_dir"
    else
        echo "Recovery checkout retained: $release_dir" >&2
    fi
}
trap cleanup EXIT
# Fetch only the selected commit, with no tags or source working files.
git init --quiet "$release_dir"
git -C "$release_dir" fetch --quiet --no-tags "$source_root" "$source_head"
git -C "$release_dir" checkout --quiet --detach FETCH_HEAD
# Copy resolved identity/signing options; no mutation of the source config.
for key in user.name user.email user.signingkey commit.gpgsign gpg.format gpg.program; do
    if value=$(git config --get "$key"); then
        git -C "$release_dir" config "$key" "$value"
    fi
done
cd "$release_dir"
# The sole generated release file is root go.mod; no dependency rewriting or
# module discovery. go mod edit is portable across Linux and macOS.
go mod edit -fmt go.mod
git add -- go.mod
if ! git diff --cached --quiet; then
    git commit --quiet -m "chore: release $version"
fi
release_head=$(git rev-parse HEAD)
export EVALY_CANDIDATE_VERSION="$version"
# The full gate checks the final candidate, never the caller working tree.
# Reports/proxy live outside its tree and cannot become public inputs.
evidence_dir="$release_dir.evidence"
mkdir -p "$evidence_dir"
export EVALY_CHECK_REPORT="$evidence_dir/summary.json"
initial=$(python3 scripts/checks.py fingerprint)
make check > "$evidence_dir/gate.log" 2>&1 || {
    cat "$evidence_dir/gate.log" >&2
    echo 'Error: required candidate gate failed; publication forbidden' >&2
    exit 1
}
python3 scripts/artifacts.py "$version" "$evidence_dir/proxy" > "$evidence_dir/artifact.json"
final=$(python3 scripts/checks.py fingerprint)
if [[ "$initial" != "$final" ]] || ! git diff --quiet HEAD -- || [[ -n $(git ls-files --others) ]]; then
    echo 'Error: candidate changed after check; publication forbidden' >&2
    exit 1
fi
if [[ $(git -C "$source_root" rev-parse HEAD) != "$source_head" || $(python3 "$source_root/scripts/checks.py" fingerprint) != "$source_fingerprint" ]] || ! git -C "$source_root" diff --quiet HEAD --; then
    echo 'Error: committed source changed after selection; publication forbidden' >&2
    exit 1
fi
# Ref creation is after every prepublication check.
git tag --no-sign "$version" "$release_head"
if [[ $(git rev-parse "refs/tags/$version") != "$release_head" ]]; then
    echo 'Error: candidate tag changed before push; publication forbidden' >&2
    exit 1
fi
if [[ $(git rev-parse HEAD) != "$release_head" || $(python3 scripts/checks.py fingerprint) != "$initial" ]]; then
    echo 'Error: candidate changed before push; publication forbidden' >&2
    exit 1
fi
keep_checkout=true
# Atomic even though there is currently one ref: never fall back to --tags.
if git push --atomic "$remote" "$release_head:refs/tags/$version"; then
    echo "Published refs/tags/$version at $release_head"
    # Published does not mean verified. Retain recovery on proxy/consumer/CI failure.
    if ! python3 scripts/verify_release.py "$version" "$release_head" "$evidence_dir/proxy/manifest.json" "$evidence_dir/public.json"; then
        echo "Error: published but verification failed; retained evidence: $evidence_dir" >&2
        echo "Recovery: cd $release_dir && python3 scripts/verify_release.py $version $release_head $evidence_dir/proxy/manifest.json $evidence_dir/public.json" >&2
        exit 1
    fi
    echo "Release verified: $version at $release_head; evidence: $evidence_dir"
    keep_checkout=false
else
    echo "Publication command failed for refs/tags/$version (expected $release_head)" >&2
    if status=$(git ls-remote --refs "$remote" "refs/tags/$version"); then
        if [[ -z "$status" ]]; then
            echo 'Remote ref is absent; nothing published. Retry uses remote version again.' >&2
            keep_checkout=false
        else
            echo "Observed remote ref: $status" >&2
            echo 'Inspect retained checkout and remote ref before retrying; no refs were deleted.' >&2
        fi
    else
        echo 'Remote status unknown. Inspect remote ref before retrying; no refs were deleted.' >&2
    fi
    exit 1
fi
