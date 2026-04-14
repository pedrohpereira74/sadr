package admin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequireAdminNotConfigured(t *testing.T) {
	dir := t.TempDir()
	err := RequireAdmin(dir)
	if err == nil {
		t.Fatal("expected error when admin not configured")
	}
}

func TestRequireAdminMissingToken(t *testing.T) {
	dir := t.TempDir()
	_, err := Setup(dir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Setenv("SADR_ADMIN_TOKEN", "")
	err = RequireAdmin(dir)
	if err == nil {
		t.Fatal("expected error when SADR_ADMIN_TOKEN is not set")
	}
}

func TestRequireAdminWrongToken(t *testing.T) {
	dir := t.TempDir()
	_, err := Setup(dir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Setenv("SADR_ADMIN_TOKEN", "wrongtoken")
	err = RequireAdmin(dir)
	if err == nil {
		t.Fatal("expected error for wrong token")
	}
}

func TestRequireAdminSuccess(t *testing.T) {
	dir := t.TempDir()
	token, err := Setup(dir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Setenv("SADR_ADMIN_TOKEN", token)
	if err := RequireAdmin(dir); err != nil {
		t.Fatalf("expected success with valid token, got: %v", err)
	}
}

func TestSetupCreatesAdminYaml(t *testing.T) {
	dir := t.TempDir()
	_, err := Setup(dir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "admin.yaml")); os.IsNotExist(err) {
		t.Error("expected admin.yaml to be created")
	}
}

func TestSetupTokenChangesOnRerun(t *testing.T) {
	dir := t.TempDir()
	token1, err := Setup(dir)
	if err != nil {
		t.Fatalf("first setup failed: %v", err)
	}
	token2, err := Setup(dir)
	if err != nil {
		t.Fatalf("second setup failed: %v", err)
	}
	if token1 == token2 {
		t.Error("expected different tokens on each Setup call")
	}
}

func TestIsConfiguredFalseWhenMissing(t *testing.T) {
	dir := t.TempDir()
	if IsConfigured(dir) {
		t.Error("expected IsConfigured to return false on empty dir")
	}
}

func TestIsConfiguredTrueAfterSetup(t *testing.T) {
	dir := t.TempDir()
	_, err := Setup(dir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if !IsConfigured(dir) {
		t.Error("expected IsConfigured to return true after Setup")
	}
}
