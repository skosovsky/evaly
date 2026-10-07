# Root release contract

`make release-patch` invokes `bash scripts/release.sh patch` with the registry's
root-only release inventory. `make release-break` preserves the separate break
policy. No runtime API changes are required for CI parity. contracttest and
integrations/recipes are tested but never published. See [checks.md](checks.md).

Start from a branch with committed tracked changes. Untracked caller files are
preserved and excluded from the isolated candidate. The script reads current
remote root semantic tags (not unpublished local tags), computes the next version,
and requests the existing y/N confirmation. Supply `y` for authorized automation.
It fetches only the selected source commit into a private detached checkout,
formats root go.mod, and commits that exact file only if changed. Source branch,
HEAD/index/files/config/refs remain untouched. Signing identity is copied for the
private commit; lightweight tags explicitly use --no-sign.

After final preparation, `make check` runs on candidate bytes with strict required
prerequisites, including Linux, fresh tests, scripts, generation and candidate
consumers. There is no release→check→release recursion. Candidate module ZIP is
built using official pinned golang.org/x/mod/zip in a disposable tooling module,
not a root dependency; nested modules, vendor and symlinks follow Go's rules.
An isolated file proxy validates ZIP/go.mod and records both h1 sums. The candidate
consumer imports its exact version from this proxy with no evaly replace and
exact supported peer checkouts. Candidate checksum-db bypass is limited to the
unpublished self module; external dependencies retain public verification.

The selected caller source HEAD/fingerprint must also remain unchanged during
validation; concurrent source changes forbid publication and remain preserved.
Immediately before ref creation/push, a byte/mode fingerprint, HEAD and dirty/
all-untracked guards reject changes after validation, including ignored .go files.
Reports and artifacts remain outside the candidate tree. Only then is the
lightweight tag created on the captured SHA and checked against it. That immutable
SHA is pushed to one exact destination ref atomically without force or
fallback. Concurrent version collisions fail; existing public tags are never
rewritten. Nine-digit version bounds prevent arithmetic overflow. Root tag push
runs the shared CI gate once; no full workflows per test-only nested module tag.

After push, `verify_release.py` downloads the exact public module using a fresh
cache, public proxy and sum.golang.org with user bypass settings disabled. Both
sums must match candidate evidence, version must match, and reported origin SHA
must match when supplied. The exact-version public consumer runs semantic smoke,
vet and pinned lint without local replaces; its actual graph is recorded. Proxy
availability has 12 bounded attempts, with 90-second network-command timeouts.
The verifier waits up to 180 polls (10s spacing, bounded API timeout) for the
latest root-tag Go workflow on the exact release SHA, and all its jobs must pass.
A cached old green run or a branch run cannot satisfy verification. Consumer
execution has a 600-second subprocess bound for cold tool caches.

Unknown/failed verification never reports release success. Evidence is retained
under the printed `.evidence` directory. If push fails and remote ref is absent,
the private checkout is removed; retry computes remote version again. If ref is
present or status unknown, the checkout is retained and no refs are deleted.
Postpublish proxy/consumer/CI failure also retains checkout and prints an exact
`verify_release.py VERSION SHA MANIFEST OUTPUT` recovery command. Re-run that
command after resolving availability. If a published implementation requires a
fix, commit it and release another patch through the same target; never rewrite
the failed tag. A successful verification prints version/SHA/evidence and removes
only the temporary checkout. No GitHub Release object is created by this target.

Local safety fixtures: `python3 -m unittest discover -s scripts -p '*_test.py' -v`.
They create temporary Git origins and do not mutate public refs. Full gate and
live public evidence must be recorded separately; fixture success alone is not
publication acceptance.
