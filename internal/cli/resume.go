package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/resumption"
	urfave "github.com/urfave/cli/v3"
)

type ResumeFunc func(context.Context, resumption.Request) (resumption.Result, error)

func resumeCommand(stdout, stderr io.Writer, operation ResumeFunc, environment Environment, status *int) *urfave.Command {
	flags := append(scopeFlags(),
		&urfave.StringFlag{Name: "ticket", Required: true, Usage: "Ticket path relative to the project"},
		&urfave.StringFlag{Name: "checkout", Usage: "Working directory whose recovery cache is inspected"},
		&urfave.BoolFlag{Name: "json", Usage: "Write the versioned JSON task-resumption report"},
		&urfave.IntFlag{Name: "max-cache-files", Value: resumption.DefaultMaxCacheFiles, Usage: "Maximum recovery cache entries inspected"},
		&urfave.Int64Flag{Name: "max-cache-bytes", Value: resumption.DefaultMaxCacheBytes, Usage: "Maximum recovery cache bytes read"},
	)
	return &urfave.Command{
		DisableSliceFlagSeparator: true,
		Name:                      "resume",
		Usage:                     "Refresh a task and inspect its recovery state",
		Writer:                    stderr,
		ErrWriter:                 stderr,
		Flags:                     flags,
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			for _, name := range []string{"ticket", "project", "workspace", "bundle", "checkout", "json", "explain-scope", "max-files", "max-bytes", "max-cache-files", "max-cache-bytes"} {
				if cmd.Count(name) > 1 {
					return fmt.Errorf("--%s may only be specified once", name)
				}
			}
			if cmd.NArg() != 0 {
				return errors.New("resume does not accept positional arguments")
			}
			if cmd.String("ticket") == "" {
				return errors.New("ticket must be nonempty")
			}
			if cmd.Count("checkout") > 0 && cmd.String("checkout") == "" {
				return errors.New("checkout must be nonempty")
			}
			if cmd.Int("max-cache-files") <= 0 || cmd.Int64("max-cache-bytes") <= 0 {
				return errors.New("max-cache-files and max-cache-bytes must be positive integers")
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
				return projectSelectionRequired("resume", scope)
			}
			working, err := resolveResumeCheckout(cmd, scope)
			if err != nil {
				return err
			}
			if operation == nil {
				return errors.New("resumption operation is unavailable")
			}
			result, err := operation(ctx, resumption.Request{
				RecordsDirectory: scope.Project.Records, WorkingDirectory: working,
				TicketPath: cmd.String("ticket"), AllowedSourceDirs: scope.Project.AllowSources,
				MaxFiles: cmd.Int("max-files"), MaxBytes: cmd.Int64("max-bytes"),
				MaxCacheFiles: cmd.Int("max-cache-files"), MaxCacheBytes: cmd.Int64("max-cache-bytes"),
			})
			if err != nil {
				return err
			}
			if cmd.Bool("json") {
				if err := json.NewEncoder(stdout).Encode(result); err != nil {
					return fmt.Errorf("write resumption JSON: %w", err)
				}
			} else if err := renderResumption(stdout, result); err != nil {
				return fmt.Errorf("write resumption text: %w", err)
			}
			if !result.Complete {
				*status = 1
			}
			return nil
		},
	}
}

func resolveResumeCheckout(cmd *urfave.Command, scope readScope) (string, error) {
	if cmd.Count("checkout") > 0 {
		entered := cmd.String("checkout")
		if !filepath.IsAbs(entered) && scope.invocationCwd == "" {
			return "", fmt.Errorf("read working directory: %w; use an absolute --checkout path", scope.cwdError)
		}
		return invocationDirectory(scope.invocationCwd, entered, "checkout")
	}
	if scope.directBundle {
		return "", errors.New("resume with --bundle requires --checkout PATH")
	}
	return invocationDirectory("", scope.BindingDirectory, "selected working")
}

