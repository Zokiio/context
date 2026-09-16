package orientation_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestRecursiveAcceptedPrerequisitesExposeSharedAuthoredEdges(t *testing.T) {
	project := acceptedLeafProject(t, "", "None\n", "")
	writeFile(t, project, "middle.md", blockedWork("middle", "completed", "leaf.md"))
	replaceFile(t, project, "dependent.md", "[Leaf](leaf.md)", "[Middle](middle.md)\n[Shared](leaf.md)")
	acceptWork(t, project, "middle.md", "None\n")
	got := orientProject(t, project)
	dependent := findWork(t, got, "dependent.md")
	if !got.Complete || !dependent.Eligible || dependent.Readiness != "ready" || checkStatus(dependent, "dependencies") != "pass" {
		t.Fatalf("dependent=%+v diagnostics=%+v", dependent, got.Diagnostics)
	}
	want := []string{"dependent->middle", "middle->leaf", "dependent->leaf"}
	if paths := graphIDs(dependent.Dependencies); !reflect.DeepEqual(paths, want) {
		t.Fatalf("authored graph: got %v want %v", paths, want)
	}
	for _, edge := range dependent.Dependencies {
		if edge.Status != "pass" || edge.Cycle || edge.To.From != edge.From.Path || edge.To.Link == "" || len(edge.Reasons) == 0 {
			t.Fatalf("edge: %+v", edge)
		}
	}
	writeFile(t, project, "evidence.txt", "Changed leaf observations.\n")
	got = orientProject(t, project)
	dependent = findWork(t, got, "dependent.md")
	if !got.Complete || dependent.Readiness != "blocked" || !hasCheckFinding(dependent, "dependencies", "snapshot_changed") || dependent.Dependencies[0].Status != "fail" {
		t.Fatalf("deeper stale evidence did not block: %+v diagnostics=%+v", dependent, got.Diagnostics)
	}
}

func TestDependencyCyclesRetainParticipantsAndUnrelatedEligibleWork(t *testing.T) {
	project := writeProject(t, map[string]string{
		"project.md":   committedManifest("dependent.md", "a.md", "b.md", "c.md", "ready.md"),
		"a.md":         blockedWork("a", "unstarted", "b.md"),
		"b.md":         blockedWork("b", "cancelled", "c.md"),
		"c.md":         blockedWork("c", "in-progress", "a.md"),
		"dependent.md": blockedWork("dependent", "unstarted", "a.md"),
		"ready.md":     workItem("ready", "unstarted", ""),
	})
	got := orientProject(t, project)
	if !got.Complete || !reflect.DeepEqual(referenceIDs(got.Shortlist), []string{"ready"}) {
		t.Fatalf("cycle should be a complete known condition: shortlist=%+v diagnostics=%+v", got.Shortlist, got.Diagnostics)
	}
	cycleDiagnostics := 0
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Code == "dependency_cycle" {
			cycleDiagnostics++
			if diagnostic.Severity != "warning" || diagnostic.From == "" || diagnostic.Link == "" || diagnostic.Path == "" {
				t.Fatalf("unattributed cycle diagnostic: %+v", diagnostic)
			}
		}
	}
	if cycleDiagnostics != 3 {
		t.Fatalf("one warning per actual cycle edge required, got %d: %+v", cycleDiagnostics, got.Diagnostics)
	}
	for _, path := range []string{"a.md", "b.md", "c.md", "dependent.md"} {
		work := findWork(t, got, path)
		if work.Readiness != "blocked" || !hasCheckFinding(work, "dependencies", "dependency_cycle") {
			t.Fatalf("cycle member/dependent %s: %+v", path, work)
		}
		cycles := []string{}
		for _, edge := range work.Dependencies {
			if edge.Status != "fail" {
				t.Fatalf("cyclic closure edge passed: %+v", edge)
			}
			if edge.Cycle {
				cycles = append(cycles, graphIDs([]orientation.DependencyEdge{edge})[0])
			}
		}
		if len(cycles) != 3 {
			t.Fatalf("cycle edges %s: %v", path, cycles)
		}
	}
	dependent := findWork(t, got, "dependent.md")
	if dependent.Dependencies[0].Cycle || !reflect.DeepEqual(graphIDs(dependent.Dependencies), []string{"dependent->a", "a->b", "b->c", "c->a"}) {
		t.Fatalf("downstream provenance: %+v", dependent.Dependencies)
	}
	for _, edge := range dependent.Dependencies {
		for _, reason := range edge.Reasons {
			if reason.Code == "dependency_cycle" {
				for _, id := range []string{"a", "b", "c"} {
					if !strings.Contains(reason.Message, id+" (") {
						t.Fatalf("missing identity in %q", reason.Message)
					}
				}
			}
		}
	}
	pickup, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "a.md"})
	if err != nil || !pickup.Complete || !pickup.TraversalComplete || len(pickup.Sources) != 3 {
		t.Fatalf("task context cycle changed: %+v error=%v", pickup, err)
	}
	replaceFile(t, project, "b.md", "[Previous](c.md)", "[Previous](c.md)\n- [Unknown](missing.md)")
	got = orientProject(t, project)
	dependent = findWork(t, got, "dependent.md")
	if got.Complete || dependent.Readiness != "blocked" || !hasCheckFinding(dependent, "dependencies", "dependency_cycle") || !hasCheckFinding(dependent, "dependencies", "dependency_unavailable") || !reflect.DeepEqual(referenceIDs(got.Shortlist), []string{"ready"}) {
		t.Fatalf("cycle plus unknown: dependent=%+v shortlist=%+v diagnostics=%+v", dependent, got.Shortlist, got.Diagnostics)
	}
}

