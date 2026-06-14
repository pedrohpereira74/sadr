---
sidebar_position: 5
---

# CI Gatekeeper (`sadr doctor`)

`sadr doctor --ci` runs in your pull-request pipeline and keeps records honest:
it validates them, detects when a code change makes a documented decision
out of date (**drift**), and — with human approval — rewrites the affected
sections automatically. Unresolved drift blocks the merge.

It is deterministic where it can be (no AI for validation) and only calls the AI
for the two judgment steps, on a compressed payload.

## What it does

1. **Collect the diff** — `git diff <base>...HEAD`, compressed (diff noise
   stripped; changed files reduced to declaration-line *skeletons*). No AST.
2. **Validate records (no AI)** — for every `active` record, check that its
   `file_ref` exists, and flag **orphans** (missing file) and **collisions**
   (one file documented by several records).
3. **Filter** — intersect the changed files with record `file_ref`s to decide
   what is worth auditing.
4. **Detect drift (AI call #1)** — send the compressed diff, skeletons and the
   affected records; the model returns the drifted sections. No drift → exit 0.
5. **ChatOps** — post the drifts as a single PR comment, each with a stable ID,
   and fail the status check.
6. **Rewrite (AI call #2)** — once a reviewer approves, rewrite only the approved
   sections.
7. **Commit** — write the sections back into the record files and commit. Any
   still-unresolved drift keeps the check red.

## Flags

| Flag | Description |
|---|---|
| `--ci` | Non-interactive mode with structured output |
| `--base` | Base branch/ref of the PR (default `main`) |
| `--apply` | Comma-separated drift IDs, or `all`, approved for rewrite |

## ChatOps approval

The approval model is the industry-standard pair: a **slash command** to pick
what to fix, and a **required status check** to block the merge.

- Doctor posts a comment listing each drift with an ID.
- A reviewer replies on the PR:

  ```
  /doctor apply 3f1a2b,9c4d   # approve specific drifts
  /doctor apply all           # approve everything
  ```

- A workflow triggered by the comment re-runs `sadr doctor --ci --apply <ids>`,
  which rewrites the approved sections and commits them.

:::warning Authorization
Only commenters whose GitHub `author_association` is `OWNER`, `MEMBER` or
`COLLABORATOR` can trigger an apply. The workflow checks this first and the
binary re-checks it (`GITHUB_ACTOR_ASSOCIATION`) as defense in depth — required
because `issue_comment` workflows run with a write token.
:::

## GitHub Actions

The repository ships `.github/workflows/doctor.yml` with two jobs:

- **detect** — on `pull_request`: builds sadr and runs
  `sadr doctor --ci --base origin/<base>`, upserts the PR comment, and fails the
  check on unresolved drift.
- **apply** — on `issue_comment`: validates the commenter, parses the command and
  runs `sadr doctor --ci --apply <ids>`, then pushes the rewrite.

Make the **detect** check required in branch protection so drift actually blocks
merges. `--smart`-style features need `GEMINI_API_KEY` available to the workflow.
