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
	// GitHub author_association and GitLab access levels.
	for _, a := range []string{"OWNER", "member", "Collaborator", "MAINTAINER", "developer"} {
		if !IsAuthorized(a) {
			t.Errorf("expected %q authorized", a)
		}
	}
	for _, a := range []string{"CONTRIBUTOR", "NONE", "", "FIRST_TIME_CONTRIBUTOR", "REPORTER", "GUEST"} {
		if IsAuthorized(a) {
			t.Errorf("expected %q NOT authorized", a)
		}
	}
}

func TestRenderComment(t *testing.T) {
	out := RenderComment([]AuditTarget{
		{FileRef: "api.go", Records: []string{"alice/1", "bob/4"}},
	})
	for _, want := range []string{CommentMarker, "api.go", "alice/1", "bob/4", "/doctor apply", "deprecated"} {
		if !strings.Contains(out, want) {
			t.Errorf("comment missing %q\n%s", want, out)
		}
	}
}
