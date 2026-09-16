package orientation_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
)

func TestLimitsPreserveWholeSourcesAndKnownOmissions(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\n[Large](../large.md)\n[Later](../later.md)\n## Current commitments\nNone\n## Open decisions\nNone\n"
	files := map[string]string{"project.md": manifest, "a.md": "Café\n", "b.md": "b\n"}
	var total int64
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(project, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		total += int64(len(content))
	}
	for name, content := range map[string]string{"large.md": strings.Repeat("x", 200), "later.md": "later"} {
		if err := os.WriteFile(filepath.Join(parent, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		total += int64(len(content))
	}
	request := orientation.Request{ProjectDir: project, AllowedSourceDirs: []string{parent}, MaxFiles: 5, MaxBytes: total}
	got, err := orientation.Orient(context.Background(), request)
	if err != nil || !got.Complete || !got.InventoryComplete || len(got.Sources) != 5 {
		t.Fatalf("exact fit: sources=%d complete=%v inventory=%v diagnostics=%+v err=%v", len(got.Sources), got.Complete, got.InventoryComplete, got.Diagnostics, err)
	}
	for _, tc := range []struct {
		name        string
		files       int
		bytes       int64
		wantSources int
		inventory   bool
		breach      string
		pending     string
	}{
		{"inventory files", 2, total, 2, false, "b.md", "large.md"},
		{"selected files", 3, total, 3, true, "large.md", "later.md"},
		{"source bytes", 5, total - 1, 4, true, "later.md", ""},
		{"manifest bytes", 5, int64(len(manifest) - 1), 0, false, "project.md", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request.MaxFiles, request.MaxBytes = tc.files, tc.bytes
			got, err := orientation.Orient(context.Background(), request)
			if err != nil || got.Complete || got.InventoryComplete != tc.inventory || len(got.Sources) != tc.wantSources {
				t.Fatalf("limit: sources=%d complete=%v inventory=%v diagnostics=%+v err=%v", len(got.Sources), got.Complete, got.InventoryComplete, got.Diagnostics, err)
			}
			breached, pending := false, tc.pending == ""
			for _, d := range got.Diagnostics {
				if d.Code == "source_limit_exceeded" && filepath.Base(d.Path) == tc.breach {
					breached = true
				}
				if d.Code == "source_omitted" && filepath.Base(d.Path) == tc.pending {
					pending = true
				}
			}
			if !breached || !pending {
				t.Fatalf("known omissions: %+v", got.Diagnostics)
			}
		})
	}
}

func TestInvalidEncodingStillConsumesInspectionBudget(t *testing.T) {
	manifest := emptyManifest()
	project := writeProject(t, map[string]string{"project.md": manifest, "a-invalid.md": strings.Repeat(string([]byte{0xff}), 4096), "z-valid.md": "later"})
	for _, tc := range []struct {
		name  string
		files int
		bytes int64
	}{
		{"bytes", 10, int64(len(manifest) + 5)},
		{"files", 2, 10000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, MaxFiles: tc.files, MaxBytes: tc.bytes})
			if err != nil || got.Complete || got.InventoryComplete || len(got.Sources) != 1 || !hasCode(got, "source_limit_exceeded") {
				t.Fatalf("sources=%d diagnostics=%+v err=%v", len(got.Sources), got.Diagnostics, err)
			}
			if tc.name == "bytes" && !hasCode(got, "source_omitted") {
				t.Fatalf("known pending source missing: %+v", got.Diagnostics)
			}
		})
	}
}

func TestDefaultCollectionLimits(t *testing.T) {
	t.Run("files", func(t *testing.T) {
		files := map[string]string{"project.md": emptyManifest()}
		for i := 0; i < 100; i++ {
			files[fmt.Sprintf("doc-%03d.md", i)] = "x"
		}
		got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, files)})
		if err != nil || got.Complete || got.InventoryComplete || len(got.Sources) != 100 || !hasCode(got, "source_limit_exceeded") {
			t.Fatalf("default file limit: sources=%d diagnostics=%+v err=%v", len(got.Sources), got.Diagnostics, err)
		}
	})
	t.Run("bytes", func(t *testing.T) {
		manifest := emptyManifest()
		manifest += strings.Repeat(" ", int(orientation.DefaultMaxBytes)-len(manifest))
		project := writeProject(t, map[string]string{"project.md": manifest})
		got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
		if err != nil || !got.Complete {
			t.Fatalf("exact default byte limit: %+v %v", got.Diagnostics, err)
		}
		if err := os.WriteFile(filepath.Join(project, "project.md"), []byte(manifest+" "), 0600); err != nil {
			t.Fatal(err)
		}
		got, err = orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
		if err != nil || got.Complete || len(got.Sources) != 0 || !hasCode(got, "source_limit_exceeded") {
			t.Fatalf("default byte limit: sources=%d diagnostics=%+v err=%v", len(got.Sources), got.Diagnostics, err)
		}
	})
}
