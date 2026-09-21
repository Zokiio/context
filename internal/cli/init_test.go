package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/resumption"
	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
)

func initEnvironment(t *testing.T) (cli.Environment, []string) {
	t.Helper()
	root := scopeTempDir(t)
	cwd, home := filepath.Join(root, "target project"), filepath.Join(root, "home")
	for _, path := range []string{cwd, home, filepath.Join(cwd, "docs")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := scopeEnvironment(cwd, home)
	env.Executable = func() (string, error) { return filepath.Join(root, "supplied binary's ctx"), nil }
	env.Input = scopeForbiddenInput{t}
	env.IsTerminal = func() bool { t.Fatal("init inspected the terminal"); return false }
	return env, []string{"ctx", "init", "--title", "A \"project\": title", "--tracker", "GitHub Issues", "--skills-dir", ".agents/skills", "--docs-dir", "docs/ctx", "--allow-source", "docs"}
}

func runInit(t *testing.T, env cli.Environment, args []string, want int) string {
	t.Helper()
	var out, stderr bytes.Buffer
	code := cli.RunWithEnvironment(context.Background(), args, &out, &stderr, cli.Operations{}, env)
	if code != want || want != 2 && stderr.Len() != 0 {
		t.Fatalf("exit=%d want=%d stdout=%s stderr=%s", code, want, out.String(), stderr.String())
	}
	return out.String()
}

func TestInitPreparesProjectAndPreservesIdenticalRerun(t *testing.T) {
	env, args := initEnvironment(t)
	cwd, _ := env.WorkingDirectory()
	home, _ := env.HomeDirectory()
	writeScopeFile(t, filepath.Join(cwd, "AGENTS.md"), "Existing agent instructions\n")
	writeScopeFile(t, filepath.Join(cwd, "CLAUDE.md"), "Existing Claude instructions\n")
	writeScopeFile(t, filepath.Join(cwd, ".gitignore"), "existing-rule")
	output := runInit(t, env, args, 0)
	for _, directory := range []string{cwd, home} {
		if !strings.Contains(output, "created: "+filepath.Join(directory, ".context/config.md.lock")) {
			t.Fatalf("missing created lock report: %s", output)
		}
	}
	for path, want := range map[string]string{"AGENTS.md": "Existing agent instructions\n", "CLAUDE.md": "Existing Claude instructions\n"} {
		if got, _ := os.ReadFile(filepath.Join(cwd, path)); string(got) != want {
			t.Fatalf("init changed %s", path)
		}
	}
	ignore, _ := os.ReadFile(filepath.Join(cwd, ".gitignore"))
	if !bytes.HasPrefix(ignore, []byte("existing-rule\n")) {
		t.Fatalf("ignore prefix changed: %q", ignore)
	}
	files := initTree(t, cwd)
	for path, data := range files {
		if strings.HasSuffix(path, ".md") {
			checkInitLinks(t, filepath.Join(cwd, path), data)
		}
	}
	for _, path := range []string{".agents/skills/task-context/SKILL.md", ".agents/skills/recovery-notes/SKILL.md"} {
		doc, err := recordread.ParseDocument(files[path])
		if err != nil || doc.Metadata["name"] == nil || doc.Metadata["description"] == nil {
			t.Fatalf("invalid skill %s: %v", path, err)
		}
	}
	manifest, err := recordread.ParseDocument(files[".context/records/project.md"])
	if err != nil || manifest.Metadata["title"] != "A \"project\": title" || manifest.Metadata["id"] == nil {
		t.Fatalf("manifest=%+v err=%v", manifest, err)
	}
	var version, stderr bytes.Buffer
	cli.RunWithEnvironment(context.Background(), []string{"ctx", "version"}, &version, &stderr, cli.Operations{}, env)
	if !bytes.Equal(version.Bytes(), files[".context/trial/ctx-version.txt"]) {
		t.Fatal("init did not retain the running binary's version output")
	}
	output = runInit(t, env, args, 0)
	for _, directory := range []string{cwd, home} {
		if !strings.Contains(output, "preserved: "+filepath.Join(directory, ".context/config.md.lock")) {
			t.Fatalf("missing preserved lock report: %s", output)
		}
	}
	if !reflect.DeepEqual(files, initTree(t, cwd)) {
		t.Fatal("identical init changed file contents or Project identity")
	}
	var out bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "orient", "--json"}, &out, &stderr, cli.Operations{Orient: orientation.Orient}, env)
	var project orientation.Result
	if err := json.Unmarshal(out.Bytes(), &project); err != nil || status != 0 || !project.Complete || len(project.WorkItems) != 0 || len(project.Decisions) != 0 {
		t.Fatalf("empty project: status=%d report=%s stderr=%s err=%v", status, out.String(), stderr.String(), err)
	}
	// Task selection remains manual. The initialized binding supports it later.
	writeScopeFile(t, filepath.Join(cwd, "docs", "domain.md"), "# Domain requirements\n")
	writeScopeFile(t, filepath.Join(cwd, ".context", "records", "task.md"), "---\ntype: WorkItem\nid: chosen-task\ntitle: Chosen task\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n\n- Meet the domain requirements.\n\n## Blocked by\n\nNone\n\n## Blocked by decisions\n\nNone\n\n## Context\n\n- [Domain](../../docs/domain.md)\n")
	for _, command := range []string{"context", "resume"} {
		out.Reset()
		args := []string{"ctx", command, "--ticket", "task.md"}
		if command == "resume" {
			args = append(args, "--json")
		}
		status := cli.RunWithEnvironment(context.Background(), args, &out, &stderr, cli.Operations{Assemble: taskcontext.Assemble, Resume: resumption.Resume}, env)
		if status != 0 || !strings.Contains(out.String(), "Domain requirements") {
			t.Fatalf("%s status=%d report=%s stderr=%s", command, status, out.String(), stderr.String())
		}
	}
	git := exec.Command("git", "init", "--quiet", cwd)
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, output)
	}
	for _, path := range []string{".context/trial/ctx-version.txt", ".context/trial/evidence/run.md", ".context/trial/snapshots/task.md", ".context-cache/note.md", ".context/records/acceptances/current.md", ".context/config.md.lock"} {
		command := exec.Command("git", "-C", cwd, "check-ignore", "--quiet", path)
		if err := command.Run(); err != nil {
			t.Errorf("disposable path is not ignored: %s", path)
		}
	}
	for _, path := range []string{".context/records/project.md", ".agents/skills/task-context/SKILL.md", "docs/ctx/acceptance.md", ".context/config.md"} {
		command := exec.Command("git", "-C", cwd, "check-ignore", "--quiet", path)
		if err := command.Run(); err == nil {
			t.Errorf("durable path is ignored: %s", path)
		}
	}
}

