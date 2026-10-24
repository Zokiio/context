package factoryworker

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

func TestStandardOfflineFixtureProducesAndChecksRealCandidate(t *testing.T) {
	dir, base := repo(t)
	a := canonical(dir, base)
	raw, _ := json.Marshal(a)
	var out bytes.Buffer
	if e := Fixture(context.Background(), bytes.NewReader(raw), &out, 0, false); e != nil {
		t.Fatal(e)
	}
	candidate, e := git(dir, "rev-parse", "HEAD")
	if e != nil || candidate == base {
		t.Fatalf("candidate %s %v", candidate, e)
	}
	if !strings.Contains(out.String(), "offline fixture") || !strings.Contains(out.String(), candidate) {
		t.Fatalf("fixture output %q", out.String())
	}
	a.Stage = factory.StageVerification
	a.CandidateRevision = candidate
	raw, _ = json.Marshal(a)
	out.Reset()
	if e = FixtureCheck(bytes.NewReader(raw), &out); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), candidate) {
		t.Fatal("check omitted actual candidate identity")
	}
	if e = os.WriteFile(filepath.Join(dir, "waymark-fixture.txt"), []byte("uncommitted change"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = FixtureCheck(bytes.NewReader(raw), &out); e == nil {
		t.Fatal("fixture check passed dirty candidate")
	}
}
func TestStandardFixtureFailureAndModeBoundary(t *testing.T) {
	dir, base := repo(t)
	a := canonical(dir, base)
	raw, _ := json.Marshal(a)
	if e := Fixture(context.Background(), bytes.NewReader(raw), &bytes.Buffer{}, 0, true); e == nil {
		t.Fatal("fixture failure passed")
	}
	head, _ := git(dir, "rev-parse", "HEAD")
	if head != base {
		t.Fatal("failed fixture changed revision")
	}
	a.Fixture = false
	raw, _ = json.Marshal(a)
	if e := Fixture(context.Background(), bytes.NewReader(raw), &bytes.Buffer{}, 0, false); e == nil {
		t.Fatal("nonfixture assignment ran fixture")
	}
}
