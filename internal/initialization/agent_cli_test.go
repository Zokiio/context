package initialization_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/resumption"
	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
)

func TestAgentCLIGuideRunsReadersForTargetFromUnrelatedDirectory(t *testing.T) {
	request := portableRequest(t)
	result := initializePortable(t, request)
	if result.Incomplete() {
		t.Fatalf("preparation is incomplete: %+v", result)
	}
	guidePath := filepath.Join(request.Directory, request.Docs, "cli.md")
	guide := portableFile(t, guidePath)
	if !strings.Contains(result.Pointers, filepath.ToSlash(filepath.Join(request.Docs, "cli.md"))) {
		t.Fatal("instruction pointers do not select the installed CLI guide")
	}
	build := exec.Command("go", "build", "-o", request.Executable, "../../cmd/ctx")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build supplied binary: %v\n%s", err, output)
	}
	const requirement = "# Selected project's requirement\n"
	const task = "---\ntype: WorkItem\nid: cli-guide-task\ntitle: CLI guide task\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n\n- Apply the selected requirement.\n\n## Blocked by\n\nNone\n\n## Blocked by decisions\n\nNone\n\n## Context\n\n- [Requirement](../../sources/requirement.md)\n"
	for path, text := range map[string]string{
		filepath.Join(request.Directory, "sources", "requirement.md"): requirement,
		filepath.Join(request.Directory, request.Records, "task.md"):  task,
	} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	caller := filepath.Join(filepath.Dir(request.Directory), "unrelated caller")
	if err := os.Mkdir(caller, 0o755); err != nil {
		t.Fatal(err)
	}
	commands := map[string]string{}
	parsed := parser.New().Parse(guide)
	_ = ast.Walk(parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		block, ok := node.(*ast.CodeBlock)
		if !ok || !entering || block.CodeBlockKind != ast.CodeBlockKindFenced {
			return ast.WalkContinue, nil
		}
		if language, ok := block.Language(guide); !ok || language != "sh" {
			return ast.WalkContinue, nil
		}
		command := string(block.Value.Bytes(guide))
		for _, operation := range []string{"orient", "context", "resume"} {
			if strings.Contains(command, " "+operation+" ") {
				if _, exists := commands[operation]; exists {
					t.Fatalf("duplicate documented %s example", operation)
				}
				commands[operation] = strings.ReplaceAll(command, "<ticket-path-relative-to-records>", "'task.md'")
			}
		}
		return ast.WalkContinue, nil
	})
	for _, operation := range []string{"orient", "context", "resume"} {
		t.Run(operation, func(t *testing.T) {
			command, exists := commands[operation]
			if !exists {
				t.Fatalf("no executable %s example in installed CLI guide", operation)
			}
			run := exec.Command("sh", "-c", command)
			run.Dir = caller
			output, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("documented %s failed: %v\n%s\n%s", operation, err, command, output)
			}
			var contextResult taskcontext.Result
			switch operation {
			case "orient":
				var report orientation.Result
				if err := json.Unmarshal(output, &report); err != nil {
					t.Fatal(err)
				}
				if !report.Complete || !report.InventoryComplete || report.Project.Title != request.Title || len(report.WorkItems) != 1 || report.WorkItems[0].ID == nil || *report.WorkItems[0].ID != "cli-guide-task" {
					t.Fatalf("orientation selected another project or omitted its task: %s", output)
				}
				return
			case "context":
				if err := json.Unmarshal(output, &contextResult); err != nil {
					t.Fatal(err)
				}
			case "resume":
				var report resumption.Result
				if err := json.Unmarshal(output, &report); err != nil {
					t.Fatal(err)
				}
				if report.Scope.WorkingDirectory != request.Directory {
					t.Fatalf("resume selected another checkout: %s", output)
				}
				contextResult = report.Context
			}
			if !contextResult.Complete || len(contextResult.Sources) != 2 || contextResult.Sources[0].Text != task || contextResult.Sources[1].Text != requirement {
				t.Fatalf("%s did not supply the selected task and requirement text: %s", operation, output)
			}
		})
	}
}
