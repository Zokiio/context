// Package cli owns command-line input, report output, and process exit status.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
	urfave "github.com/urfave/cli/v3"
)

type AssembleFunc func(context.Context, taskcontext.Request) (taskcontext.Result, error)
type OrientFunc func(context.Context, orientation.Request) (orientation.Result, error)

// Operations are the application boundaries invoked by the CLI.
type Operations struct {
	Assemble AssembleFunc
	Orient   OrientFunc
}

// Run renders one report, or an invocation/execution error on stderr.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, operations Operations) int {
	return RunWithEnvironment(ctx, args, stdout, stderr, operations, Environment{})
}

// RunWithEnvironment exposes the same command behavior with an injected caller
// environment. Getters are evaluated only by commands that need their values.
func RunWithEnvironment(ctx context.Context, args []string, stdout, stderr io.Writer, operations Operations, environment Environment) int {
	environment = environment.defaults()
	status := 0
	usageError := func(_ context.Context, _ *urfave.Command, err error, _ bool) error { return err }
	exitHandler := func(context.Context, *urfave.Command, error) {}
	command := &urfave.Command{
		Name: "ctx", Usage: "Read project orientation and ticket context", Writer: stderr, ErrWriter: stderr,
		ExitErrHandler: exitHandler, OnUsageError: usageError,
		Action: func(context.Context, *urfave.Command) error {
			return errors.New("expected context or orient subcommand")
		},
		Commands: []*urfave.Command{{
			DisableSliceFlagSeparator: true,
			Name:                      "context", Usage: "Read a ticket from the selected project",
			Writer: stderr, ErrWriter: stderr, ExitErrHandler: exitHandler, OnUsageError: usageError,
			Flags: append(scopeFlags(),
				&urfave.StringFlag{Name: "ticket", Required: true, Usage: "Ticket path relative to the project"},
			),
			Action: func(ctx context.Context, cmd *urfave.Command) error {
				if cmd.NArg() != 0 {
					return errors.New("context does not accept positional arguments")
				}
				if cmd.String("ticket") == "" {
					return errors.New("ticket must be nonempty")
				}
				scope, err := resolveReadScope(ctx, cmd, environment)
				if err != nil {
					return err
				}
				if cmd.Bool("explain-scope") {
					if err := explainReadScope(stderr, scope); err != nil {
						return err
					}
				}
				if scope.Kind == discovery.Workspace {
					return projectSelectionRequired("context", scope)
				}
				if operations.Assemble == nil {
					return errors.New("context operation is unavailable")
				}
				result, err := operations.Assemble(ctx, taskcontext.Request{ProjectDir: scope.Project.Records, TicketPath: cmd.String("ticket"), AllowedSourceDirs: scope.Project.AllowSources, MaxFiles: cmd.Int("max-files"), MaxBytes: cmd.Int64("max-bytes")})
				if err != nil {
					return err
				}
				if err := json.NewEncoder(stdout).Encode(result); err != nil {
					return fmt.Errorf("write context JSON: %w", err)
				}
				if !result.Complete {
					status = 1
				}
				return nil
			},
		}, {
			DisableSliceFlagSeparator: true,
			Name:                      "orient", Usage: "Inspect the selected project or workspace",
			Writer: stderr, ErrWriter: stderr, ExitErrHandler: exitHandler, OnUsageError: usageError,
			Flags: append(scopeFlags(), &urfave.BoolFlag{Name: "json", Usage: "Write the versioned JSON orientation report"}),
			Action: func(ctx context.Context, cmd *urfave.Command) error {
				for _, name := range []string{"max-files", "max-bytes", "json"} {
					if cmd.Count(name) > 1 {
						return fmt.Errorf("--%s may only be specified once", name)
					}
				}
				if cmd.NArg() != 0 {
					return errors.New("orient does not accept positional arguments")
				}
				scope, err := resolveReadScope(ctx, cmd, environment)
				if err != nil {
					return err
				}
				if cmd.Bool("explain-scope") {
					if err := explainReadScope(stderr, scope); err != nil {
						return err
					}
				}
				if scope.Kind == discovery.Workspace {
					return runWorkspaceOrientation(ctx, stdout, *scope.Workspace, cmd.Bool("json"))
				}
				if operations.Orient == nil {
					return errors.New("orientation operation is unavailable")
				}
				result, err := operations.Orient(ctx, orientation.Request{ProjectDir: scope.Project.Records, AllowedSourceDirs: scope.Project.AllowSources, MaxFiles: cmd.Int("max-files"), MaxBytes: cmd.Int64("max-bytes")})
				if err != nil {
					return err
				}
				if cmd.Bool("json") {
					if err := json.NewEncoder(stdout).Encode(result); err != nil {
						return fmt.Errorf("write orientation JSON: %w", err)
					}
				} else if err := renderOrientation(stdout, result); err != nil {
					return fmt.Errorf("write orientation text: %w", err)
				}
				if !result.Complete {
					status = 1
				}
				return nil
			},
		}},
	}
	if err := command.Run(ctx, args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return status
}