func renderResumption(output io.Writer, result resumption.Result) error {
	var text strings.Builder
	fmt.Fprintln(&text, "Task resumption")
	fmt.Fprintf(&text, "  records: %s\n  working directory: %s\n  cache: %s\n", result.Scope.RecordsDirectory, result.Scope.WorkingDirectory, result.Scope.CacheRoot)
	fmt.Fprintf(&text, "  project: %s\n  task: %s\n", knownString(result.Scope.ProjectID), knownString(result.Scope.TaskID))
	fmt.Fprintf(&text, "  evaluation: %s; task context: %s; project inventory: %s\n", completeness(result.Complete), completeness(result.Context.Complete), completeness(result.Orientation.InventoryComplete))
	fmt.Fprintf(&text, "Recovery: %s; graph: %s; inventory: %s\n", result.Recovery.Status, result.Recovery.GraphStatus, completeness(result.Recovery.InventoryComplete))
	if result.Comparison.BaselineAvailable {
		fmt.Fprintln(&text, "Baseline: available")
	} else {
		fmt.Fprintln(&text, "Baseline: unavailable; no claim is made that current sources are unchanged.")
	}
	for _, observation := range result.Recovery.Observations {
		fmt.Fprintf(&text, "Historical observation %s by %s at %s\n  note: %s\n  retained context: %s [%s]\n", observation.ID, observation.Actor, observation.ObservedAt, observation.Source.Path, observation.Snapshot.Path, observation.Snapshot.Status)
		fmt.Fprintln(&text, "Reported progress and checks; completion not established by this note:")
		text.WriteString(observation.Body)
		if !strings.HasSuffix(observation.Body, "\n") {
			text.WriteByte('\n')
		}
	}
	for _, comparison := range result.Comparison.Candidates {
		fmt.Fprintf(&text, "Source text comparison for %s: %s\n", comparison.ObservationID, completeness(comparison.Complete))
		for _, source := range comparison.Sources {
			fmt.Fprintf(&text, "  %s\n", source.Status)
			if source.Previous != nil {
				fmt.Fprintf(&text, "    previous: %s [%s]\n", source.Previous.Path, source.Previous.Availability)
			}
			if source.Current != nil {
				fmt.Fprintf(&text, "    current: %s [%s]\n", source.Current.Path, source.Current.Availability)
			}
		}
		if !comparison.Complete {
			fmt.Fprintln(&text, "  Comparison is incomplete; inspect diagnostics and current context.")
		}
	}
	if task := selectedTask(result); task != nil {
		fmt.Fprintf(&text, "Current task: %s [%s]\n  source: %s\n  execution: %s; readiness: %s\n", knownString(task.Title), knownString(task.ID), task.Source, knownString(task.Execution), task.Readiness)
		fmt.Fprintln(&text, "  Current checks:")
		for _, check := range task.Checks {
			fmt.Fprintf(&text, "    %s: %s\n", check.Name, check.Status)
			for _, reason := range check.Reasons {
				writeFinding(&text, "      ", reason)
			}
		}
		writeAcceptance(&text, task.Acceptance)
	}
	fmt.Fprintln(&text, "Current context sources:")
	if len(result.Context.Sources) == 0 {
		fmt.Fprintln(&text, "  none observed")
	}
	for _, source := range result.Context.Sources {
		fmt.Fprintf(&text, "  %s\n    sha256: %s\n", source.Path, source.SHA256)
		for _, reason := range source.Reasons {
			fmt.Fprintf(&text, "    %s%s\n", reason.Kind, relationshipDetails(reason.From, reason.Link))
		}
	}
	fmt.Fprintln(&text, "Diagnostics:")
	if len(result.Diagnostics) == 0 {
		fmt.Fprintln(&text, "  none")
	}
	for _, diagnostic := range result.Diagnostics {
		fmt.Fprintf(&text, "  %s %s: %s", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
		if diagnostic.Path != nil {
			fmt.Fprintf(&text, " [%s]", *diagnostic.Path)
		}
		fmt.Fprintln(&text)
	}
	_, err := io.WriteString(output, text.String())
	return err
}

func selectedTask(result resumption.Result) *orientation.WorkItem {
	if result.Scope.TaskID == nil {
		return nil
	}
	for index := range result.Orientation.WorkItems {
		work := &result.Orientation.WorkItems[index]
		if work.ID != nil && strings.TrimSpace(*work.ID) == *result.Scope.TaskID && !work.IdentityAmbiguous {
			return work
		}
	}
	return nil
}
