//go:build !darwin && !linux

package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnsupportedSetupPermissionsFailBeforeReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.md")
	if err := os.WriteFile(path, []byte("existing document\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := preserveSetupPermissions(path+".temp", captureSetupPermissions(path, before)); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported replacement = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "existing document\n" {
		t.Fatalf("original file changed: %q, %v", data, err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !sameSetupPermissionSnapshots(captureSetupPermissions(path, before), captureSetupPermissions(path, after)) {
		t.Fatal("read-only access invalidated the permission snapshot")
	}
}