func TestInitReportsConflictsWithoutOverwriting(t *testing.T) {
	env, args := initEnvironment(t)
	cwd, _ := env.WorkingDirectory()
	for _, path := range []string{".agents/skills/task-context/SKILL.md", "docs/ctx/acceptance.md", ".context/trial/ctx-version.txt"} {
		writeScopeFile(t, filepath.Join(cwd, path), "Existing content\n")
	}
	out := runInit(t, env, args, 1)
	for _, path := range []string{".agents/skills/task-context/SKILL.md", "docs/ctx/acceptance.md", ".context/trial/ctx-version.txt"} {
		data, _ := os.ReadFile(filepath.Join(cwd, path))
		if string(data) != "Existing content\n" || !strings.Contains(out, "skipped: "+filepath.Join(cwd, path)) {
			t.Fatalf("conflict not preserved/reported: %s\n%s", path, out)
		}
	}
	before := initTree(t, cwd)
	runInit(t, env, append(args, "--records", ".context/other-records"), 1)
	after := initTree(t, cwd)
	if !bytes.Equal(before[".context/config.md"], after[".context/config.md"]) || !bytes.Equal(before[".context/records/project.md"], after[".context/records/project.md"]) {
		t.Fatal("init replaced a binding or existing Project")
	}
}

func TestInitReportsRetainedLocksWhenSetupFails(t *testing.T) {
	env, args := initEnvironment(t)
	cwd, _ := env.WorkingDirectory()
	home, _ := env.HomeDirectory()
	config := filepath.Join(cwd, ".context/config.md")
	// The registry lock is created before setup encounters this invalid sidecar.
	if err := os.MkdirAll(config+".lock", 0o755); err != nil {
		t.Fatal(err)
	}
	output := runInit(t, env, args, 2)
	for _, want := range []string{
		"created: " + filepath.Join(home, ".context/config.md.lock"),
		"preserved: " + config + ".lock",
		"Binding not changed:",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("missing %q in report: %s", want, output)
		}
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatalf("failed setup created a binding: %v", err)
	}
}

