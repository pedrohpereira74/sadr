---
sidebar_position: 4
---

# Team Workflows

## Multi-user projects

sadr is designed for shared repositories. Each contributor's records are stored under their own username subdirectory inside `.sadr/`, so multiple people can write records in the same project without conflicts.

### Directory layout

```
.sadr/
├── configs/
│   └── default-config.yaml     ← shared schema, committed by the team
├── alice/
│   ├── records/
│   │   ├── sadr-record-0001-retry-logic.md
│   │   └── sadr-record-0002-auth-middleware.md
│   └── exports/
└── bob/
    ├── records/
    │   └── sadr-record-0001-database-indexing.md
    └── exports/
```

Each user's records directory has its own sequential ID counter starting from 1. IDs are unique per user, not globally. To reference a specific user's record unambiguously, use `user/id` format:

```bash
sadr search --id alice/2
sadr edit --id bob/1
```

### Setup for a new team member

```bash
# 1. Clone the repository (which already has .sadr/configs/)
git clone ...

# 2. Set up personal global config (one-time, not committed)
sadr init --global
# → set username, editor, language, API key

# 3. Start capturing records
sadr new --diff --smart
```

### Schema ownership

The `configs/` directory is team-owned and git-tracked. Treat changes to it like any other code change: PR, review, merge.

```bash
# After changing the schema:
git add .sadr/configs/default-config.yaml
git commit -m "feat(sadr): add consequences field to schema"
```

### Viewing other contributors' records

```bash
# Search across all users in the project
sadr search "retry"

# Filter to a specific author
sadr search "retry" --user alice

# View a specific record by user/id
sadr search --id bob/3
```

---

## Personal vault

The personal vault (`~/.sadr/`) is completely separate from any project. It holds records that are not meant to be shared — personal notes, cross-project snippets, private observations.

```bash
# Save to personal vault
sadr new --clipboard --smart --global

# Search personal vault
sadr search "pattern" --global

# Export from personal vault
sadr export --all --global
```

The personal vault has its own `configs/` with a schema you control privately. It never touches the project `.sadr/` directory.

---

## Git workflow recommendations

### What to commit

```bash
# Commit the shared schema — always
git add .sadr/configs/

# Commit your records — yes, that's the point
git add .sadr/alice/records/

# Do NOT commit exports or answers
# (add to .gitignore if preferred)
```

### `.gitignore` suggestion

```gitignore
# sadr exports and AI answers — generated, not source
.sadr/*/exports/
.sadr/*/answers/
```

### Committing records with the code

The most effective workflow is to commit a record in the same commit as the code it documents:

```bash
# You just wrote a feature
git add src/payment/retry.go
sadr new --diff --smart    # captures the diff as snippet
git add .sadr/alice/records/sadr-record-0042-payment-retry.md
git commit -m "feat: add retry logic to payment client

Captured in sadr record #42."
```

This creates a durable link between the decision and the change in the git history.

---

## Using multiple schemas

For large projects, you may want different schemas for different contexts:

```
.sadr/configs/
├── default-config.yaml   ← general purpose
├── db.yaml               ← database decisions
├── api.yaml              ← API contract decisions
└── security.yaml         ← security review records
```

Select at capture time:

```bash
sadr new --diff --smart --config db
sadr new --clipboard --config security
```

If multiple configs exist and `--config` is not passed, sadr prompts you to choose. If only one exists, it's used automatically.
