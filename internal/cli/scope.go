package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/taskcontext"
	urfave "github.com/urfave/cli/v3"
)

type readScope struct {
	discovery.Scope
	directBundle bool
}

func scopeFlags() []urfave.Flag {
	return []urfave.Flag{
		&urfave.IntFlag{Name: "max-files", Value: taskcontext.DefaultMaxFiles, Usage: "Maximum number of inspected source files"},
		&urfave.Int64Flag{Name: "max-bytes", Value: taskcontext.DefaultMaxBytes, Usage: "Maximum total source bytes before rendering"},
		&urfave.StringFlag{Name: "project", Usage: "Discover a project from a directory or select a personal @alias"},
		&urfave.StringFlag{Name: "workspace", Usage: "Discover a workspace from a directory or select a personal @alias"},
		&urfave.StringFlag{Name: "bundle", Usage: "Read a records directory directly, bypassing discovery configuration"},
		&urfave.StringSliceFlag{Name: "allow-source", Usage: "Additional directory allowed for linked documents; repeat for multiple directories"},
		&urfave.BoolFlag{Name: "explain-scope", Usage: "Explain the selected scope and its origins on stderr"},
	}
}

func resolveReadScope(ctx context.Context, cmd *urfave.Command, environment Environment) (readScope, error) {
	selector := ""
	for _, name := range []string{"project", "workspace", "bundle"} {
		if cmd.Count(name) > 1 {
			return readScope{}, fmt.Errorf("--%s may only be specified once", name)
		}
		if cmd.Count(name) == 0 {
			continue
		}
		if cmd.String(name) == "" {
			return readScope{}, fmt.Errorf("--%s must be nonempty", name)
		}
		if selector != "" {
			return readScope{}, errors.New("--project, --workspace, and --bundle are mutually exclusive")
		}
		selector = name
	}
	if cmd.Int("max-files") <= 0 || cmd.Int64("max-bytes") <= 0 {
		return readScope{}, errors.New("max-files and max-bytes must be positive integers")
	}
	cwd, err := environment.WorkingDirectory()
	if err != nil {
		return readScope{}, fmt.Errorf("read working directory: %w", err)
	}
	var selected readScope
	if selector == "bundle" {
		records, err := invocationDirectory(cwd, cmd.String("bundle"), "bundle")
		if err != nil {
			return readScope{}, err
		}
		selected = readScope{Scope: discovery.Scope{
			Kind: discovery.Project, StartDirectory: cwd, BindingDirectory: records,
			Project: &discovery.ResolvedProject{Records: records, AllowSources: []string{}},
		}, directBundle: true}
	} else {
		home, err := environment.HomeDirectory()
		if err != nil {
			return readScope{}, fmt.Errorf("read home directory for discovery: %w; use --bundle PATH for direct records access", err)
		}
		request := discovery.Request{Cwd: cwd, Home: home}
		if selector != "" {
			request.Kind, request.Selector = discovery.Kind(selector), cmd.String(selector)
		}
		selected.Scope, err = discovery.Resolve(ctx, request)
		if err != nil {
			return readScope{}, err
		}
	}
	for _, directory := range cmd.StringSlice("allow-source") {
		if directory == "" {
			return readScope{}, errors.New("allowed source directory must be nonempty")
		}
		if selected.Project != nil {
			resolved, err := invocationDirectory(cwd, directory, "allowed source")
			if err != nil {
				return readScope{}, err
			}
			selected.Project.AllowSources = append(selected.Project.AllowSources, resolved)
		}
	}
	return selected, nil
}

func invocationDirectory(cwd, entered, label string) (string, error) {
	path := entered
	if !filepath.IsAbs(path) {
		path = cwd + string(filepath.Separator) + path
	}
	fail := func(err error) (string, error) {
		return "", fmt.Errorf("%s directory %q from cwd %q resolves to %q: %w; choose an accessible directory", label, entered, cwd, path, err)
	}
	// Validate the entered traversal before canonicalizing it. Cleaning first can
	// silently turn missing/../records into an available, different location.
	directory, err := os.Open(path)
	if err != nil {
		return fail(err)
	}
	defer directory.Close()
	info, err := directory.Stat()
	if err != nil {
		return fail(err)
	}
	if !info.IsDir() {
		return fail(errors.New("path is not a directory"))
	}
	if _, err := directory.ReadDir(1); err != nil && !errors.Is(err, io.EOF) {
		return fail(err)
	}
	canonical, err := discovery.CanonicalPath(path)
	if err != nil {
		return fail(err)
	}
	return canonical, nil
}

func explainReadScope(output io.Writer, scope readScope) error {
	var text strings.Builder
	fmt.Fprintf(&text, "Scope: %s\n  binding directory: %s\n", scope.Kind, scope.BindingDirectory)
	if scope.Project != nil {
		fmt.Fprintf(&text, "  records: %s\n", scope.Project.Records)
	}
	if scope.Workspace != nil {
		fmt.Fprintf(&text, "  workspace: %s [%s]\n", scope.Workspace.Title, scope.Workspace.ID)
	}
	if scope.directBundle {
		fmt.Fprintln(&text, "  origin: explicit --bundle; discovery bypassed")
	}
	for _, origin := range scope.Origins {
		fmt.Fprintf(&text, "  origin: %s field %s\n", origin.Path, origin.Field)
	}
	if _, err := io.WriteString(output, text.String()); err != nil {
		return fmt.Errorf("write scope explanation: %w", err)
	}
	return nil
}

func projectSelectionRequired(command string, scope readScope) error {
	return fmt.Errorf("%s requires project scope; selected workspace %q [%s] at %q; select a project with --project PATH or --project @alias, or use --bundle PATH for direct records access", command, scope.Workspace.Title, scope.Workspace.ID, scope.BindingDirectory)
}
