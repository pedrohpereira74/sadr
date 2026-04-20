package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedrohpereira74/sadr/internal/discover"
)

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		query  string
		maxLen int
		want   string
	}{
		{
			name:   "title within limit",
			title:  "short title",
			query:  "title",
			maxLen: 40,
			want:   "short title",
		},
		{
			name:   "title exact limit",
			title:  "exactly forty characters long title here",
			query:  "query",
			maxLen: 40,
			want:   "exactly forty characters long title here",
		},
		{
			name:   "query not found, truncate from start",
			title:  "This is a very long title without the search term anywhere",
			query:  "zzz",
			maxLen: 20,
			want:   "This is a very lo...",
		},
		{
			name:   "empty query, truncate from start",
			title:  "This is a very long title that exceeds the max length",
			query:  "",
			maxLen: 20,
			want:   "This is a very...",
		},
		{
			name:   "query at the beginning",
			title:  "Architecture Decision Record about microservices",
			query:  "Architecture",
			maxLen: 20,
			want:   "Architecture D...",
		},
		{
			name:   "query at the end",
			title:  "Decision record about the microservices pattern",
			query:  "pattern",
			maxLen: 20,
			want:   "...rvices pattern",
		},
		{
			name:   "query in the middle",
			title:  "Long prefix text queryterm long suffix text here",
			query:  "queryterm",
			maxLen: 24,
			want:   "...ext queryterm long...",
		},
		{
			name:   "case insensitive match",
			title:  "Architecture Decision Record about microservices",
			query:  "ARCHITECTURE",
			maxLen: 20,
			want:   "Architecture D...",
		},
		{
			name:   "query longer than window",
			title:  "This is a very long title with a verylongquerythatexceedswindow inside",
			query:  "verylongquerythatexceedswindow",
			maxLen: 20,
			want:   "...querythatexcee...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateTitle(tt.title, tt.query, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateTitle(%q, %q, %d)\n got  %q\n want %q", tt.title, tt.query, tt.maxLen, got, tt.want)
			}
		})
	}
}

// --- parseID ---

func TestParseIDEmpty(t *testing.T) {
	n, err := parseID("")
	if err != nil || n != 0 {
		t.Errorf("expected (0, nil), got (%d, %v)", n, err)
	}
}

func TestParseIDValid(t *testing.T) {
	n, err := parseID("42")
	if err != nil || n != 42 {
		t.Errorf("expected (42, nil), got (%d, %v)", n, err)
	}
}

func TestParseIDNegative(t *testing.T) {
	if _, err := parseID("-1"); err == nil {
		t.Error("expected error for negative id")
	}
}

func TestParseIDNonNumeric(t *testing.T) {
	if _, err := parseID("abc"); err == nil {
		t.Error("expected error for non-numeric id")
	}
}

// --- parseUserID ---

func TestParseUserIDEmpty(t *testing.T) {
	u, n, err := parseUserID("")
	if err != nil || u != "" || n != 0 {
		t.Errorf("expected (\"\", 0, nil), got (%q, %d, %v)", u, n, err)
	}
}

func TestParseUserIDNumberOnly(t *testing.T) {
	u, n, err := parseUserID("7")
	if err != nil || u != "" || n != 7 {
		t.Errorf("expected (\"\", 7, nil), got (%q, %d, %v)", u, n, err)
	}
}

func TestParseUserIDNameSlashNumber(t *testing.T) {
	u, n, err := parseUserID("pedro/3")
	if err != nil || u != "pedro" || n != 3 {
		t.Errorf("expected (\"pedro\", 3, nil), got (%q, %d, %v)", u, n, err)
	}
}

func TestParseUserIDNegative(t *testing.T) {
	if _, _, err := parseUserID("-5"); err == nil {
		t.Error("expected error for negative id")
	}
}

func TestParseUserIDInvalidFormat(t *testing.T) {
	if _, _, err := parseUserID("pedro/abc"); err == nil {
		t.Error("expected error for non-numeric part")
	}
}

