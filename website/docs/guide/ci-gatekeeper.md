---
sidebar_position: 5
---

# CI Gatekeeper (`sadr doctor`)

`sadr doctor --ci` runs in your pull-/merge-request pipeline and keeps records
consistent. It is fully **deterministic — no AI**: it validates records
**repo-wide** and flags any file documented by more than one active record
(**conflicts**). A reviewer resolves a conflict by **deprecating** the stale
record. Unresolved conflicts block the merge.

This follows the classic ADR convention: records are historical: you don't
rewrite an old decision, you mark it `deprecated` when a newer one supersedes it.
Detection is repo-wide on purpose — a conflict (even a pre-existing one) can be
cleared by any PR, so it never gets stuck unresolved.

## What it does

1. **Validate records** — every `active` record's `file_ref` must exist on disk;
   missing ones are reported as **orphans**.
2. **Detect conflicts** — flag any file documented by **two or more active
   records**. Those are the overlaps to reconcile.
3. **Gate** — if there are conflicts or orphans, post a comment and fail the
   check (non-zero exit), blocking the merge.
4. **Resolve** — a reviewer approves deprecating the stale record(s); doctor sets
   their status to `deprecated`, commits, and the check goes green once every
   file has at most one active record.

## Flags

| Flag | Description |
|---|---|
| `--ci` | Non-interactive mode with structured output |
| `--apply` | Comma-separated record IDs (e.g. `alice/1`) to deprecate |

## Approval

On GitHub the reviewer replies on the PR:

```
/doctor apply alice/1          # deprecate a stale record
/doctor apply alice/1,bob/4    # deprecate several
```

A workflow triggered by the comment re-runs `sadr doctor --ci --apply <ids>`,
which deprecates those records and commits. On GitLab the equivalent is a
**manual job** (set `APPLY_IDS` when running it) — see [GitLab CI](#gitlab-ci).

:::warning Authorization
Only authorized actors can trigger an apply. `doctor.IsAuthorized` accepts both
GitHub `author_association` (OWNER/MEMBER/COLLABORATOR) and GitLab access levels
(OWNER/MAINTAINER/DEVELOPER), read from `GITHUB_ACTOR_ASSOCIATION` or
`DOCTOR_ACTOR_ROLE`. On GitHub the `issue_comment` workflow checks it first; on
GitLab the manual job is gated natively by project role. The binary re-checks as
defense in depth.
:::

## CI integration

The same binary drives both platforms; only the trigger and how the comment is
posted differ. doctor needs no AI key.

### GitHub Actions

`.github/workflows/doctor.yml` has two jobs:

- **detect** — on `pull_request`: runs `sadr doctor --ci`, upserts the PR
  comment, and fails the check on conflicts/orphans.
- **apply** — on `issue_comment`: validates the commenter's `author_association`,
  parses `/doctor apply <ids>` and runs `sadr doctor --ci --apply <ids>`, then
  pushes the deprecation commit.

Make the **detect** check required in branch protection so conflicts block merges.

### GitLab CI

`.gitlab-ci.yml` has two jobs in MR pipelines:

- **doctor:detect** — runs `sadr doctor --ci`, posts the report as an MR note
  (GitLab API), and fails on conflicts/orphans.
- **doctor:apply** — a **manual** job: a reviewer runs it with `APPLY_IDS` set to
  the record ids to deprecate; the commit is pushed back to the MR source branch.

Set **Settings → Merge requests → Pipelines must succeed** so unresolved
conflicts block the merge. Requires a `GITLAB_TOKEN` CI/CD variable (api scope)
for posting notes and pushing.
