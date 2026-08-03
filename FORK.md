# FORK.md — operating this fork

`frd1201/vibing-steampunk` is a **downstream distribution** of
[`oisee/vibing-steampunk`](https://github.com/oisee/vibing-steampunk): own
release line, own pace, but permanently mergeable with upstream. Fixes that are
useful to everyone still go up as pull requests.

Rationale and decisions live in
[`reports/2026-08-03-001-fork-strategy.md`](reports/2026-08-03-001-fork-strategy.md).
**This file is the operational short reference** — the one to keep open.

---

## Setup

```bash
git clone https://github.com/frd1201/vibing-steampunk.git
cd vibing-steampunk
git remote add upstream https://github.com/oisee/vibing-steampunk.git
git fetch upstream --prune

go build -o vsp ./cmd/vsp
```

`go install github.com/frd1201/vibing-steampunk/cmd/vsp@latest` does **not**
work by design: `go.mod` deliberately keeps the upstream module path
`github.com/oisee/vibing-steampunk` so that upstream merges stay conflict-free.
Build from this repo, or use the release artifacts.

---

## The two rules

**1. Anything upstream-worthy branches off `upstream/main`, never off `main`.**
Otherwise the pull request drags this fork's commits along and becomes noise for
the upstream maintainer.

**2. Never cherry-pick, always merge.** A cherry-pick produces an identical
commit under a different SHA, after which `git branch --merged` and
`git log upstream/main..main` no longer tell the truth about what has already
been submitted upstream. The single exception is back-filling a PR for work
that is already on `main` — see below.

---

## Branches

| Prefix | Purpose | Branches off | Merges into |
|---|---|---|---|
| `main` | integration branch, source of all tags | — | — |
| `feat/*`, `fix/*` | own work, upstream-worthy | `upstream/main` | PR upstream **and** merge into `main` |
| `fork-only/*` | deliberately not upstream-worthy | `main` | `main` only |
| `upstream-pr/<n>` | adopting someone else's upstream PR | that PR's head | `main` |
| `sync/upstream-YYYY-MM` | catch branch for an upstream merge | `main` | `main` |
| `probe/pr-<n>` | throwaway trial merge | `main` | deleted |

**Never delete a `feat/*` or `fix/*` branch while its upstream PR is open** —
GitHub auto-closes a pull request when its head branch disappears.

When merging a fork-internal PR on GitHub, always pick **"Create a merge
commit"**. "Squash and merge" is a cherry-pick in disguise and breaks rule 2.

---

## Monthly upstream check

Two minutes, once a month, or whenever GitHub reports activity upstream:

```bash
git fetch upstream --prune
git log --oneline main..upstream/main        # empty? done, nothing to do.
```

If it is not empty, merge through a catch branch rather than straight onto
`main`:

```bash
git switch -c sync/upstream-$(date +%Y-%m) main
git merge upstream/main
go build ./... && go test ./...              # gate: must be green
git switch main && git merge --no-ff sync/upstream-$(date +%Y-%m)
git branch -d sync/upstream-$(date +%Y-%m)
```

---

## Workflow A — your own change

One question decides the branch type: *does this solve a problem every user has,
and is it free of site- or customer-specific detail?*

**Yes — upstream-worthy:**

```bash
git fetch upstream
git switch -c feat/<topic> upstream/main       # not off main!
# ... develop, test ...
git push -u origin feat/<topic>
gh pr create --repo oisee/vibing-steampunk --base main
git switch main && git merge --no-ff feat/<topic>
```

Do **not** wait for the upstream merge — the change goes into `main` right away
so it is available in production here.

Review fixes requested on the upstream PR are pushed to the same branch and come
back with **another** `git merge --no-ff feat/<topic>`. Never by copying the
commit across.

**No — fork-only:**

```bash
git switch -c fork-only/<topic> main
# ... develop, test ...
git switch main && git merge --no-ff fork-only/<topic>
```

Then add a row to *Fork-only changes* below.

---

## Workflow B — adopt an upstream PR

The trigger is always a **concrete problem here** that already has a fix
upstream. Not "that PR looks useful".

```bash
gh pr checkout <n> --repo oisee/vibing-steampunk --branch upstream-pr/<n>

git diff upstream/main...upstream-pr/<n>       # 1. read the diff

git switch -c probe/pr-<n> main                # 2. trial merge, judge conflicts
git merge upstream-pr/<n>

go build ./... && go test ./...                # 3. gate
go test -tags=integration -v ./pkg/adt/        #    against a real system

git switch main                                # 4. adopt, with provenance
git merge --no-ff upstream-pr/<n> \
  -m "Merge upstream PR #<n> (<author>) — <summary>"
git branch -D probe/pr-<n>
```

Then: CHANGELOG entry under *Adopted from upstream*, and a row in the
*Upstream PR decisions* table below.

**If the PR touches code we already changed, decide before merging:** either our
version wins (record it as rejected, with the reason), or theirs wins (roll ours
back, close our PR with a pointer), or both are partly right (new `feat/*`
branch off `upstream/main` combining them, offered upstream as a new PR).

---

## Back-filling a PR

If something upstream-worthy ends up on `main` without a PR, this is the only
place a cherry-pick is correct:

```bash
git switch -c feat/<topic> upstream/main
git cherry-pick <sha>
go build ./... && go test ./...
git push -u origin feat/<topic>
gh pr create --repo oisee/vibing-steampunk --base main
```

The cost is that the change now exists under two SHAs. Rule 1 exists precisely
to avoid ever needing this.

---

## Our open upstream PRs

Keep these branches alive until the PR is closed.

| PR | Branch | Subject | Status |
|---|---|---|---|
| [#120](https://github.com/oisee/vibing-steampunk/pull/120) | `fix/csrf-head-fallback-and-session-type` | CSRF HEAD→GET fallback, secure-cookie fix, `SAP_SESSION_TYPE` | open since 2026-04-23 |
| [#121](https://github.com/oisee/vibing-steampunk/pull/121) | `feat/incl-write-support` | INCL (PROG/I) write support | open since 2026-04-23 |
| [#126](https://github.com/oisee/vibing-steampunk/pull/126) | `fix/search-type-filter-issue-119` | server-side search type filter | open since 2026-05-01 |

All three are already merged into `main` here. If one stays unanswered for about
twelve months, close it with a factual pointer to the fork commit.

---

## Upstream PR decisions

Every adopted or rejected upstream PR gets a row, with the reason. Nothing is
adopted yet.

| PR | Author | Subject | Decision | Reason |
|---|---|---|---|---|
| [#108](https://github.com/oisee/vibing-steampunk/pull/108) | dme007 | deploy session ordering, MODIFICATION_SUPPORT | **adopt** (decided 2026-08-03) | `1bc5804` shows SAP's `IF_ADT_LOCK_RESULT` documents `NoModification` as `CO_MOD_SUPPORT_NOT_NEEDED`, so the guard from `22517d4` was a false positive on customer-namespace objects. Also brings redirect header preservation and `ICMENOSESSION` recovery, which we lack. Merge pending — plan task 5. |
| [#125](https://github.com/oisee/vibing-steampunk/pull/125) | dme007 | skip redundant mutation gate after lock | **superseded** by #108 | same subject area; #108 covers it |
| [#139](https://github.com/oisee/vibing-steampunk/pull/139) | enricoandreoli | program includes as source-bearing objects | **pending** | collides with our #121 |
| [#145](https://github.com/oisee/vibing-steampunk/pull/145) | zooloo303 | reuse an object's open transport instead of 409 | **pending** | adjacent to our write paths |

Watched, no collision known: #150 (ActivateMultiple), #149 (SRVB read), #148
(activation parsing), #138 (InstallZADTVSP source deploy), #130 (ENHO read),
#128 (browser-auth client), #107 (WebSocket proxy), #106 (install description).

---

## Fork-only changes

Deliberately not upstreamed. No PR is owed for these.

| Commit | Subject | Why fork-only |
|---|---|---|
| `d752536` | CHANGELOG for v3.0.0 | our own version line |
| `3f7a90c` | goreleaser release target → `frd1201` | must not point upstream releases at this fork |

---

## Known SHA-tracking gaps

Content is correct in all cases; only git's ability to match commits is lost.
Relevant when syncing after upstream merges #120 or #121.

| On `main` | Duplicate of | On branch |
|---|---|---|
| `a47b225` | `2ea6004` | `feat/incl-write-support` |
| `886a9b2` | `59b401b` | `fix/csrf-head-fallback-and-session-type` |

To re-audit what on `main` is not covered by any upstream PR, see section 9.1 of
the strategy report.

---

## Release

- The fork owns the **3.x** version band; upstream is on 2.x. If upstream ever
  tags a 3.x, switch to `v3.4.0-fork.1`.
- Tags are cut from `main` only, and only when the CI job on `main` is green
  **and** an integration run against a real SAP system has passed.

### Local test baseline

`go test ./...` is **not** fully green on a Windows dev box without a C
compiler. Measured 2026-08-03 on `main` at `4deea3b`, Go 1.26.5:

| | |
|---|---|
| packages `ok` | 14 |
| packages without tests | 4 |
| packages failing | 2 — `cmd/vsp`, `pkg/cache` |
| failing tests | 7, all `go-sqlite3 requires cgo to work` |

`CGO_ENABLED` defaults to `0` when no C compiler is on `PATH`, which stubs out
`go-sqlite3`. This is an environment limitation, not a defect. Local gate is
therefore **"no new failures beyond these 7"**; the CI job on `ubuntu-latest`
runs with cgo enabled and is the authoritative gate. Install MinGW/MSYS2 if you
want the sqlite tests locally.
- CHANGELOG keeps two sections per release: *Own changes* and *Adopted from
  upstream* (with PR number and author).

## Module path — review trigger

`go.mod` stays on `github.com/oisee/vibing-steampunk`. Move it to a fork-owned
path in a single commit if any of these happens:

- upstream goes **six months without a code commit** — the clock started
  2026-04-15, so **check on 2026-10-15**; or
- our upstream PRs are rejected; or
- we deliberately decide to hard-fork.

Cost of the move: 104 files (73 of them Go), plus a permanent merge tax on every
upstream sync.
