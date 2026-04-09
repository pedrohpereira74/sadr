# Implementation Plan

Second review pass after the `refactor/code-quality` branch.
Each item includes file, line(s), root cause, and the concrete fix.

---

## Priority 1 — Bugs (wrong behaviour today)

### 1.1 Search result matching by title+type is wrong
**File:** `cmd/search.go:156-162`

After calling `search.Search(allRecords, ...)`, the code does a reverse
lookup to recover `fileID` and `author`:

```go
for _, e := range searchEntries {
    if e.Record.Title == r.Title && e.Record.Type == r.Type { // ← fragile
```

Two records can share the same title and type.  When that happens the loop
returns the fileID of whichever one appears first in the list — which may be
the wrong record.  A user could `edit` or `delete` a record they never
intended to touch.

**Fix:** Change `search.Search` to accept and return `[]storage.RecordEntry`
instead of `[]model.Record`.  The `search` package already imports `model`;
adding `storage` creates no cycle.

```go
// internal/search/search.go
func Search(entries []storage.RecordEntry, query string, deep bool) []storage.RecordEntry
```

`cmd/search.go` then passes `searchEntries` directly and iterates the
returned slice — no reverse lookup needed.

---

### 1.2 `writeGlobalConfig` parses YAML as plain text
**File:** `cmd/init.go:145-163`

`writeGlobalConfig` reads the file line-by-line looking for `username:` and
does string substitution.  This breaks if:
- `username:` appears inside a comment or a nested key
- The value is already quoted in a format the code doesn't expect
- The file uses Windows line endings

**Fix:** Use `config.LoadGlobalFromFile` → mutate the struct → marshal back
with `gopkg.in/yaml.v3`.  Same pattern already used everywhere else.

```go
func writeGlobalConfig(path string, cfg config.GlobalConfig) error {
    data, err := yaml.Marshal(cfg)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0600)
}
```

---

### 1.3 `~/.sadr` directory created world-readable
**File:** `cmd/config.go:57`, `cmd/init.go:70`

The global config directory (which contains the Gemini API key) is created
with `0755`.  Any user on the same machine can list its contents.

```go
_ = os.MkdirAll(globalDir, 0755) // ← should be 0700
```

**Fix:** Change both occurrences to `0700`.

---

### 1.4 Git diff snippet has no size limit
**File:** `cmd/new.go:168-181`

`readSnippetFromSource` enforces a 10 MB cap when reading from `--file`
(`model.MaxSnippetFileSize`) but reads the entire `git diff` output
unconditionally.  A large monorepo diff can fill memory.

**Fix:** Add the same cap after reading the diff output:

```go
output, err := cmd.Output()
if len(output) > model.MaxSnippetFileSize {
    output = output[:model.MaxSnippetFileSize]
}
```

---

### 1.5 `initGlobal` silently ignores load error after writing template
**File:** `cmd/init.go:90`

```go
cfg, _ = config.LoadGlobalFromFile(globalConfigPath) // error discarded
```

If the newly-written template file is somehow corrupt or unreadable, `cfg`
stays as a zero-value `GlobalConfig`.  The username is set and written back,
but language, AI config, etc., are all zeroed out — silently overwriting
whatever the user had.

**Fix:**

```go
cfg, err = config.LoadGlobalFromFile(globalConfigPath)
if err != nil {
    ui.Error(os.Stderr, fmt.Sprintf("failed to load global config after writing: %v", err))
    return
}
```

---

## Priority 2 — Security

### 2.1 API key exposed in process list
**File:** `cmd/config.go:49-108` (`--set-api-key` flag)

Passing a secret as a CLI argument writes it to `/proc/<pid>/cmdline` and
makes it visible via `ps aux` to every user on the machine.

**Fix (minimal):** Read the key from stdin when `--set-api-key` is provided
without a value, or print a clear warning.  Better long-term: provide a
dedicated `sadr config set-api-key` interactive sub-command that reads from
stdin.

```go
// immediate workaround — print warning when value is provided inline
if opts.setAPIKey != "" && !isFromStdin() {
    ui.Warning(os.Stderr, "tip: API keys passed as arguments are visible in process listings. consider piping via stdin.")
}
```

---

### 2.2 `--file` flag allows reading files outside project root
**File:** `cmd/new.go:150-165`

The `--file` path is read without checking whether it escapes the project.
A user running `sadr new --file /etc/shadow` would read that file into a
record snippet.

**Fix:** After resolving the absolute path, verify it sits within `cwd`:

```go
abs, err := filepath.Abs(opts.file)
if err != nil || !strings.HasPrefix(abs, cwd+string(filepath.Separator)) {
    ui.Error(os.Stderr, "file must be inside the current project directory.")
    return ""
}
```

---

## Priority 3 — Code Quality / Smells

### 3.1 Duplicated tag parsing in three places
The pattern "split by comma, trim spaces" appears independently in:
- `internal/storage/storage.go` (`buildFrontmatter`, `formatBody`, `LoadRecord`)
- `cmd/export.go:119-126`
- `cmd/ask.go:filterRecordEntries` (uses `search.HasAnyTag`)

**Fix:** Add `ParseTags(s string) []string` to the `model` package (next to
the existing field constants).  Replace all inline split-and-trim with calls
to it.

```go
// internal/model/record.go
func ParseTags(s string) []string {
    var tags []string
    for t := range strings.SplitSeq(s, ",") {
        t = strings.TrimSpace(t)
        if t != "" {
            tags = append(tags, t)
        }
    }
    return tags
}
```

