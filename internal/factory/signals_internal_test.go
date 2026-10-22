package factory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcceptanceCriteriaReadsListItems(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "work.md")
	body := "---\ntype: WorkItem\nid: w\ntitle: W\n---\n## Acceptance criteria\n\n- [ ] First criterion\n  continues here.\n- [x] Second criterion\n1. Third criterion\n\n## Blocked by\n- [Other](other.md)\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	criteria, err := acceptanceCriteria(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(criteria, "|"); got != "First criterion continues here.|Second criterion|Third criterion" {
		t.Fatalf("criteria = %q", got)
	}
}
