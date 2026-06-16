package cmd

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/doctor"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	rp, wp, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = wp
	fn()
	_ = wp.Close()
	os.Stdout = old
	data, _ := io.ReadAll(rp)
	return string(data)
}

func setupDoctorOrphan(t *testing.T) string {
	t.Helper()
	setupTestUser(t, t.TempDir())
	proj := t.TempDir()
	recordsDir := filepath.Join(proj, ".sadr", "testuser", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatal(err)
	}
	s := storage.NewStorage(recordsDir)
	r, _ := model.NewRecordWithOptions("orphan rec", "full")
	r.Author = "testuser"
	r.Status = "active"
	r.FileRef = "missing.go"
	if _, err := s.SaveRecord(r); err != nil {
		t.Fatal(err)
	}
	old := gitTopLevelFn
	gitTopLevelFn = func() (string, error) { return proj, nil }
	t.Cleanup(func() { gitTopLevelFn = old })
	return recordsDir
}

func TestUnderRoot(t *testing.T) {
	for _, p := range []string{"a.go", "internal/api/server.go", "./x"} {
		if !underRoot(p) {
			t.Errorf("expected %q to be under root", p)
		}
	}
	for _, p := range []string{"/etc/passwd", "../escape.go", "../../x"} {
		if underRoot(p) {
			t.Errorf("expected %q to be rejected", p)
		}
	}
}

func TestDoctorE2EOrphanPostsCommentAndBlocks(t *testing.T) {
	setupDoctorOrphan(t)
	var err error
	out := captureStdout(t, func() {
		err = runDoctor(&doctorOptions{ci: true})(nil, nil)
	})
	if err == nil {
		t.Error("expected non-zero exit on orphan record")
	}
	for _, want := range []string{doctor.CommentMarker, "missing.go", "testuser/1"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected comment to contain %q, got:\n%s", want, out)
		}
	}
}

func TestDoctorE2EApplyResolvesOrphan(t *testing.T) {
	recordsDir := setupDoctorOrphan(t)
	oldCommit := gitCommitFn
	gitCommitFn = func(_ string, _ []string, _ string) error { return nil }
	t.Cleanup(func() { gitCommitFn = oldCommit })

	if err := runDoctor(&doctorOptions{ci: true, apply: "testuser/1"})(nil, nil); err != nil {
		t.Fatalf("deprecating the orphan should resolve, got %v", err)
	}
	entries, _ := storage.NewStorage(recordsDir).ListRecordEntries()
	if len(entries) != 1 || entries[0].Record.Status != "deprecated" {
		t.Errorf("expected the record deprecated, got %+v", entries[0].Record.Status)
	}
}

func TestDoctorCommandRegistered(t *testing.T) {
	cmd := findSubCmd("doctor")
	if cmd == nil {
		t.Fatal("expected 'doctor' command to be registered on root")
	}
	for _, name := range []string{"ci", "apply"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag on doctor command", name)
		}
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
		mk("alice", 3, "exists.go", "active"),
		mk("alice", 4, "old.go", "proposed"),
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

func setupDoctorE2E(t *testing.T, statuses ...string) string {
	t.Helper()
	setupTestUser(t, t.TempDir())

	proj := t.TempDir()
	recordsDir := filepath.Join(proj, ".sadr", "testuser", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "api.go"), []byte("package api\n"), 0644); err != nil {
		t.Fatal(err)
	}
	s := storage.NewStorage(recordsDir)
	for i, st := range statuses {
		r, _ := model.NewRecordWithOptions("record "+string(rune('a'+i)), "full")
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

	return recordsDir
}

func TestDoctorNoProjectVault(t *testing.T) {
	setupTestUser(t, t.TempDir())
	proj := t.TempDir()
	old := gitTopLevelFn
	gitTopLevelFn = func() (string, error) { return proj, nil }
	t.Cleanup(func() { gitTopLevelFn = old })

	if err := runDoctor(&doctorOptions{ci: true})(nil, nil); err != nil {
		t.Errorf("expected pass when repo has no project records, got %v", err)
	}
}

func TestDoctorE2ENoConflict(t *testing.T) {
	setupDoctorE2E(t, "active")
	if err := runDoctor(&doctorOptions{ci: true})(nil, nil); err != nil {
		t.Errorf("expected exit 0 with no conflict, got %v", err)
	}
}

func TestDoctorE2EConflictBlocks(t *testing.T) {
	setupDoctorE2E(t, "active", "active")
	if err := runDoctor(&doctorOptions{ci: true})(nil, nil); err == nil {
		t.Error("expected non-zero exit (gate) when records conflict")
	}
}

func TestDoctorE2ERelatedSuppresses(t *testing.T) {
	setupTestUser(t, t.TempDir())
	proj := t.TempDir()
	recordsDir := filepath.Join(proj, ".sadr", "testuser", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "api.go"), []byte("package api\n"), 0644); err != nil {
		t.Fatal(err)
	}
	s := storage.NewStorage(recordsDir)
	r1, _ := model.NewRecordWithOptions("a", "full")
	r1.Author = "testuser"
	r1.Status = "active"
	r1.FileRef = "api.go"
	if _, err := s.SaveRecord(r1); err != nil {
		t.Fatal(err)
	}
	r2, _ := model.NewRecordWithOptions("b", "full")
	r2.Author = "testuser"
	r2.Status = "active"
	r2.FileRef = "api.go"
	r2.Related = []string{"testuser/1"}
	if _, err := s.SaveRecord(r2); err != nil {
		t.Fatal(err)
	}

	old := gitTopLevelFn
	gitTopLevelFn = func() (string, error) { return proj, nil }
	t.Cleanup(func() { gitTopLevelFn = old })

	if err := runDoctor(&doctorOptions{ci: true})(nil, nil); err != nil {
		t.Errorf("expected related records to pass, got %v", err)
	}
}

func TestDoctorE2EApplyDeprecatesResolves(t *testing.T) {
	recordsDir := setupDoctorE2E(t, "active", "active")

	committed := false
	oldCommit := gitCommitFn
	gitCommitFn = func(_ string, _ []string, _ string) error { committed = true; return nil }
	t.Cleanup(func() { gitCommitFn = oldCommit })

	if err := runDoctor(&doctorOptions{ci: true, apply: "testuser/1"})(nil, nil); err != nil {
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
	if err := runDoctor(&doctorOptions{ci: true, apply: "testuser/1"})(nil, nil); err == nil {
		t.Error("expected unauthorized actor to be rejected")
	}
}
