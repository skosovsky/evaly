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
git tag --no-sign "$version"
release_head=$(git rev-parse HEAD)
keep_checkout=true
# Atomic even though there is currently one ref: never fall back to --tags.
if git push --atomic "$remote" "refs/tags/$version:refs/tags/$version"; then
    echo "Published refs/tags/$version at $release_head"
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