func TestAcceptedSelfCycleCannotSatisfyItsOwnDependency(t *testing.T) {
	project := writeProject(t, map[string]string{"project.md": committedManifest("dependent.md"), "self.md": blockedWork("self", "completed", "self.md"), "dependent.md": blockedWork("dependent", "unstarted", "self.md")})
	acceptWork(t, project, "self.md", "None\n")
	got := orientProject(t, project)
	self, dependent := findWork(t, got, "self.md"), findWork(t, got, "dependent.md")
	if !got.Complete || self.Acceptance.Status != "valid" || self.Readiness != "blocked" || dependent.Readiness != "blocked" || len(got.Shortlist) != 0 || len(self.Dependencies) != 1 || !self.Dependencies[0].Cycle || !hasCheckFinding(dependent, "dependencies", "dependency_cycle") {
		t.Fatalf("self cycle self=%+v dependent=%+v diagnostics=%+v", self, dependent, got.Diagnostics)
	}
}

func TestRecursiveDependenciesPreserveKnownFailuresAndEveryUnknown(t *testing.T) {
	project := acceptedLeafProject(t, "", "None\n", "")
	writeFile(t, project, "middle.md", blockedWork("middle", "completed", "leaf.md", "cancelled.md", "missing.md", "wrong.txt"))
	writeFile(t, project, "cancelled.md", workItem("cancelled", "cancelled", ""))
	writeFile(t, project, "wrong.txt", "A plain document.\n")
	replaceFile(t, project, "dependent.md", "[Leaf](leaf.md)", "[Middle](middle.md)")
	acceptWork(t, project, "middle.md", "None\n")
	if err := os.Remove(filepath.Join(project, "evidence.txt")); err != nil {
		t.Fatal(err)
	}
	got := orientProject(t, project)
	dependent := findWork(t, got, "dependent.md")
	if got.Complete || dependent.Readiness != "blocked" || len(dependent.Dependencies) != 5 {
		t.Fatalf("mixed graph status=%s edges=%v diagnostics=%+v", dependent.Readiness, graphIDs(dependent.Dependencies), got.Diagnostics)
	}
	for _, code := range []string{"cancelled_dependency", "dependency_unavailable", "snapshot_unavailable"} {
		if !hasCheckFinding(dependent, "dependencies", code) {
			t.Fatalf("lost %s: %+v", code, dependent.Checks)
		}
	}
	for _, edge := range dependent.Dependencies {
		want := "unknown"
		if edge.To.ID != nil && (*edge.To.ID == "middle" || *edge.To.ID == "cancelled") {
			want = "fail"
		}
		if edge.Status != want || edge.Cycle {
			t.Fatalf("edge status=%s want=%s edge=%+v", edge.Status, want, edge)
		}
	}
	writeFile(t, project, "evidence.txt", "Recorded passing observations.\n")
	writeFile(t, project, "duplicate.md", workItem("leaf", "unstarted", ""))
	got = orientProject(t, project)
	if got.Complete || !hasCode(got, "duplicate_identity") || !hasCheckFinding(findWork(t, got, "dependent.md"), "dependencies", "dependency_unavailable") {
		t.Fatalf("ambiguous deeper identity: %+v", got.Diagnostics)
	}
}

