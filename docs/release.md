# Root release contract

`make release-patch` and `make release-break` run validation before executing
`bash scripts/release.sh {patch|break} .`. The only release target is the root
module. The script requires Bash, Git and Go on Linux or macOS; editing uses
`go mod edit -fmt`, without BSD/GNU sed dependencies. Local fixtures have been
executed on macOS; Linux execution is not claimed by that result.

Start from a branch at the repository root with no tracked staged/unstaged changes.
Detached source HEAD is rejected before preparation. Untracked files are allowed:
the isolated repository fetches only the selected commit, no source working files
or tags. Source HEAD, index, files, local refs and Git configuration are preserved
on success and failure. Preparation formats only root `go.mod`, staging that exact
file; it does not rewrite dependencies, discover submodules or stage unrelated
files. If formatting changes it, the private commit uses `chore: release VERSION`.
Resolved source user identity and commit signing settings are copied for that
commit; global Git settings continue to apply. The script uses the push URL of
`origin`, including resolution of relative local URLs before changing directory.

Version selection uses published root semantic tags on that remote, excluding
submodule tags and unpublished local versions. Without a version the base is
v0.0.0. Patch increments patch; break increments minor before v1 and major from v1
onward, resetting lesser components. Components are bounded to nine decimal digits
to prevent shell arithmetic overflow. A concurrent release collision fails the
non-forced push; inspect remote status before retrying.

The script creates a lightweight tag with `--no-sign`, overriding automatic global
tag signing so that the exact tag ref is the reported commit. It pushes exactly
`refs/tags/VERSION:refs/tags/VERSION` using `--atomic`. There is no fallback when
the remote lacks atomic capability. No working branch, existing tag or other
module ref is published. The command does not create a GitHub Release.

Preparation or tag failure removes the temporary checkout. A successful push
reports the ref and expected commit and removes the checkout. After a failed push,
the script queries the exact remote ref:

- Ref absent: reports no publication and removes temporary state. Retrying chooses
  from remote versions again, without an unpublished source tag skipping a version.
- Ref present: reports its observed object ID and retains the checkout, even if
  it matches the intended commit. Inspect both before retrying: transport failure
  is not evidence that publication failed.
- Status unavailable: explicitly reports unknown publication and retains checkout.

Recovery output names the retained path and intended ref/commit. Query
`git ls-remote --refs PUSH_URL refs/tags/VERSION` when connectivity returns and
inspect the retained checkout. Remove the printed temporary directory only after
resolving publication status. The script never deletes remote tags to recover.
Interruption after push starts can leave the same uncertain state; the EXIT trap
retains the checkout while publication may have occurred.

Local verification (synthetic markers, temporary repositories and bare origins):

```sh
python3 scripts/release_test.py
python3 scripts/release_test.py --baseline
bash -n scripts/release.sh
```

The baseline mode obtains the reviewed script from `76c224a`, reproducing F01
(untracked content and unrelated tag publication) and F02 (detached source and
unpublished local version after rejection). It does not publish to a real remote.
Current fixtures verify the new behavior, including preparation/tag failure,
rejected push/retry, source-state preservation, root-only refs, detached policy
and ambiguous publication recovery. Python is only a test dependency.
