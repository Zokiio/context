package discovery

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func setupWriteFixture(t *testing.T, existing bool) (SetupRequest, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home, directory, records, docs := filepath.Join(root, "home"), filepath.Join(root, "checkout"), filepath.Join(root, "records"), filepath.Join(root, "docs")
	for _, path := range []string{home, directory, records, docs} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(records, "project.md"), []byte("---\ntype: Project\nid: testing\ntitle: Testing\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, ".context", "config.md")
	if existing {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("---\ntype: ContextConfig\nversion: 1\nproject: {records: ../../records}\n---\nKeep my notes.\n"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	return SetupRequest{Cwd: directory, Home: home, Records: records, AllowSources: []string{docs}, Replace: true}, path
}

func setupWritePlan(t *testing.T, request SetupRequest) *SetupPlan {
	t.Helper()
	plan, err := PrepareSetup(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func setupWriteRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSetupApplyDetectsConcurrentDestinationChanges(t *testing.T) {
	for _, change := range []string{"content", "mode", "permissions-round-trip", "replacement", "creation", "symlink"} {
		t.Run(change, func(t *testing.T) {
			request, path := setupWriteFixture(t, change != "creation")
			plan := setupWritePlan(t, request)
			switch change {
			case "content", "creation":
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("A concurrent author's document.\n"), 0o640); err != nil {
					t.Fatal(err)
				}
			case "mode":
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
			case "permissions-round-trip":
				for _, mode := range []os.FileMode{0o600, 0o640} {
					if err := os.Chmod(path, mode); err != nil {
						t.Fatal(err)
					}
				}
			case "replacement":
				data := setupWriteRead(t, path)
				if err := os.WriteFile(path+".new", data, 0o640); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(path+".new", path); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+".moved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".moved", path); err != nil {
					t.Fatal(err)
				}
			}
			before := setupWriteRead(t, path)
			if err := plan.Apply(context.Background()); err == nil || !strings.Contains(err.Error(), "changed since setup read") {
				t.Fatalf("concurrent change = %v", err)
			}
			if !bytes.Equal(before, setupWriteRead(t, path)) {
				t.Fatal("concurrent edit was overwritten")
			}
		})
	}
}

type setupFaultFile struct {
	file   *os.File
	stage  string
	onSync func()
}

var setupInjectedFailure = errors.New("injected setup failure")

func (f *setupFaultFile) Name() string { return f.file.Name() }
func (f *setupFaultFile) Write(data []byte) (int, error) {
	if f.stage == "write" || f.stage == "short-write" {
		n, err := f.file.Write(data[:7])
		if err != nil || f.stage == "short-write" {
			return n, err
		}
		return n, setupInjectedFailure
	}
	return f.file.Write(data)
}
func (f *setupFaultFile) Sync() error {
	if f.onSync != nil {
		f.onSync()
	}
	if f.stage == "sync" {
		return setupInjectedFailure
	}
	return f.file.Sync()
}
func (f *setupFaultFile) Close() error {
	err := f.file.Close()
	if f.stage == "close" {
		return setupInjectedFailure
	}
	return err
}

func TestSetupWriteFailuresKeepPreviousUsableFile(t *testing.T) {
	for _, stage := range []string{"create", "write", "short-write", "permissions", "sync", "close", "rename"} {
		t.Run(stage, func(t *testing.T) {
			request, path := setupWriteFixture(t, true)
			plan := setupWritePlan(t, request)
			before := setupWriteRead(t, path)
			ops := defaultSetupWriteOps()
			ops.createTemp = func(directory, pattern string) (setupTempFile, error) {
				if stage == "create" {
					return nil, setupInjectedFailure
				}
				file, err := os.CreateTemp(directory, pattern)
				return &setupFaultFile{file: file, stage: stage}, err
			}
			if stage == "rename" {
				ops.rename = func(string, string) error { return setupInjectedFailure }
			}
			if stage == "permissions" {
				ops.preservePermissions = func(string, setupPermissionSnapshot) error { return setupInjectedFailure }
			}
			if err := plan.apply(context.Background(), ops); err == nil {
				t.Fatal("injected failure was ignored")
			}
			if !bytes.Equal(before, setupWriteRead(t, path)) {
				t.Fatal("failed write changed previous file")
			}
			if _, err := ReadConfig(path, SharedConfig); err != nil {
				t.Fatalf("previous file is no longer usable: %v", err)
			}
			files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".ctx-setup-*"))
			if err != nil || len(files) != 0 {
				t.Fatalf("temporary files not cleaned: %v, %v", files, err)
			}
		})
	}
}

func TestSetupRechecksChangesWhileTemporaryFileIsWritten(t *testing.T) {
	for _, otherDeclaration := range []bool{false, true} {
		t.Run(map[bool]string{false: "destination", true: "other-declaration"}[otherDeclaration], func(t *testing.T) {
			request, path := setupWriteFixture(t, true)
			plan := setupWritePlan(t, request)
			before := setupWriteRead(t, path)
			ops := defaultSetupWriteOps()
			ops.createTemp = func(directory, pattern string) (setupTempFile, error) {
				file, err := os.CreateTemp(directory, pattern)
				return &setupFaultFile{file: file, onSync: func() {
					if !otherDeclaration {
						before = append(before, []byte("Concurrent notes.\n")...)
						if err := os.WriteFile(path, before, 0o640); err != nil {
							t.Fatal(err)
						}
						return
					}
					registry := filepath.Join(request.Home, ".context", "config.md")
					if err := os.MkdirAll(filepath.Dir(registry), 0o755); err != nil {
						t.Fatal(err)
					}
					data := "---\ntype: ContextConfig\nversion: 1\nprojects:\n  - key: concurrent\n    directory: " + request.Cwd + "\n    records: " + request.Records + "\n---\n"
					if err := os.WriteFile(registry, []byte(data), 0o644); err != nil {
						t.Fatal(err)
					}
				}}, err
			}
			if err := plan.apply(context.Background(), ops); err == nil {
				t.Fatal("concurrent change while writing was ignored")
			}
			if !bytes.Equal(before, setupWriteRead(t, path)) {
				t.Fatal("concurrent change was overwritten")
			}
		})
	}
}

func TestSetupSerializesCooperatingWriters(t *testing.T) {
	request, path := setupWriteFixture(t, true)
	plans := []*SetupPlan{setupWritePlan(t, request), setupWritePlan(t, request)}
	errs := make([]error, len(plans))
	var wait sync.WaitGroup
	for i, plan := range plans {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errs[i] = plan.Apply(context.Background())
		}()
	}
	wait.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("want exactly one writer and one stale plan, got %v", errs)
	}
	if string(setupWriteRead(t, path)) != plans[0].Document() {
		t.Fatal("concurrent writers did not leave the exact proposal")
	}
}
