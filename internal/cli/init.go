package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"github.com/Zokiio/context/internal/initialization"
	urfave "github.com/urfave/cli/v3"
)

func initCommand(stdout, stderr io.Writer, environment Environment, status *int) *urfave.Command {
	return &urfave.Command{
		Name: "init", Usage: "Prepare project files from explicit tracker and agent-directory choices",
		Writer: stderr, ErrWriter: stderr, DisableSliceFlagSeparator: true,
		ExitErrHandler: func(context.Context, *urfave.Command, error) {},
		OnUsageError:   func(_ context.Context, _ *urfave.Command, err error, _ bool) error { return err },
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "title", Required: true, Usage: "Title for a new Project manifest"},
			&urfave.StringFlag{Name: "tracker", Required: true, Usage: "Existing authoritative tracker or local files when no tracker exists"},
			&urfave.StringFlag{Name: "skills-dir", Required: true, Usage: "Selected agent tool's project skills directory"},
			&urfave.StringFlag{Name: "docs-dir", Required: true, Usage: "Directory for ctx acceptance and tracker guidance"},
			&urfave.StringFlag{Name: "records", Value: ".context/records", Usage: "Project records directory"},
			&urfave.StringFlag{Name: "local-dir", Value: ".context/trial", Usage: "Disposable trial directory inside the project"},
			&urfave.StringSliceFlag{Name: "allow-source", Usage: "Existing document directory to authorize; repeat for multiple roots"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() != 0 {
				return errors.New("init does not accept positional arguments; run it from the target project root")
			}
			for _, name := range []string{"title", "tracker", "skills-dir", "docs-dir", "records", "local-dir"} {
				if cmd.Count(name) > 1 {
					return fmt.Errorf("--%s may only be specified once", name)
				}
				if strings.TrimSpace(cmd.String(name)) == "" {
					return fmt.Errorf("--%s must be nonempty", name)
				}
			}
			cwd, err := environment.WorkingDirectory()
			if err != nil {
				return err
			}
			home, err := environment.HomeDirectory()
			if err != nil {
				return err
			}
			executable, err := environment.Executable()
			if err != nil {
				return err
			}
			var version bytes.Buffer
			info, _ := debug.ReadBuildInfo()
			if err := renderVersion(&version, info); err != nil {
				return err
			}
			result, initErr := initialization.Initialize(ctx, initialization.Request{
				Directory: cwd, Home: home, Executable: executable, Version: version.String(),
				Title: cmd.String("title"), Tracker: cmd.String("tracker"), Skills: cmd.String("skills-dir"),
				Docs: cmd.String("docs-dir"), Records: cmd.String("records"), Local: cmd.String("local-dir"),
				AllowSources: cmd.StringSlice("allow-source"),
			})
			if err := renderInitialization(stdout, result); err != nil {
				return err
			}
			if result.Incomplete() {
				*status = 1
			}
			return initErr
		},
	}
}

func renderInitialization(output io.Writer, result initialization.Result) error {
	var text bytes.Buffer
	for _, file := range result.Files {
		fmt.Fprintf(&text, "%s: %s", file.Status, file.Path)
		if file.Reason != "" {
			fmt.Fprintf(&text, " (%s)", file.Reason)
		}
		fmt.Fprintln(&text)
	}
	if result.Binding != nil {
		if err := printSetupSummary(&text, *result.Binding); err != nil {
			return err
		}
	}
	if result.BindingError != "" {
		fmt.Fprintf(&text, "Binding not changed: %s\n", result.BindingError)
	}
	if result.Pointers != "" {
		if result.Incomplete() {
			fmt.Fprintln(&text, "\nPreparation is incomplete. Review the skipped files and binding before using these pointers.")
		}
		fmt.Fprintf(&text, "\nAdd these pointers to your existing agent instructions after reviewing them:\n\n%s", result.Pointers)
		fmt.Fprintln(&text, "\nRemaining manual work: select a real task in the authoritative tracker, check its freshness, and select its source documents and allowed roots. Create an identified local snapshot only when needed, retaining its original source. Review existing instruction conflicts and any skipped files. Init has not established task readiness, acceptance, human review, hosted CI, or a tracker update.")
	}
	n, err := io.WriteString(output, text.String())
	if err == nil && n != text.Len() {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("write init output: %w", err)
	}
	return nil
}
