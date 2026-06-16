package doctor

import (
	"reflect"
	"testing"
)

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

func TestValidateMultiFileFileRef(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "a.go,missing.go", Status: "active"},
		{ID: "bob/2", FileRef: "a.go", Status: "active"},
	}
	exists := func(p string) bool { return p == "a.go" }
	res := Validate(records, exists)

	if len(res.Orphans) != 1 || res.Orphans[0].Record != "alice/1" || res.Orphans[0].FileRef != "missing.go" {
		t.Errorf("expected orphan alice/1->missing.go, got %+v", res.Orphans)
	}
	if len(res.Collisions) != 1 || res.Collisions[0].FileRef != "a.go" || len(res.Collisions[0].Records) != 2 {
		t.Errorf("expected collision on a.go with 2 records, got %+v", res.Collisions)
	}
}

func TestValidateRelatedSuppressesCollision(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "api.go", Status: "active"},
		{ID: "bob/2", FileRef: "api.go", Status: "active", Related: []string{"alice/1"}},
	}
	res := Validate(records, func(string) bool { return true })
	if len(res.Collisions) != 0 {
		t.Errorf("expected related records not to collide, got %+v", res.Collisions)
	}
}

func TestValidateRelatedOnlySuppressesAcknowledgedPair(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "api.go", Status: "active"},
		{ID: "bob/2", FileRef: "api.go", Status: "active", Related: []string{"alice/1"}},
		{ID: "carol/3", FileRef: "api.go", Status: "active"},
	}
	res := Validate(records, func(string) bool { return true })
	if len(res.Collisions) != 1 || res.Collisions[0].FileRef != "api.go" {
		t.Fatalf("expected one collision on api.go, got %+v", res.Collisions)
	}
	got := res.Collisions[0].Records
	want := []string{"alice/1", "bob/2", "carol/3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected carol/3's unreconciled pairs to flag all three, got %v", got)
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
