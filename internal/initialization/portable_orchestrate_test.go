package initialization_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/initialization"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/resumption"
	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/goccy/go-yaml"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
)

func portableRequest(t *testing.T) initialization.Request {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request := initialization.Request{
		Directory: filepath.Join(root, "project's \"target\""),
		Home:      filepath.Join(root, "home"), Executable: filepath.Join(root, "supplied's \"ctx\""),
		Version: "test binary\n", Title: "Portable project", Tracker: "Existing local roadmap",
		Skills: "agent's \"skills\"", Docs: "guide's \"docs\"", Records: ".context/records", Local: ".context/local",
	}
	for _, path := range []string{request.Directory, request.Home, filepath.Join(request.Directory, "sources")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	request.AllowSources = []string{"sources"}
	return request
}

func initializePortable(t *testing.T, request initialization.Request) initialization.Result {
	t.Helper()
	result, err := initialization.Initialize(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func portableFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPortableOrchestrateInstallsClosedExplicitSkill(t *testing.T) {
	request := portableRequest(t)
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		if err := os.WriteFile(filepath.Join(request.Directory, name), []byte("Existing project instructions\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if result := initializePortable(t, request); result.Incomplete() {
		t.Fatalf("new project preparation is incomplete: %+v", result)
	}
	skills := filepath.Join(request.Directory, request.Skills)
	docs := filepath.Join(request.Directory, request.Docs)
	for _, name := range []string{"orchestrate", "retro"} {
		skill, err := recordread.ParseDocument(portableFile(t, filepath.Join(skills, name, "SKILL.md")))
		if err != nil || skill.Metadata["name"] != name {
			t.Fatalf("invalid installed %s skill: metadata=%v err=%v", name, skill.Metadata, err)
		}
		if description, ok := skill.Metadata["description"].(string); !ok || strings.TrimSpace(description) == "" {
			t.Fatalf("installed %s skill has no description: %v", name, skill.Metadata)
		}
		if skill.Metadata["disable-model-invocation"] != true {
			t.Fatalf("installed %s skill must disable model invocation for Claude Code: %v", name, skill.Metadata)
		}
		var metadata struct {
			Policy struct {
				AllowImplicitInvocation *bool `yaml:"allow_implicit_invocation"`
			} `yaml:"policy"`
		}
		if err := yaml.Unmarshal(portableFile(t, filepath.Join(skills, name, "agents", "openai.yaml")), &metadata); err != nil {
			t.Fatal(err)
		}
		if metadata.Policy.AllowImplicitInvocation == nil || *metadata.Policy.AllowImplicitInvocation {
			t.Fatalf("%s must explicitly disable implicit invocation", name)
		}
	}
	installed := map[string][]byte{}
	for _, root := range []string{skills, docs} {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data := portableFile(t, path)
			installed[path] = data
			if filepath.Ext(path) != ".md" {
				return nil
			}
			parsed := parser.New().Parse(data)
			return ast.Walk(parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
				link, ok := node.(*ast.Link)
				if !ok || !entering {
					return ast.WalkContinue, nil
				}
				destination, err := url.Parse(string(link.Destination.Value(data)))
				if err != nil {
					t.Fatal(err)
				}
				if destination.IsAbs() || destination.Host != "" || destination.Path == "" {
					return ast.WalkContinue, nil
				}
				target := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(destination.Path)))
				inInstallation := false
				for _, allowed := range []string{skills, docs} {
					rel, err := filepath.Rel(allowed, target)
					inInstallation = inInstallation || err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
				}
				if !inInstallation {
					t.Errorf("installed guidance %s requires a source outside the installation: %s", path, target)
				} else if info, err := os.Stat(target); err != nil || !info.Mode().IsRegular() {
					t.Errorf("installed guidance %s has unavailable local reference %s: %v", path, target, err)
				}
				return ast.WalkContinue, nil
			})
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		if data := portableFile(t, filepath.Join(request.Directory, name)); string(data) != "Existing project instructions\n" {
			t.Fatalf("init changed %s", name)
		}
	}
	for _, unselected := range []string{".agents", ".claude", ".codex"} {
		if _, err := os.Lstat(filepath.Join(request.Directory, unselected)); !os.IsNotExist(err) {
			t.Errorf("init created an unselected agent directory %s: %v", unselected, err)
		}
	}
	if result := initializePortable(t, request); result.Incomplete() {
		t.Fatalf("identical rerun is incomplete: %+v", result)
	}
	for path, before := range installed {
		if !bytes.Equal(before, portableFile(t, path)) {
			t.Errorf("identical rerun changed installed file %s", path)
		}
	}
}

func TestPortableOrchestratePreservesExistingFilesAndSymlinks(t *testing.T) {
	for _, name := range []string{"SKILL.md", "agents/openai.yaml"} {
		for _, symlink := range []bool{false, true} {
			t.Run(name+"/symlink="+map[bool]string{false: "false", true: "true"}[symlink], func(t *testing.T) {
				request := portableRequest(t)
				path := filepath.Join(request.Directory, request.Skills, "orchestrate", name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				target := path
				if symlink {
					target = filepath.Join(request.Directory, "existing authority")
				}
				if err := os.WriteFile(target, []byte("Existing user content\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if symlink {
					if err := os.Symlink(target, path); err != nil {
						t.Fatal(err)
					}
				}
				result := initializePortable(t, request)
				var reported []string
				for _, file := range result.Files {
					if file.Path == path {
						reported = append(reported, file.Status)
						if file.Reason == "" {
							t.Fatal("conflict report has no reason")
						}
					}
				}
				if !result.Incomplete() || !reflect.DeepEqual(reported, []string{"skipped"}) {
					t.Fatalf("existing orchestration file was not reported as preserved for review: %+v", result)
				}
				if data := portableFile(t, target); string(data) != "Existing user content\n" {
					t.Fatal("init overwrote existing content")
				}
				if symlink {
					if got, err := os.Readlink(path); err != nil || got != target {
						t.Fatalf("init replaced the symlink: target=%q err=%v", got, err)
					}
				}
			})
		}
	}
}

func portableDocumentedReaderCommand(t *testing.T, data []byte, operation string) string {
	t.Helper()
	var commands []string
	parsed := parser.New().Parse(data)
	_ = ast.Walk(parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		block, ok := node.(*ast.CodeBlock)
		if !ok || !entering || block.CodeBlockKind != ast.CodeBlockKindFenced {
			return ast.WalkContinue, nil
		}
		if language, ok := block.Language(data); !ok || language != "sh" {
			return ast.WalkContinue, nil
		}
		command := string(block.Value.Bytes(data))
		if strings.Contains(command, " "+operation+" ") && strings.Contains(command, "--ticket") {
			commands = append(commands, command)
		}
		return ast.WalkContinue, nil
	})
	if len(commands) != 1 {
		t.Fatalf("want one runnable documented %s command, got %d", operation, len(commands))
	}
	return commands[0]
}

func TestPortableOrchestrateReaderCommandsUseSuppliedBinaryAndProject(t *testing.T) {
	request := portableRequest(t)
	initializePortable(t, request)
	// Requiring the entrypoint first makes the original packaging gap visible.
	portableFile(t, filepath.Join(request.Directory, request.Skills, "orchestrate", "SKILL.md"))
	build := exec.Command("go", "build", "-o", request.Executable, "../../cmd/ctx")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build supplied binary: %v\n%s", err, output)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(request.Directory, "sources", "domain.md"), "# Target-project requirement\n")
	write(filepath.Join(request.Directory, request.Records, "task.md"), "---\ntype: WorkItem\nid: selected-target-task\ntitle: Selected target task\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n\n- Apply the target-project requirement.\n\n## Blocked by\n\nNone\n\n## Blocked by decisions\n\nNone\n\n## Context\n\n- [Domain](../../sources/domain.md)\n")
	caller := filepath.Join(filepath.Dir(request.Directory), "unrelated caller")
	if err := os.MkdirAll(caller, 0o755); err != nil {
		t.Fatal(err)
	}
	contextOutput := filepath.Join(caller, "context's \"output\".json")
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	for _, operation := range []string{"context", "resume"} {
		name := "task-context"
		if operation == "resume" {
			name = "recovery-notes"
		}
		data := portableFile(t, filepath.Join(request.Directory, request.Skills, name, "SKILL.md"))
		command := portableDocumentedReaderCommand(t, data, operation)
		command = strings.ReplaceAll(command, "<ticket-path>", quote("task.md"))
		command = strings.ReplaceAll(command, "<ticket-path-relative-to-records>", quote("task.md"))
		command = strings.ReplaceAll(command, "<temporary-context-used.json>", quote(contextOutput))
		run := exec.Command("sh", "-c", command)
		run.Dir = caller
		run.Env = append(os.Environ(), "HOME="+request.Home)
		output, err := run.CombinedOutput()
		if err != nil {
			t.Fatalf("installed %s command from unrelated project: %v\n%s\n%s", operation, err, command, output)
		}
		if operation == "context" {
			if redirected, err := os.ReadFile(contextOutput); err == nil {
				output = redirected
			}
		}
		var contextResult taskcontext.Result
		if operation == "context" {
			if err := json.Unmarshal(output, &contextResult); err != nil {
				t.Fatalf("documented context command did not return JSON: %v\n%s", err, output)
			}
		} else {
			var resume resumption.Result
			if err := json.Unmarshal(output, &resume); err != nil {
				t.Fatalf("documented resume command did not return JSON: %v\n%s", err, output)
			}
			contextResult = resume.Context
		}
		if !contextResult.Complete || len(contextResult.Sources) != 2 {
			t.Fatalf("%s did not select the target task and its requirement: %+v", operation, contextResult)
		}
		if !strings.Contains(contextResult.Sources[1].Text, "Target-project requirement") {
			t.Fatalf("%s returned sources from another project: %+v", operation, contextResult.Sources)
		}
	}
}
