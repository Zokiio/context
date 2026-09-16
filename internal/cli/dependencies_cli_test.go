package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
)

func TestOrientationTextExplainsDirectAndTransitiveEdges(t *testing.T) {
	var stdout, stderr bytes.Buffer
	result := orientation.Result{Complete: true, WorkItems: []orientation.WorkItem{{
		ID: str("work"), Title: str("Work"), Source: "/bundle/work.md",
		Checks: []orientation.Check{{Name: "dependencies", Status: "fail"}},
		Dependencies: []orientation.DependencyEdge{
			{From: orientation.Reference{Path: "/bundle/work.md", ID: str("work"), Title: str("Work")},
				To: orientation.Reference{Path: "/bundle/middle.md", ID: str("middle"), Title: str("Middle"), From: "/bundle/work.md", Link: "./middle.md#scope"}, Status: "fail",
				Reasons: []orientation.Finding{{Code: "blocked_chain", Message: "Middle has a cyclic prerequisite."}}},
			{From: orientation.Reference{Path: "/bundle/middle.md", ID: str("middle"), Title: str("Middle")},
				To: orientation.Reference{Path: "/bundle/middle.md", ID: str("middle"), Title: str("Middle"), From: "/bundle/middle.md", Link: "middle.md"}, Status: "fail", Cycle: true,
				Reasons: []orientation.Finding{{Code: "dependency_cycle", Message: "Middle depends on itself.", Path: "/bundle/middle.md", From: "/bundle/middle.md", Link: "middle.md"}}},
		},
	}}}
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--project", "."}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		return result, nil
	}})
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stderr=%q stdout=%q", status, stderr.String(), stdout.String())
	}
	for _, fact := range []string{"Dependency edges:", "Work [work] /bundle/work.md", "Middle [middle] /bundle/middle.md", "status: fail; cycle: false", "status: fail; cycle: true",
		"link \"./middle.md#scope\"", "from /bundle/middle.md; link \"middle.md\"", "blocked_chain: Middle has a cyclic prerequisite.", "dependency_cycle: Middle depends on itself.", "dependencies: fail"} {
		if !strings.Contains(stdout.String(), fact) {
			t.Errorf("missing dependency fact %q", fact)
		}
	}
	if t.Failed() {
		t.Log(stdout.String())
	}
}
