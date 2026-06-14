package doctor

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseApplyCommand(t *testing.T) {
	cases := []struct {
		body string
		ids  []string
		all  bool
		ok   bool
	}{
		{"looks good\n/doctor apply 1a2b, 3c4d\nthanks", []string{"1a2b", "3c4d"}, false, true},
		{"/doctor apply all", nil, true, true},
		{"/doctor apply", nil, false, false},
		{"no command here", nil, false, false},
		{"/doctor apply abc", []string{"abc"}, false, true},
	}
	for _, c := range cases {
		ids, all, ok := ParseApplyCommand(c.body)
		if ok != c.ok || all != c.all || !reflect.DeepEqual(ids, c.ids) {
			t.Errorf("ParseApplyCommand(%q) = (%v,%v,%v), want (%v,%v,%v)", c.body, ids, all, ok, c.ids, c.all, c.ok)
		}
	}
}

func TestIsAuthorized(t *testing.T) {
	for _, a := range []string{"OWNER", "member", "Collaborator"} {
		if !IsAuthorized(a) {
			t.Errorf("expected %q authorized", a)
		}
	}
	for _, a := range []string{"CONTRIBUTOR", "NONE", "", "FIRST_TIME_CONTRIBUTOR"} {
		if IsAuthorized(a) {
			t.Errorf("expected %q NOT authorized", a)
		}
	}
}

func TestSelectDrifts(t *testing.T) {
	drifts := []Drift{
		{ID: "aaa", Record: "alice/1"},
		{ID: "bbb", Record: "alice/2"},
		{ID: "ccc", Record: "bob/3"},
	}
	if got := SelectDrifts(drifts, nil, true); len(got) != 3 {
		t.Errorf("all should select everything, got %d", len(got))
	}
	got := SelectDrifts(drifts, []string{"aaa", "ccc"}, false)
	if len(got) != 2 || got[0].ID != "aaa" || got[1].ID != "ccc" {
		t.Errorf("unexpected selection %+v", got)
	}
	if got := SelectDrifts(drifts, []string{"zzz"}, false); len(got) != 0 {
		t.Errorf("unknown id should select nothing, got %+v", got)
	}
}

func TestRenderComment(t *testing.T) {
	out := RenderComment([]Drift{{ID: "aaa", Record: "alice/1", Section: "Decision", Summary: "sig changed"}})
	for _, want := range []string{CommentMarker, "aaa", "alice/1", "Decision", "sig changed", "/doctor apply"} {
		if !strings.Contains(out, want) {
			t.Errorf("comment missing %q\n%s", want, out)
		}
	}
}
