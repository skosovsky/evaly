#!/usr/bin/env bash
set -euo pipefail

version=${1:?Usage: make release RELEASE_VERSION=v0.1.0}
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    printf 'Invalid release version: %s\n' "$version" >&2
    exit 1
fi

cd "$(git rev-parse --show-toplevel)"
if [[ -n "$(git status --porcelain)" ]]; then
    printf 'Commit all changes before releasing.\n' >&2
    exit 1
fi
branch=$(git branch --show-current)
if [[ "$branch" != main ]]; then
    printf 'Release from main; current branch: %s\n' "$branch" >&2
    exit 1
fi
if [[ ! -f docs/release-notes.md ]]; then
    printf 'Missing docs/release-notes.md.\n' >&2
    exit 1
fi

repository=skosovsky/evaly
remote=$(git remote get-url origin)
case "$remote" in
    https://github.com/skosovsky/evaly|https://github.com/skosovsky/evaly.git|git@github.com:skosovsky/evaly.git) ;;
    *) printf 'Unexpected origin: %s\n' "$remote" >&2; exit 1 ;;
esac

git fetch origin --tags --quiet
if git show-ref --verify --quiet "refs/tags/$version"; then
    printf 'Tag already exists: %s\n' "$version" >&2
    exit 1
fi

git tag -a "$version" -m "evaly $version"
git push --atomic --set-upstream origin main "refs/tags/$version"
gh release create "$version" --repo "$repository" --verify-tag \
    --title "evaly $version" --notes-file docs/release-notes.md
printf 'Published https://github.com/%s/releases/tag/%s\n' "$repository" "$version"
