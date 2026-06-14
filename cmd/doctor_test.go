package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
		{"1", []string{"1"}},
		{"1,2,3", []string{"1", "2", "3"}},
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

func TestBuildSkeletons(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal/api"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := "package api\nfunc Serve() {\n\tx := 1\n\t_ = x\n}\n"
	if err := os.WriteFile(filepath.Join(root, "internal/api/server.go"), []byte(src), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// One real file and one that does not exist (e.g. deleted in the diff).
	skeletons := buildSkeletons(root, []string{"internal/api/server.go", "gone.go"})

	if _, ok := skeletons["gone.go"]; ok {
		t.Error("unreadable files should be skipped")
	}
	got, ok := skeletons["internal/api/server.go"]
	if !ok {
		t.Fatal("expected skeleton for existing file")
	}
	if !strings.Contains(got, "func Serve()") {
		t.Errorf("skeleton should keep the signature, got %q", got)
	}
	if strings.Contains(got, "x := 1") {
		t.Errorf("skeleton should drop the body, got %q", got)
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

func TestRecordDocsForTargets(t *testing.T) {
	r1, _ := model.NewRecordWithOptions("t", "full")
	r1.FileRef = "api.go"
	r1.Fields = map[string]string{"decision": "uses Old()"}
	r2, _ := model.NewRecordWithOptions("t", "full")
	r2.FileRef = "db.go"
	r2.Fields = map[string]string{"context": "store"}

	entries := []storage.RecordEntry{
		{Record: r1, FileID: 1, Author: "alice"},
		{Record: r2, FileID: 2, Author: "bob"},
	}
	targets := []doctor.AuditTarget{
		{FileRef: "api.go", Records: []string{"alice/1"}},
	}

	docs := recordDocsForTargets(targets, entries)
	if len(docs) != 1 {
		t.Fatalf("expected docs only for targeted records, got %d", len(docs))
	}
	if docs[0].ID != "alice/1" || docs[0].FileRef != "api.go" {
		t.Errorf("unexpected doc %+v", docs[0])
	}
	if docs[0].Sections["decision"] != "uses Old()" {
		t.Errorf("expected record sections to be carried, got %+v", docs[0].Sections)
	}
}

func TestRewriteRequestsForDrifts(t *testing.T) {
	r1, _ := model.NewRecordWithOptions("t", "full")
	r1.FileRef = "api.go"
	r1.Fields = map[string]string{"decision": "uses Old()"}
	entries := []storage.RecordEntry{{Record: r1, FileID: 1, Author: "alice"}}

	drifts := []doctor.Drift{
		{ID: "x", Record: "alice/1", FileRef: "api.go", Section: "decision", Summary: "renamed"},
		{ID: "y", Record: "ghost/9", Section: "decision", Summary: "n/a"}, // unknown record skipped
	}

	reqs := rewriteRequestsForDrifts(drifts, entries)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request (unknown record skipped), got %d", len(reqs))
	}
	if reqs[0].Current != "uses Old()" || reqs[0].Summary != "renamed" {
		t.Errorf("unexpected request %+v", reqs[0])
	}
}

func TestApplyRewritesUpdatesRecordInPlace(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)
	r, _ := model.NewRecordWithOptions("Ordering", "full")
	r.Status = "active"
	r.FieldOrder = []string{"decision"}
	r.Fields = map[string]string{"decision": "uses Old()"}
	if _, err := s.SaveRecord(r); err != nil {
		t.Fatalf("save: %v", err)
	}

	entries, err := s.ListRecordEntries()
	if err != nil || len(entries) != 1 {
		t.Fatalf("list: %v (n=%d)", err, len(entries))
	}
	entries[0].Author = "alice" // entryID -> alice/1

	rewrites := []doctor.Rewrite{{Record: "alice/1", Section: "decision", Content: "now uses New()"}}
	changed, resolved, err := applyRewrites(rewrites, entries)
	if err != nil {
		t.Fatalf("applyRewrites: %v", err)
	}
	if len(changed) != 1 {
		t.Fatalf("expected one changed path, got %v", changed)
	}
	if !resolved[doctor.DriftID("alice/1", "decision")] {
		t.Error("expected the drift id to be marked resolved")
	}

	reloaded, err := storage.LoadRecord(changed[0])
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Fields["decision"] != "now uses New()" {
		t.Errorf("section not rewritten on disk, got %q", reloaded.Fields["decision"])
	}
}

// --- end-to-end (all seams stubbed, real temp .sadr) ---

func setupDoctorE2E(t *testing.T) string {
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
	r, _ := model.NewRecordWithOptions("API contract", "full")
	r.Author = "testuser"
	r.Status = "active"
	r.FileRef = "api.go"
	r.FieldOrder = []string{"decision"}
	r.Fields = map[string]string{"decision": "uses Old()"}
	if _, err := s.SaveRecord(r); err != nil {
		t.Fatal(err)
	}

	old := gitTopLevelFn
	gitTopLevelFn = func() (string, error) { return proj, nil }
	t.Cleanup(func() { gitTopLevelFn = old })

	oldDiff := gitDiffFn
	gitDiffFn = func(string) (string, error) {
		return "diff --git a/api.go b/api.go\n@@ -1 +1 @@\n-func Old()\n+func New()\n", nil
	}
	t.Cleanup(func() { gitDiffFn = oldDiff })

	return recordsDir
}

func stubGenerate(t *testing.T, fn func(prompt string) (string, error)) {
	t.Helper()
	old := generateTextFn
	generateTextFn = func(_ context.Context, prompt, _, _ string, _ time.Duration) (string, error) {
		return fn(prompt)
	}
	t.Cleanup(func() { generateTextFn = old })
}

func TestDoctorE2ENoDrift(t *testing.T) {
	setupDoctorE2E(t)
	stubGenerate(t, func(string) (string, error) { return "[]", nil })

	if err := runDoctor(&doctorOptions{base: "main", ci: true})(nil, nil); err != nil {
		t.Errorf("expected exit 0 when no drift, got %v", err)
	}
}

func TestDoctorE2EDriftBlocksMerge(t *testing.T) {
	setupDoctorE2E(t)
	stubGenerate(t, func(string) (string, error) {
		return `[{"record":"testuser/1","file_ref":"api.go","section":"decision","summary":"renamed Old->New"}]`, nil
	})

	err := runDoctor(&doctorOptions{base: "main", ci: true})(nil, nil)
	if err == nil {
		t.Error("expected non-zero exit (gate) when drift detected")
	}
}

func TestDoctorE2EApplyResolves(t *testing.T) {
	recordsDir := setupDoctorE2E(t)
	stubGenerate(t, func(prompt string) (string, error) {
		if strings.Contains(prompt, "documentation auditor") {
			return `[{"record":"testuser/1","file_ref":"api.go","section":"decision","summary":"renamed Old->New"}]`, nil
		}
		return `[{"record":"testuser/1","section":"decision","content":"now uses New()"}]`, nil
	})

	committed := false
	oldCommit := gitCommitFn
	gitCommitFn = func(_ string, _ []string, _ string) error { committed = true; return nil }
	t.Cleanup(func() { gitCommitFn = oldCommit })

	if err := runDoctor(&doctorOptions{base: "main", ci: true, apply: "all"})(nil, nil); err != nil {
		t.Fatalf("apply should resolve all drift, got %v", err)
	}
	if !committed {
		t.Error("expected the rewritten record to be committed")
	}

	entries, _ := storage.NewStorage(recordsDir).ListRecordEntries()
	if len(entries) != 1 || entries[0].Record.Fields["decision"] != "now uses New()" {
		t.Errorf("expected section rewritten on disk, got %+v", entries[0].Record.Fields)
	}
}