func TestRecursiveGraphSharesExternalSnapshotsAndRespectsCollectionBounds(t *testing.T) {
	shared := "Shared external requirements.\n"
	extra := "## Context\n[Shared](../shared.txt)\n"
	requirements := "- [Shared](../shared.txt) `" + sourceHash(shared) + "`\n"
	project := acceptedLeafProject(t, extra, requirements, "")
	writeFile(t, filepath.Dir(project), "shared.txt", shared)
	writeFile(t, project, "middle.md", blockedWork("middle", "completed", "leaf.md")+extra)
	writeFile(t, project, "a-ready.md", workItem("ready", "unstarted", ""))
	writeFile(t, project, "project.md", committedManifest("dependent.md", "a-ready.md"))
	replaceFile(t, project, "dependent.md", "[Leaf](leaf.md)", "[Middle](middle.md)\n[Shared leaf](leaf.md#details)")
	acceptWork(t, project, "middle.md", requirements)
	request := orientation.Request{ProjectDir: project, AllowedSourceDirs: []string{filepath.Dir(project)}}
	got, err := orientation.Orient(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Complete || len(got.Sources) != 10 || !findWork(t, got, "dependent.md").Eligible {
		t.Fatalf("graph sources=%d diagnostics=%+v", len(got.Sources), got.Diagnostics)
	}
	var bytes int64
	for _, source := range got.Sources {
		info, err := os.Stat(source.Path)
		if err != nil {
			t.Fatal(err)
		}
		bytes += info.Size()
		if filepath.Base(source.Path) == "shared.txt" && len(source.Reasons) != 4 {
			t.Fatalf("shared source provenance: %+v", source)
		}
	}
	request.MaxFiles, request.MaxBytes = len(got.Sources), bytes
	exact, err := orientation.Orient(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !exact.Complete || !reflect.DeepEqual(got.Sources, exact.Sources) || !reflect.DeepEqual(findWork(t, got, "dependent.md").Dependencies, findWork(t, exact, "dependent.md").Dependencies) {
		t.Fatalf("exact fit changed graph: %+v", exact.Diagnostics)
	}
	request.MaxBytes = bytes - 1
	partial, err := orientation.Orient(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	dependent := findWork(t, partial, "dependent.md")
	if partial.Complete || !partial.InventoryComplete || dependent.Readiness != "unknown" || !hasCode(partial, "source_limit_exceeded") || !hasCheckFinding(dependent, "dependencies", "snapshot_unavailable") || !findWork(t, partial, "a-ready.md").Eligible {
		t.Fatalf("partial linked collection: readiness=%s diagnostics=%+v", dependent.Readiness, partial.Diagnostics)
	}
	request.MaxBytes, request.MaxFiles = bytes, 2
	partial, err = orientation.Orient(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	ready := findWork(t, partial, "a-ready.md")
	if partial.InventoryComplete || partial.Complete || len(partial.Shortlist) != 0 || ready.Readiness != "ready" || !hasExclusion(ready, "inventory_incomplete") {
		t.Fatalf("partial inventory ready=%+v diagnostics=%+v", ready, partial.Diagnostics)
	}
	request.MaxFiles, request.MaxBytes = 0, 0
	replaceFile(t, project, "project.md", "## Goals\n", "")
	partial, err = orientation.Orient(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if partial.Complete || !partial.InventoryComplete || !findWork(t, partial, "dependent.md").Eligible || !hasCode(partial, "missing_required_section") {
		t.Fatalf("unrelated manifest gap erased known chain: %+v", partial.Diagnostics)
	}
}

func TestRecursiveDependencyRequiresDeclarationAtEveryReachedWorkItem(t *testing.T) {
	for _, declaration := range []string{"", "## Blocked by\nWe'll decide later.\n"} {
		project := acceptedLeafProject(t, "", "None\n", "")
		writeFile(t, project, "middle.md", blockedWork("middle", "completed", "leaf.md"))
		replaceFile(t, project, "dependent.md", "[Leaf](leaf.md)", "[Middle](middle.md)")
		acceptWork(t, project, "middle.md", "None\n")
		replaceFile(t, project, "leaf.md", "## Blocked by\nNone\n", declaration)
		reacceptLeaf(t, project, "None\n", "")
		got := orientProject(t, project)
		dependent := findWork(t, got, "dependent.md")
		if got.Complete || dependent.Readiness != "unknown" || findWork(t, got, "leaf.md").Acceptance.Status != "valid" || !hasCheckFinding(dependent, "dependencies", "incomplete_relationship_declaration") {
			t.Fatalf("missing deeper declaration: readiness=%s diagnostics=%+v", dependent.Readiness, got.Diagnostics)
		}
	}
}

func blockedWork(id, state string, paths ...string) string {
	links := []string{}
	for _, path := range paths {
		links = append(links, "- [Previous]("+path+")")
	}
	if len(links) == 0 {
		return workItem(id, state, "")
	}
	return strings.Replace(workItem(id, state, ""), "## Blocked by\nNone", "## Blocked by\n"+strings.Join(links, "\n"), 1)
}

func acceptWork(t *testing.T, project, path, requirements string) {
	t.Helper()
	before := orientProject(t, project)
	work := findWork(t, before, path)
	evidence := "Evidence for " + *work.ID + ".\n"
	writeFile(t, project, *work.ID+"-evidence.txt", evidence)
	writeFile(t, project, *work.ID+"-acceptance.md", acceptanceRecord(work, requirements, "- [Evidence]("+*work.ID+"-evidence.txt) `"+sourceHash(evidence)+"`\n", ""))
	data, err := os.ReadFile(filepath.Join(project, path))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, project, path, string(data)+"## Acceptance\n[Current]("+*work.ID+"-acceptance.md)\n")
}

func graphIDs(edges []orientation.DependencyEdge) []string {
	result := []string{}
	for _, edge := range edges {
		from, to := filepath.Base(edge.From.Path), filepath.Base(edge.To.Path)
		if edge.From.ID != nil {
			from = *edge.From.ID
		}
		if edge.To.ID != nil {
			to = *edge.To.ID
		}
		result = append(result, from+"->"+to)
	}
	return result
}
