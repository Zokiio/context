// Package cli owns command-line input, report output, and process exit status.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

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
			Name:                      "context", Usage: "Read a ticket from an explicit project bundle",
			Writer: stderr, ErrWriter: stderr, ExitErrHandler: exitHandler, OnUsageError: usageError,
			Flags: append(scopeFlags(),
				&urfave.StringFlag{Name: "ticket", Required: true, Usage: "Ticket path relative to the project"},
			),
			Action: func(ctx context.Context, cmd *urfave.Command) error {
				if cmd.NArg() != 0 {
					return errors.New("context does not accept positional arguments")
				}
				if cmd.String("project") == "" || cmd.String("ticket") == "" {
					return errors.New("project and ticket must be nonempty")
				}
				project, allowed, err := resolveScope(cmd)
				if err != nil {
					return err
				}
				if operations.Assemble == nil {
					return errors.New("context operation is unavailable")
				}
				result, err := operations.Assemble(ctx, taskcontext.Request{ProjectDir: project, TicketPath: cmd.String("ticket"), AllowedSourceDirs: allowed, MaxFiles: cmd.Int("max-files"), MaxBytes: cmd.Int64("max-bytes")})
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
			Name:                      "orient", Usage: "Inspect goals, commitments, and work in an explicit project bundle",
			Writer: stderr, ErrWriter: stderr, ExitErrHandler: exitHandler, OnUsageError: usageError,
			Flags: append(scopeFlags(), &urfave.BoolFlag{Name: "json", Usage: "Write the versioned JSON orientation report"}),
			Action: func(ctx context.Context, cmd *urfave.Command) error {
				for _, name := range []string{"project", "max-files", "max-bytes", "json"} {
					if cmd.Count(name) > 1 {
						return fmt.Errorf("--%s may only be specified once", name)
					}
				}
				if cmd.NArg() != 0 {
					return errors.New("orient does not accept positional arguments")
				}
				project, allowed, err := resolveScope(cmd)
				if err != nil {
					return err
				}
				if operations.Orient == nil {
					return errors.New("orientation operation is unavailable")
				}
				result, err := operations.Orient(ctx, orientation.Request{ProjectDir: project, AllowedSourceDirs: allowed, MaxFiles: cmd.Int("max-files"), MaxBytes: cmd.Int64("max-bytes")})
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

func scopeFlags() []urfave.Flag {
	return []urfave.Flag{
		&urfave.IntFlag{Name: "max-files", Value: taskcontext.DefaultMaxFiles, Usage: "Maximum number of inspected source files"},
		&urfave.Int64Flag{Name: "max-bytes", Value: taskcontext.DefaultMaxBytes, Usage: "Maximum total source bytes before rendering"},
		&urfave.StringFlag{Name: "project", Required: true, Usage: "Project bundle directory"},
		&urfave.StringSliceFlag{Name: "allow-source", Usage: "Additional directory allowed for linked documents; repeat for multiple directories"},
	}
}

func resolveScope(cmd *urfave.Command) (string, []string, error) {
	if cmd.String("project") == "" {
		return "", nil, errors.New("project must be nonempty")
	}
	if cmd.Int("max-files") <= 0 || cmd.Int64("max-bytes") <= 0 {
		return "", nil, errors.New("max-files and max-bytes must be positive integers")
	}
	project, err := filepath.Abs(cmd.String("project"))
	if err != nil {
		return "", nil, fmt.Errorf("resolve project: %w", err)
	}
	allowed := cmd.StringSlice("allow-source")
	for index, directory := range allowed {
		if directory == "" {
			return "", nil, errors.New("allowed source directory must be nonempty")
		}
		absolute, err := filepath.Abs(directory)
		if err != nil {
			return "", nil, fmt.Errorf("resolve allowed source: %w", err)
		}
		allowed[index] = absolute
	}
	return project, allowed, nil
}
