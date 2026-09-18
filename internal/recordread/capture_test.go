package recordread_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zokiio/context/internal/recordread"
)

func TestCaptureReusesBytesAndRechecksRoleAuthorization(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	document := filepath.Join(parent, "requirements.md")
	if err := os.WriteFile(document, []byte("before\n"), 0600); err != nil {
		t.Fatal(err)
	}
	capture, err := recordread.NewCapture(project, []string{parent}, recordread.Limits{MaxFiles: 1, MaxBytes: 7})
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()

	first, diagnostic := capture.Read(document, recordread.DocumentSource)
	if diagnostic != nil || first.Text != "before\n" || capture.Admit(first) != nil {
		t.Fatalf("first read: source=%+v diagnostic=%+v", first, diagnostic)
	}
	if err := os.Remove(document); err != nil {
		t.Fatal(err)
	}
	second, diagnostic := capture.Read(document, recordread.DocumentSource)
	if diagnostic != nil || second.Text != first.Text || second.SHA256 != first.SHA256 || capture.Admit(second) != nil {
		t.Fatalf("cached read: source=%+v diagnostic=%+v", second, diagnostic)
	}
	if usage := capture.Usage(); usage.Files != 1 || usage.Bytes != int64(len("before\n")) {
		t.Fatalf("source counted more than once: %+v", usage)
	}
	if _, diagnostic := capture.Read(document, recordread.RecordSource); diagnostic == nil || diagnostic.Code != "source_outside_scope" {
		t.Fatalf("record role reused unauthorized bytes: %+v", diagnostic)
	}
}

func TestCaptureSharesOneWholeSourceBudget(t *testing.T) {
	project := t.TempDir()
	for name, content := range map[string]string{"a.md": "aaa", "b.md": "b"} {
		if err := os.WriteFile(filepath.Join(project, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	capture, err := recordread.NewCapture(project, nil, recordread.Limits{MaxFiles: 1, MaxBytes: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()

	a, diagnostic := capture.Read(filepath.Join(project, "a.md"), recordread.RecordSource)
	if diagnostic != nil || capture.Admit(a) != nil || capture.Admit(a) != nil {
		t.Fatalf("admit shared source: %+v", diagnostic)
	}
	b, diagnostic := capture.Read(filepath.Join(project, "b.md"), recordread.RecordSource)
	if diagnostic != nil {
		t.Fatal(diagnostic)
	}
	if limit := capture.Admit(b); limit == nil || limit.Error() != "file limit of 1" {
		t.Fatalf("second physical source admitted: %+v", limit)
	}
	if usage := capture.Usage(); usage.Files != 1 || usage.Bytes != 3 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}