---

### 3.2 `search.go` strips `RecordEntry` metadata, forcing O(n²) reverse lookup
**Root cause of bug 1.1, also a design smell.**  `search.Search` accepts
`[]model.Record` and loses `fileID`, `author`, and `path`.  The caller must
then re-scan all entries to recover them.

**Fix:** covered by 1.1 (change `Search` signature to use `RecordEntry`).

---

### 3.3 `addToGitignore` silently swallows all errors
**File:** `cmd/init.go:294-306`

Every write error in `addToGitignore` is discarded.  If the disk is full or
the file is read-only, `sadr init` still reports "done!" with no indication
that the gitignore was not updated.

**Fix:** Return `error` from `addToGitignore` and surface it to the caller:

```go
func addToGitignore(dir string) error { ... }

// in initFresh:
if err := addToGitignore(cwd); err != nil {
    ui.Warning(os.Stderr, fmt.Sprintf("could not update .gitignore: %v", err))
}
```

---

### 3.4 Config field count has no upper limit
**File:** `internal/config/config.go:validate()`

A config file with 10 000 fields would load and validate without complaint,
creating a wizard with 10 000 steps.

**Fix:** Add a cap in `validate()`:

```go
const maxFields = 50
if len(cfg.Fields) > maxFields {
    return fmt.Errorf("config has %d fields; maximum allowed is %d", len(cfg.Fields), maxFields)
}
```

---

### 3.5 Wizard textarea height can go negative on tiny terminals
**File:** `internal/wizard/wizard.go` — `handleKeyMsg`, `tea.WindowSizeMsg` branch

```go
if msg.Height > 8 {
    m.textarea.SetHeight(msg.Height - 8)
}
```

If the terminal is exactly 8 lines tall, the textarea height is never updated
and stays at the default.  Heights of 1–8 result in no update at all.

**Fix:**

```go
m.textarea.SetHeight(max(2, msg.Height-8))
```

---

### 3.6 `resolveConfigPath` duplicated between `cmd/new.go` and `cmd/config.go`
**Files:** `cmd/new.go:68-109`, `cmd/config.go:155-181`

Both functions enumerate `.sadr/configs/*.yaml`, prompt the user if multiple
exist, and return a path.  They differ only in the prompt text.

**Fix:** Extract to a shared `pickConfigFile(configsDir, prompt string) (string, error)` in `cmd/helpers.go`.

---

## Priority 4 — Missing Tests

### 4.1 `filterRecordEntries` has no tests
**File:** `cmd/ask.go` (recently extracted)

Test cases needed:
- tag filter: only matching tags pass
- field filter: `key=value` exact match
- status filter: `proposed`, `deprecated`, `superseded` are excluded
- range cutoff: records older than the cutoff are excluded
- limit: only the last N records survive
- combined filters

---

### 4.2 `writeAnswerFile` has no tests
**File:** `cmd/ask.go` (recently extracted)

Test cases needed:
- happy path: file is written with correct name pattern and content
- answers dir is created if it doesn't exist
- error is returned when the directory cannot be created

---

### 4.3 `search.Search` has no unit tests
**File:** `internal/search/search.go`

Test cases needed:
- basic query match by title
- basic query match by tags
- `deep=true` matches snippet body
- `deep=true` matches custom fields
- case-insensitive matching
- no results returns empty slice (not nil — to avoid nil vs empty issues)

---

### 4.4 `cmd/export.go` has no tests
All export paths (by ID, `--all`, `--tags`) are untested.

---

### 4.5 `cmd/delete.go` — `--force` path untested
The confirmation prompt path is tested implicitly (mock returns "yes"), but
the `--force` flag that bypasses the prompt entirely has no dedicated test.

---

### 4.6 `cmd/edit.go` has no tests
Opening the editor on a valid record, invalid ID, and missing editor are all
untested.

---

## Priority 5 — Low / Cosmetic

| # | File | Issue | Fix |
|---|------|-------|-----|
| 5.1 | `internal/enricher/sourcecode/sourcecode.go:23-60` | No deduplication of `file_ref` entries — same file read twice if duplicated | Build a `seen` set before the loop |
| 5.2 | `cmd/config.go:156` | `entries, _ := os.ReadDir(...)` error discarded | Log or surface the error |
| 5.3 | `cmd/helpers.go:171-183` | `listAllRecordEntries` silently skips dirs that fail to read | Return a multi-error or warn the user |
| 5.4 | `internal/config/config.go` | Reserved field names (`snippet`, `file_ref`) can be redefined in config | Add check to `validate()` |
| 5.5 | `internal/wizard/wizard.go` | `handleEnterKey` editor branch falls back to `"vim"` hardcoded | Use `findEditor()` from `cmd` — or move `findEditor` to a shared package |

---

## Execution Order

```
P1 bugs first (1.1 → 1.5)   — correctness, some involve user data
P2 security (2.1 → 2.2)     — low effort, high value
P3 quality (3.1 → 3.6)      — housekeeping, reduces future bugs
P4 tests (4.1 → 4.6)        — run in parallel with P3
P5 low priority              — opportunistic, with related changes
```

Branch naming suggestion: `fix/search-entry-lookup` for 1.1+3.2 (same root
cause), `fix/init-yaml` for 1.2, remaining items in `refactor/quality-pass-2`.