// --- configDisplayName / configFilename ---

func TestConfigDisplayNameDefault(t *testing.T) {
	if got := configDisplayName("default-config.yaml"); got != "default" {
		t.Errorf("got %q, want \"default\"", got)
	}
}

func TestConfigDisplayNameCustom(t *testing.T) {
	if got := configDisplayName("my-project.yaml"); got != "my-project" {
		t.Errorf("got %q, want \"my-project\"", got)
	}
}

func TestConfigFilenameDefault(t *testing.T) {
	if got := configFilename("default"); got != "default-config.yaml" {
		t.Errorf("got %q, want \"default-config.yaml\"", got)
	}
}

func TestConfigFilenameCustom(t *testing.T) {
	if got := configFilename("my-project"); got != "my-project.yaml" {
		t.Errorf("got %q, want \"my-project.yaml\"", got)
	}
}

func TestConfigDisplayNameConfigFilenameRoundtrip(t *testing.T) {
	names := []string{"default", "my-project", "team-backend"}
	for _, name := range names {
		if got := configDisplayName(configFilename(name)); got != name {
			t.Errorf("roundtrip failed for %q: got %q", name, got)
		}
	}
}

// --- resolveRecordDirs ---

func TestResolveRecordDirsGlobal(t *testing.T) {
	paths := discover.SadrPaths{Records: "/home/user/.sadr/records"}
	dirs := resolveRecordDirs(true, paths)
	if len(dirs) != 1 || dirs[0] != "/home/user/.sadr/records" {
		t.Errorf("expected single records dir, got %v", dirs)
	}
}

func TestResolveRecordDirsLocal(t *testing.T) {
	root := t.TempDir()
	user1 := filepath.Join(root, "alice", "records")
	user2 := filepath.Join(root, "bob", "records")
	_ = os.MkdirAll(user1, 0755)
	_ = os.MkdirAll(user2, 0755)

	paths := discover.SadrPaths{Root: root}
	dirs := resolveRecordDirs(false, paths)
	if len(dirs) != 2 {
		t.Errorf("expected 2 record dirs, got %d: %v", len(dirs), dirs)
	}
}

// --- parseRangeCutoff ---

func TestParseRangeCutoffEmpty(t *testing.T) {
	if got := parseRangeCutoff(""); !got.IsZero() {
		t.Errorf("expected zero time for empty input, got %v", got)
	}
}

func TestParseRangeCutoffInvalid(t *testing.T) {
	for _, bad := range []string{"abc", "0d", "-1w", "1x"} {
		if got := parseRangeCutoff(bad); !got.IsZero() {
			t.Errorf("parseRangeCutoff(%q): expected zero time, got %v", bad, got)
		}
	}
}

func TestParseRangeCutoffDays(t *testing.T) {
	before := time.Now().AddDate(0, 0, -7)
	got := parseRangeCutoff("7d")
	after := time.Now().AddDate(0, 0, -7)
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("parseRangeCutoff(\"7d\") = %v, expected ~%v", got, before)
	}
}

func TestParseRangeCutoffWeeks(t *testing.T) {
	before := time.Now().AddDate(0, 0, -14)
	got := parseRangeCutoff("2w")
	after := time.Now().AddDate(0, 0, -14)
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("parseRangeCutoff(\"2w\") = %v, expected ~%v", got, before)
	}
}

func TestParseRangeCutoffMonths(t *testing.T) {
	before := time.Now().AddDate(0, -3, 0)
	got := parseRangeCutoff("3m")
	after := time.Now().AddDate(0, -3, 0)
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("parseRangeCutoff(\"3m\") = %v, expected ~%v", got, before)
	}
}

func TestParseRangeCutoffYears(t *testing.T) {
	before := time.Now().AddDate(-1, 0, 0)
	got := parseRangeCutoff("1y")
	after := time.Now().AddDate(-1, 0, 0)
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("parseRangeCutoff(\"1y\") = %v, expected ~%v", got, before)
	}
}
