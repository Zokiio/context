package orientation_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
)

func TestFingerprintConformance(t *testing.T) {
	paths, err := filepath.Glob("testdata/fingerprints/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("find conformance fixtures: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Source         string
				TicketSHA256   string
				CriteriaSHA256 string
			}
			if err := json.Unmarshal(content, &fixture); err != nil {
				t.Fatal(err)
			}
			result, work := fingerprintWork(t, fixture.Source)
			if work.FingerprintVersion == nil || *work.FingerprintVersion != 1 ||
				work.TicketSHA256 == nil || *work.TicketSHA256 != fixture.TicketSHA256 ||
				work.CriteriaSHA256 == nil || *work.CriteriaSHA256 != fixture.CriteriaSHA256 {
				t.Fatalf("fingerprints: version=%v ticket=%v criteria=%v; expected ticket=%s criteria=%s; diagnostics=%+v",
					work.FingerprintVersion, stringOrNil(work.TicketSHA256), stringOrNil(work.CriteriaSHA256), fixture.TicketSHA256, fixture.CriteriaSHA256, result.Diagnostics)
			}
		})
	}
}

func fingerprintWork(t *testing.T, source string) (orientation.Result, orientation.WorkItem) {
	t.Helper()
	project := writeProject(t, map[string]string{"project.md": emptyManifest(), "work.md": source})
	result, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil || len(result.WorkItems) != 1 {
		t.Fatalf("orientation: work=%+v diagnostics=%+v err=%v", result.WorkItems, result.Diagnostics, err)
	}
	return result, result.WorkItems[0]
}

func stringOrNil(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func TestFingerprintsTrackRequirementsWithoutCommentBookkeeping(t *testing.T) {
	source := "---\ntype: WorkItem\nid: work\ntitle: Work\ntriage: ready-for-agent\nexecution: unstarted\n---\n" +
		"Intro requirement [guide][guide].\n\n## Scope\nScope requirement.\n\n" +
		"## Acceptance criteria\n- [ ] Satisfy [rule][rule].\n\n" +
		"## Blocked by\n[Prerequisite](prior.md)\n\n## Blocked by decisions\nNone\n\n" +
		"## Comments\nRoutine note.\n\n[guide]: guide.md\n[rule]: rule.md\n\n" +
		"## Acceptance\n[Current](acceptance.md)\n"
	baseline, before := fingerprintWork(t, source)
	if baseline.Complete || !hasCode(baseline, "source_missing") {
		t.Fatalf("fingerprints must retain unrelated partial diagnostics: %+v", baseline)
	}
	for _, tc := range []struct {
		name, old, replacement string
		ticket, criteria       bool
	}{
		{"intro", "Intro requirement", "Changed intro requirement", true, false},
		{"scope", "Scope requirement", "Changed scope requirement", true, false},
		{"criterion", "- [ ] Satisfy", "- [x] Satisfy", true, true},
		{"dependency", "(prior.md)", "(other.md)", true, false},
		{"criterion reference", "[rule]: rule.md", "[rule]: changed.md", true, true},
		{"intro reference", "[guide]: guide.md", "[guide]: changed.md", true, false},
		{"ordinary comment", "Routine note.", "A new completion note.", false, false},
		{"current acceptance", "(acceptance.md)", "(new-acceptance.md)", false, false},
		{"title metadata", "title: Work", "title: Renamed work", false, false},
		{"execution metadata", "execution: unstarted", "execution: completed", false, false},
		{"line endings", "\n", "\r\n", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, tc.old, tc.replacement)
			result, after := fingerprintWork(t, changed)
			if before.TicketSHA256 == nil || after.TicketSHA256 == nil ||
				before.CriteriaSHA256 == nil || after.CriteriaSHA256 == nil {
				t.Fatalf("fingerprints unavailable: before=%+v after=%+v", before, after)
			}
			if (*before.TicketSHA256 != *after.TicketSHA256) != tc.ticket ||
				(*before.CriteriaSHA256 != *after.CriteriaSHA256) != tc.criteria {
				t.Fatalf("wrong freshness effect: ticket=%t criteria=%t",
					*before.TicketSHA256 != *after.TicketSHA256, *before.CriteriaSHA256 != *after.CriteriaSHA256)
			}
			if sourceDigest(baseline, "work.md") == sourceDigest(result, "work.md") {
				t.Fatal("changed source must have a different whole-file digest")
			}
		})
	}
}

func TestUnavailableCriteriaAndMalformedInputStayVisible(t *testing.T) {
	result, work := fingerprintWork(t, "---\ntype: WorkItem\nid: work\ntitle: Work\ntriage: ready-for-agent\nexecution: unstarted\n---\nNo criteria section.\n")
	if result.Complete || len(result.Diagnostics) == 0 || work.FingerprintVersion == nil ||
		work.TicketSHA256 == nil || work.CriteriaSHA256 != nil {
		t.Fatalf("missing criteria: work=%+v diagnostics=%+v", work, result.Diagnostics)
	}
	project := writeProject(t, map[string]string{"project.md": emptyManifest(), "work.md": "---\ntype: WorkItem\ntitle: [broken\n---\n## Acceptance criteria\n- Existing bytes.\n"})
	result, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil || result.Complete || len(result.WorkItems) != 0 || !hasCode(result, "invalid_frontmatter") {
		t.Fatalf("malformed input must not acquire invented fingerprints: result=%+v err=%v", result, err)
	}
}

func sourceDigest(result orientation.Result, name string) string {
	for _, source := range result.Sources {
		if filepath.Base(source.Path) == name {
			return source.SHA256
		}
	}
	return ""
}
