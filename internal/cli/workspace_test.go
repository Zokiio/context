package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/workspace"
)

func TestWorkspaceRendererPreservesContractAndStableText(t *testing.T) {
	result := workspace.Result{Complete: true, Workspace: workspace.Identity{ID: "workspace-id", Title: "Workspace title"},
		WorkStatus: "unevaluated", Members: []workspace.Member{
			{Key: "z-first", Title: "First checkout", Directory: "/missing checkout", Records: "/first records",
				AllowSources: []string{"/first docs", "/shared's docs"}, Availability: workspace.Available},
			{Key: "a-second", Records: "/missing records", AllowSources: []string{}, Availability: workspace.Unavailable},
		}, Diagnostics: []workspace.Diagnostic{{Code: "member_records_unavailable", Severity: "warning", MemberKey: "a-second", Path: "/missing records", Message: "records directory is missing"}}}
	report := workspaceNavigationReport(result)
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schemaVersion"] != float64(1) || decoded["kind"] != "workspace-navigation" || decoded["workStatus"] != "unevaluated" || decoded["complete"] != true {
		t.Fatalf("JSON contract = %s", encoded)
	}
	for _, projectField := range []string{"inventoryComplete", "workItems", "sources", "shortlist"} {
		if _, present := decoded[projectField]; present {
			t.Fatalf("workspace JSON contains project field %q", projectField)
		}
	}
	wantFirst := []string{"ctx", "orient", "--bundle", "/first records", "--allow-source", "/first docs", "--allow-source", "/shared's docs"}
	wantSecond := []string{"ctx", "orient", "--bundle", "/missing records"}
	if !reflect.DeepEqual(report.Members[0].SelectionArgs, wantFirst) || !reflect.DeepEqual(report.Members[1].SelectionArgs, wantSecond) {
		t.Fatalf("member selection arguments = %#v", report.Members)
	}
	second := decoded["members"].([]any)[1].(map[string]any)
	if _, present := second["directory"]; present {
		t.Fatal("absent checkout should not appear in JSON")
	}
	var first, again strings.Builder
	if err := renderWorkspace(&first, report); err != nil {
		t.Fatal(err)
	}
	if err := renderWorkspace(&again, report); err != nil {
		t.Fatal(err)
	}
	if first.String() != again.String() {
		t.Fatal("unchanged navigation output is unstable")
	}
	for _, fact := range []string{"Workspace title", "workspace-id", "Work status was not evaluated", "z-first", "a-second", "available", "unavailable", "/missing checkout", posixCommand(wantFirst), posixCommand(wantSecond), "records directory is missing"} {
		if !strings.Contains(first.String(), fact) {
			t.Fatalf("text omitted %q: %s", fact, first.String())
		}
	}
	if strings.Index(first.String(), "z-first") > strings.Index(first.String(), "a-second") {
		t.Fatal("renderer changed authored member order")
	}
}

func TestWorkspaceSelectionCommandRoundTripsThroughPOSIXShell(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell is unavailable")
	}
	root := t.TempDir()
	marker := filepath.Join(root, "must-not-execute")
	records := root + "/record ' \" $HOME $(touch " + marker + ") ; * ? [ ]\nnext"
	args := []string{"ctx", "orient", "--bundle", records, "--allow-source", root + "/docs'quoted", "--allow-source", root + "/backtick\x60touch " + marker + "\x60"}
	command := posixCommand(args)
	output, err := exec.Command(shell, "-c", "set -- "+command+"; printf '%s\\000' \"$@\"").Output()
	if err != nil {
		t.Fatalf("parse selection command: %v", err)
	}
	got := strings.Split(string(output), "\x00")
	got = got[:len(got)-1]
	if !reflect.DeepEqual(got, args) {
		t.Fatalf("shell arguments = %#v; want %#v; command = %s", got, args, command)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("shell expanded a path expression: %v", err)
	}
}

type workspaceFailingWriter struct{ err error }

func (writer workspaceFailingWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestWorkspaceRenderingReportsWriteFailures(t *testing.T) {
	failure := errors.New("output failed")
	declaration := discovery.WorkspaceDeclaration{ID: "empty", Title: "Empty", Members: []discovery.Member{}}
	for _, asJSON := range []bool{false, true} {
		err := runWorkspaceOrientation(context.Background(), workspaceFailingWriter{failure}, declaration, asJSON)
		if !errors.Is(err, failure) || !strings.Contains(err.Error(), "workspace navigation") {
			t.Fatalf("write error = %v", err)
		}
	}
}

func TestWorkspaceEmptyJSONUsesEmptyArrays(t *testing.T) {
	var output bytes.Buffer
	declaration := discovery.WorkspaceDeclaration{ID: "empty", Title: "Empty"}
	if err := runWorkspaceOrientation(context.Background(), &output, declaration, true); err != nil {
		t.Fatal(err)
	}
	for _, fact := range []string{"\"members\":[]", "\"diagnostics\":[]", "\"complete\":true"} {
		if !strings.Contains(output.String(), fact) {
			t.Fatalf("empty workspace JSON lacks %s: %s", fact, output.String())
		}
	}
}
