package doctor

import (
	"strings"
	"testing"
)

func TestSkeletonKeepsDeclarationsDropsBodies(t *testing.T) {
	src := `package api

// NewServer builds a server.
func NewServer(cfg Config) *Server {
	x := 1
	return &Server{x: x}
}

type Config struct {
	Port int
}
`
	got := Skeleton(src)

	mustContain := []string{"package api", "func NewServer(cfg Config) *Server", "type Config struct"}
	for _, w := range mustContain {
		if !strings.Contains(got, w) {
			t.Errorf("skeleton should keep %q\n--- got ---\n%s", w, got)
		}
	}
	mustDrop := []string{"x := 1", "return &Server", "Port int", "NewServer builds a server"}
	for _, w := range mustDrop {
		if strings.Contains(got, w) {
			t.Errorf("skeleton should drop %q\n--- got ---\n%s", w, got)
		}
	}
}

func TestSkeletonMultiLanguage(t *testing.T) {
	cases := map[string]string{
		"def handler(req):":       "def handler(req):\n    return 1",
		"class Foo:":              "class Foo:\n    pass",
		"export function bar() {": "export function bar() {\n  doThing();\n}",
	}
	for want, src := range cases {
		got := Skeleton(src)
		if !strings.Contains(got, want) {
			t.Errorf("Skeleton(%q) should keep %q, got %q", src, want, got)
		}
	}
}

func TestSkeletonEmpty(t *testing.T) {
	if got := Skeleton("   \n\n  "); got != "" {
		t.Errorf("expected empty skeleton, got %q", got)
	}
}
