package doctor

import "testing"

func TestValidateCleanWhenAllFileRefsExist(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "a.go", Status: "active"},
		{ID: "alice/2", FileRef: "b.go", Status: "active"},
	}
	exists := func(string) bool { return true }
	res := Validate(records, exists)
	if !res.OK() {
		t.Errorf("expected clean result, got %+v", res)
	}
}

func TestValidateDetectsOrphans(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "exists.go", Status: "active"},
		{ID: "alice/2", FileRef: "gone.go", Status: "active"},
	}
	exists := func(p string) bool { return p == "exists.go" }
	res := Validate(records, exists)
	if len(res.Orphans) != 1 || res.Orphans[0].Record != "alice/2" || res.Orphans[0].FileRef != "gone.go" {
		t.Errorf("expected one orphan alice/2->gone.go, got %+v", res.Orphans)
	}
}

func TestValidateDetectsCollisions(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "shared.go", Status: "active"},
		{ID: "bob/3", FileRef: "shared.go", Status: "active"},
	}
	res := Validate(records, func(string) bool { return true })
	if len(res.Collisions) != 1 {
		t.Fatalf("expected one collision, got %+v", res.Collisions)
	}
	c := res.Collisions[0]
	if c.FileRef != "shared.go" || len(c.Records) != 2 {
		t.Errorf("unexpected collision %+v", c)
	}
}

func TestRemainingCollisions(t *testing.T) {
	collisions := []Collision{
		{FileRef: "api.go", Records: []string{"alice/1", "bob/2"}},
		{FileRef: "db.go", Records: []string{"alice/3", "alice/4", "alice/5"}},
	}
	remaining := RemainingCollisions(collisions, map[string]bool{"bob/2": true, "alice/3": true})
	if len(remaining) != 1 || remaining[0].FileRef != "db.go" || len(remaining[0].Records) != 2 {
		t.Errorf("expected db.go still conflicting with 2 records, got %+v", remaining)
	}
}

func TestValidateIgnoresNonActiveAndNA(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "gone.go", Status: "proposed"},
		{ID: "alice/2", FileRef: "N/A", Status: "active"},
		{ID: "alice/3", FileRef: "", Status: "active"},
	}
	res := Validate(records, func(string) bool { return false })
	if !res.OK() {
		t.Errorf("expected clean result (all ignored), got %+v", res)
	}
}
