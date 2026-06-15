package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/doctor"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func TestDoctorCommandRegistered(t *testing.T) {
	cmd := findSubCmd("doctor")
	if cmd == nil {
		t.Fatal("expected 'doctor' command to be registered on root")
	}
	for _, name := range []string{"ci", "base", "apply"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag on doctor command", name)
		}
	}
	if got := cmd.Flags().Lookup("base").DefValue; got != "main" {
		t.Errorf("expected --base default 'main', got %q", got)
	}
}

func TestParseApplyIDs(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"alice/1", []string{"alice/1"}},
		{"alice/1,bob/2", []string{"alice/1", "bob/2"}},
		{" a , , b ", []string{"a", "b"}},
		{",,", nil},
	}
	for _, c := range cases {
		if got := parseApplyIDs(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseApplyIDs(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCollectDoctorDiff(t *testing.T) {
	oldDiff := gitDiffFn
	defer func() { gitDiffFn = oldDiff }()

	var gotBase string
	gitDiffFn = func(base string) (string, error) {
		gotBase = base
		return `diff --git a/internal/api/server.go b/internal/api/server.go
index 111..222 100644
--- a/internal/api/server.go
+++ b/internal/api/server.go
@@ -1 +1 @@
-func Old()
+func New()
diff --git a/internal/api/client.go b/internal/api/client.go
index 333..444 100644`, nil
	}

	diff, files, err := collectDoctorDiff("main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBase != "main" {
		t.Errorf("expected base 'main' passed to git, got %q", gotBase)
	}
	if diff == "" {
		t.Error("expected non-empty raw diff")
	}
	want := []string{"internal/api/server.go", "internal/api/client.go"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("changed files = %v, want %v", files, want)
	}
}

func TestCollectDoctorDiffError(t *testing.T) {
	oldDiff := gitDiffFn
	defer func() { gitDiffFn = oldDiff }()
	gitDiffFn = func(base string) (string, error) { return "", errors.New("boom") }

	if _, _, err := collectDoctorDiff("main"); err == nil {
		t.Error("expected error to propagate from git diff failure")
	}
}

func TestRecordRefsFromEntriesAndValidate(t *testing.T) {
	mk := func(author string, id int, fileRef, status string) storage.RecordEntry {
		r, _ := model.NewRecordWithOptions("t", "full")
		r.FileRef = fileRef
		r.Status = status
		return storage.RecordEntry{Record: r, FileID: id, Author: author}
	}
	entries := []storage.RecordEntry{
		mk("alice", 1, "exists.go", "active"),
		mk("bob", 2, "gone.go", "active"),
		mk("alice", 3, "exists.go", "active"), // collides with alice/1
		mk("alice", 4, "old.go", "proposed"), // ignored (not active)
	}

	refs := recordRefsFromEntries(entries)
	if refs[0].ID != "alice/1" || refs[1].ID != "bob/2" {
		t.Fatalf("unexpected ids: %+v", refs)
	}

	res := doctor.Validate(refs, func(p string) bool { return p == "exists.go" })
	if len(res.Orphans) != 1 || res.Orphans[0].Record != "bob/2" {
		t.Errorf("expected orphan bob/2, got %+v", res.Orphans)
	}
	if len(res.Collisions) != 1 || res.Collisions[0].FileRef != "exists.go" {
		t.Errorf("expected collision on exists.go, got %+v", res.Collisions)
	}
}

func TestDeprecateRecords(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)
	for _, title := range []string{"first", "second"} {
		r, _ := model.NewRecordWithOptions(title, "full")
		r.Author = "alice"
		r.Status = "active"
		r.FileRef = "api.go"
		if _, err := s.SaveRecord(r); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	entries, _ := s.ListRecordEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 records, got %d", len(entries))
	}

	changed, done, err := deprecateRecords([]string{"alice/1", "ghost/9"}, entries)
	if err != nil {
		t.Fatalf("deprecateRecords: %v", err)
	}
	if len(changed) != 1 || !done["alice/1"] || done["ghost/9"] {
		t.Fatalf("expected only alice/1 deprecated, changed=%v done=%v", changed, done)
	}
	reloaded, _ := storage.LoadRecord(changed[0])
	if reloaded.Status != "deprecated" {
		t.Errorf("expected status deprecated on disk, got %q", reloaded.Status)
	}
}

// --- end-to-end (conflict detection + deprecate, all seams stubbed) ---

// setupDoctorE2E builds a temp project with two active records both documenting
// api.go (a conflict), a stubbed repo root and a diff that touches api.go.
// Returns the records dir.
func setupDoctorE2E(t *testing.T, statuses ...string) string {
	t.Helper()
	setupTestUser(t, t.TempDir()) // sets HOME to a sandbox

	proj := t.TempDir()
	recordsDir := filepath.Join(proj, ".sadr", "testuser", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "api.go"), []byte("package api\nfunc New() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	s := storage.NewStorage(recordsDir)
	for i, st := range statuses {
		r, _ := model.NewRecordWithOptions("record "+st+string(rune('a'+i)), "full")
		r.Author = "testuser"
		r.Status = st
		r.FileRef = "api.go"
		if _, err := s.SaveRecord(r); err != nil {
			t.Fatal(err)
		}
	}

	oldTop := gitTopLevelFn
	gitTopLevelFn = func() (string, error) { return proj, nil }
	t.Cleanup(func() { gitTopLevelFn = oldTop })

	oldDiff := gitDiffFn
	gitDiffFn = func(string) (string, error) {
		return "diff --git a/api.go b/api.go\n@@ -1 +1 @@\n-func Old()\n+func New()\n", nil
	}
	t.Cleanup(func() { gitDiffFn = oldDiff })

	return recordsDir
}

func TestDoctorE2ENoConflict(t *testing.T) {
	setupDoctorE2E(t, "active") // single active record on api.go
	if err := runDoctor(&doctorOptions{base: "main", ci: true})(nil, nil); err != nil {
		t.Errorf("expected exit 0 with no conflict, got %v", err)
	}
}

func TestDoctorE2EConflictBlocks(t *testing.T) {
	setupDoctorE2E(t, "active", "active") // two active records on api.go
	err := runDoctor(&doctorOptions{base: "main", ci: true})(nil, nil)
	if err == nil {
		t.Error("expected non-zero exit (gate) when records conflict")
	}
}

func TestDoctorE2EApplyDeprecatesResolves(t *testing.T) {
	recordsDir := setupDoctorE2E(t, "active", "active")

	committed := false
	oldCommit := gitCommitFn
	gitCommitFn = func(_ string, _ []string, _ string) error { committed = true; return nil }
	t.Cleanup(func() { gitCommitFn = oldCommit })

	// Deprecate one of the two conflicting records -> conflict resolved.
	if err := runDoctor(&doctorOptions{base: "main", ci: true, apply: "testuser/1"})(nil, nil); err != nil {
		t.Fatalf("apply should resolve the conflict, got %v", err)
	}
	if !committed {
		t.Error("expected the deprecation to be committed")
	}

	entries, _ := storage.NewStorage(recordsDir).ListRecordEntries()
	var deprecated int
	for _, e := range entries {
		if e.Record.Status == "deprecated" {
			deprecated++
		}
	}
	if deprecated != 1 {
		t.Errorf("expected exactly one deprecated record, got %d", deprecated)
	}
}

func TestDoctorE2EApplyUnauthorized(t *testing.T) {
	setupDoctorE2E(t, "active", "active")
	t.Setenv("GITHUB_ACTOR_ASSOCIATION", "NONE")
	if err := runDoctor(&doctorOptions{base: "main", ci: true, apply: "testuser/1"})(nil, nil); err == nil {
		t.Error("expected unauthorized actor to be rejected")
	}
}