func TestInitRejectsInvalidInputBeforeWriting(t *testing.T) {
	for _, extra := range [][]string{{"--tracker", "other"}, {"extra"}, {"--agent"}, {"--local-dir", "."}, {"--local-dir", ".agents"}, {"--records", ".context/trial/records"}, {"--allow-source", "missing"}} {
		t.Run(strings.Join(extra, " "), func(t *testing.T) {
			env, args := initEnvironment(t)
			cwd, _ := env.WorkingDirectory()
			before := initTree(t, cwd)
			runInit(t, env, append(args, extra...), 2)
			if !reflect.DeepEqual(before, initTree(t, cwd)) {
				t.Fatal("invalid invocation wrote project files")
			}
		})
	}
	env, _ := initEnvironment(t)
	runInit(t, env, []string{"ctx", "init"}, 2)
}

func TestInitPreservesInvalidManifestAndSymlinks(t *testing.T) {
	env, args := initEnvironment(t)
	cwd, _ := env.WorkingDirectory()
	manifest := filepath.Join(cwd, ".context/records/project.md")
	writeScopeFile(t, manifest, "# Existing local plan\n")
	target := filepath.Join(cwd, "existing-tracker.md")
	writeScopeFile(t, target, "Existing tracker authority\n")
	link := filepath.Join(cwd, "docs/ctx/tracker.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	out := runInit(t, env, args, 1)
	if !strings.Contains(out, "Binding not changed") || !strings.Contains(out, "skipped: "+link) {
		t.Fatalf("missing conflicts: %s", out)
	}
	if got, _ := os.ReadFile(manifest); string(got) != "# Existing local plan\n" {
		t.Fatal("existing manifest was changed")
	}
	if got, _ := os.Readlink(link); got != target {
		t.Fatal("existing symlink was changed")
	}
	if got, _ := os.ReadFile(target); string(got) != "Existing tracker authority\n" {
		t.Fatal("existing symlink target was changed")
	}
	if _, err := os.Stat(filepath.Join(cwd, ".context/config.md")); !os.IsNotExist(err) {
		t.Fatalf("invalid Project was bound: %v", err)
	}
}

func TestInitUsesExplicitDirectoriesAndSeparateRecords(t *testing.T) {
	env, args := initEnvironment(t)
	cwd, _ := env.WorkingDirectory()
	for i, value := range args {
		if value == ".agents/skills" {
			args[i] = ".claude/skills"
		}
		if value == "GitHub Issues" {
			args[i] = "docs/roadmap.md"
		}
	}
	writeScopeFile(t, filepath.Join(cwd, "docs/roadmap.md"), "# Existing authoritative roadmap\n")
	args = append(args, "--records", "../separate records", "--local-dir", "trial [local]")
	runInit(t, env, args, 0)
	for path, data := range initTree(t, cwd) {
		if strings.HasSuffix(path, ".md") {
			checkInitLinks(t, filepath.Join(cwd, path), data)
		}
	}
	if _, err := os.Stat(filepath.Join(cwd, ".agents")); !os.IsNotExist(err) {
		t.Fatal("init guessed another skills location")
	}
	if data, _ := os.ReadFile(filepath.Join(cwd, "docs/roadmap.md")); string(data) != "# Existing authoritative roadmap\n" {
		t.Fatal("init changed the existing roadmap")
	}
	if data, _ := os.ReadFile(filepath.Join(cwd, "../separate records/.gitignore")); string(data) != "/acceptances/\n" {
		t.Fatalf("external records ignore rules: %q", data)
	}
	if output, err := exec.Command("git", "init", "--quiet", cwd).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, output)
	}
	if err := exec.Command("git", "-C", cwd, "check-ignore", "--quiet", "trial [local]/ctx-version.txt").Run(); err != nil {
		t.Fatalf("literal local path not ignored: %v", err)
	}
}

func initTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func checkInitLinks(t *testing.T, path string, data []byte) {
	t.Helper()
	doc := parser.New().Parse(data)
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if link, ok := node.(*ast.Link); ok && entering {
			target, err := url.PathUnescape(link.Destination.Value(data))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), target)); err != nil {
				t.Errorf("broken reference in %s to %s: %v", path, target, err)
			}
		}
		return ast.WalkContinue, nil
	})
}
