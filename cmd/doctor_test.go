package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
