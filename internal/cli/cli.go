// Package cli owns command-line input, JSON output, and process exit status.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Zokiio/context/internal/taskcontext"
	urfave "github.com/urfave/cli/v3"
)

type AssembleFunc func(context.Context, taskcontext.Request) (taskcontext.Result, error)

// Run renders one context result, or an invocation/execution error on stderr.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, assemble AssembleFunc) int {
	status := 0
	usageError := func(_ context.Context, _ *urfave.Command, err error, _ bool) error { return err }
	exitHandler := func(context.Context, *urfave.Command, error) {}
	command := &urfave.Command{
		Name: "ctx", Usage: "Read current ticket context", Writer: stderr, ErrWriter: stderr,
		ExitErrHandler: exitHandler, OnUsageError: usageError,
		Action: func(context.Context, *urfave.Command) error { return errors.New("expected context subcommand") },
		Commands: []*urfave.Command{{
			DisableSliceFlagSeparator: true,
			Name:                      "context", Usage: "Read a ticket from an explicit project bundle",
			Writer: stderr, ErrWriter: stderr, ExitErrHandler: exitHandler, OnUsageError: usageError,
			Flags: []urfave.Flag{
				&urfave.IntFlag{Name: "max-files", Value: taskcontext.DefaultMaxFiles, Usage: "Maximum number of included source files"},
				&urfave.Int64Flag{Name: "max-bytes", Value: taskcontext.DefaultMaxBytes, Usage: "Maximum total source bytes before JSON encoding"},
				&urfave.StringFlag{Name: "project", Required: true, Usage: "Project bundle directory"},
				&urfave.StringFlag{Name: "ticket", Required: true, Usage: "Ticket path relative to the project"},
				&urfave.StringSliceFlag{Name: "allow-source", Usage: "Additional directory allowed for linked documents; repeat for multiple directories"},
			},
			Action: func(ctx context.Context, cmd *urfave.Command) error {
				if cmd.NArg() != 0 {
					return errors.New("context does not accept positional arguments")
				}
				if cmd.String("project") == "" || cmd.String("ticket") == "" {
					return errors.New("project and ticket must be nonempty")
				}
				if cmd.Int("max-files") <= 0 || cmd.Int64("max-bytes") <= 0 {
					return errors.New("max-files and max-bytes must be positive integers")
				}
				project, err := filepath.Abs(cmd.String("project"))
				if err != nil {
					return fmt.Errorf("resolve project: %w", err)
				}
				allowed := cmd.StringSlice("allow-source")
				for index, directory := range allowed {
					if directory == "" {
						return errors.New("allowed source directory must be nonempty")
					}
					absolute, err := filepath.Abs(directory)
					if err != nil {
						return fmt.Errorf("resolve allowed source: %w", err)
					}
					allowed[index] = absolute
				}
				result, err := assemble(ctx, taskcontext.Request{ProjectDir: project, TicketPath: cmd.String("ticket"), AllowedSourceDirs: allowed, MaxFiles: cmd.Int("max-files"), MaxBytes: cmd.Int64("max-bytes")})
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
		}},
	}
	if err := command.Run(ctx, args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return status
}
